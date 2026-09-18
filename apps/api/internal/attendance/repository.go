package attendance

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/odysight/crm/pkg/pagination"
)

var (
	ErrNotFound          = errors.New("attendance record not found")
	ErrAlreadyCheckedIn  = errors.New("already checked in")
	ErrAlreadyCheckedOut = errors.New("already checked out")
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const attendanceColumns = `a.id, a.cleaner_id,
	COALESCE(c.first_name || ' ' || c.last_name, ''),
	a.work_date, a.check_in_at, a.check_out_at, COALESCE(a.note, ''), a.created_at`

const attendanceFrom = `FROM attendance a JOIN cleaners c ON c.id = a.cleaner_id`

func scanRecord(row pgx.Row) (Record, error) {
	var rec Record
	var workDate time.Time
	err := row.Scan(&rec.ID, &rec.CleanerID, &rec.CleanerName, &workDate,
		&rec.CheckInAt, &rec.CheckOutAt, &rec.Note, &rec.CreatedAt)
	if err != nil {
		return Record{}, err
	}
	rec.WorkDate = workDate.Format("2006-01-02")
	return rec, nil
}

func (r *Repository) List(ctx context.Context, cleanerID int64, from, to string, params pagination.Params) ([]Record, int, error) {
	args := []any{}
	conds := []string{}
	if cleanerID > 0 {
		args = append(args, cleanerID)
		conds = append(conds, "a.cleaner_id = $"+itoa(len(args)))
	}
	if from != "" {
		args = append(args, from)
		conds = append(conds, "a.work_date >= $"+itoa(len(args))+"::date")
	}
	if to != "" {
		args = append(args, to)
		conds = append(conds, "a.work_date <= $"+itoa(len(args))+"::date")
	}
	if params.Search != "" {
		like := "%" + params.Search + "%"
		args = append(args, like)
		conds = append(conds, "(c.first_name || ' ' || c.last_name ILIKE $"+itoa(len(args))+")")
	}
	where := ""
	if len(conds) > 0 {
		where = "WHERE " + joinAnd(conds)
	}
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) `+attendanceFrom+` `+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count attendance: %w", err)
	}
	args = append(args, params.Limit, params.Offset)
	query := `SELECT ` + attendanceColumns + ` ` + attendanceFrom + ` ` + where +
		` ORDER BY a.work_date DESC, a.id DESC LIMIT $` + itoa(len(args)-1) + ` OFFSET $` + itoa(len(args))
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query attendance: %w", err)
	}
	defer rows.Close()

	items := []Record{}
	for rows.Next() {
		item, err := scanRecord(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan attendance: %w", err)
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

// FindToday returns the record for one cleaner on one YYYY-MM-DD date.
func (r *Repository) FindToday(ctx context.Context, cleanerID int64, date string) (Record, error) {
	rec, err := scanRecord(r.pool.QueryRow(ctx,
		`SELECT `+attendanceColumns+` `+attendanceFrom+` WHERE a.cleaner_id = $1 AND a.work_date = $2::date`,
		cleanerID, date))
	if errors.Is(err, pgx.ErrNoRows) {
		return Record{}, ErrNotFound
	}
	if err != nil {
		return Record{}, fmt.Errorf("get attendance for cleaner %d on %s: %w", cleanerID, date, err)
	}
	return rec, nil
}

func (r *Repository) getByID(ctx context.Context, id int64) (Record, error) {
	rec, err := scanRecord(r.pool.QueryRow(ctx,
		`SELECT `+attendanceColumns+` `+attendanceFrom+` WHERE a.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Record{}, ErrNotFound
	}
	if err != nil {
		return Record{}, fmt.Errorf("get attendance %d: %w", id, err)
	}
	return rec, nil
}

// CheckIn stamps check_in_at for one cleaner on one YYYY-MM-DD date. A row
// that already has check_in_at set yields ErrAlreadyCheckedIn; otherwise the
// row is upserted so a checkout-first row is completed rather than rejected.
func (r *Repository) CheckIn(ctx context.Context, cleanerID int64, date string) (Record, error) {
	var id int64
	var checkInAt *time.Time
	err := r.pool.QueryRow(ctx,
		`SELECT id, check_in_at FROM attendance WHERE cleaner_id = $1 AND work_date = $2::date`,
		cleanerID, date).Scan(&id, &checkInAt)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return Record{}, fmt.Errorf("lookup attendance for cleaner %d on %s: %w", cleanerID, date, err)
	}
	if err == nil && checkInAt != nil {
		return Record{}, ErrAlreadyCheckedIn
	}
	err = r.pool.QueryRow(ctx,
		`INSERT INTO attendance (cleaner_id, work_date, check_in_at)
		 VALUES ($1, $2::date, now())
		 ON CONFLICT (cleaner_id, work_date)
		 DO UPDATE SET check_in_at = now(), updated_at = now()
		 RETURNING id`,
		cleanerID, date).Scan(&id)
	if err != nil {
		return Record{}, fmt.Errorf("check in cleaner %d on %s: %w", cleanerID, date, err)
	}
	return r.getByID(ctx, id)
}

// CheckOut stamps check_out_at for one cleaner on one YYYY-MM-DD date. With
// no row yet (checkin-first flow missing), it inserts a checkout-only row; a
// row that already has check_out_at set yields ErrAlreadyCheckedOut.
func (r *Repository) CheckOut(ctx context.Context, cleanerID int64, date string) (Record, error) {
	var id int64
	var checkOutAt *time.Time
	err := r.pool.QueryRow(ctx,
		`SELECT id, check_out_at FROM attendance WHERE cleaner_id = $1 AND work_date = $2::date`,
		cleanerID, date).Scan(&id, &checkOutAt)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return Record{}, fmt.Errorf("lookup attendance for cleaner %d on %s: %w", cleanerID, date, err)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		err = r.pool.QueryRow(ctx,
			`INSERT INTO attendance (cleaner_id, work_date, check_out_at)
			 VALUES ($1, $2::date, now())
			 RETURNING id`,
			cleanerID, date).Scan(&id)
		if err != nil {
			return Record{}, fmt.Errorf("check out cleaner %d on %s: %w", cleanerID, date, err)
		}
		return r.getByID(ctx, id)
	}
	if checkOutAt != nil {
		return Record{}, ErrAlreadyCheckedOut
	}
	if _, err := r.pool.Exec(ctx,
		`UPDATE attendance SET check_out_at = now(), updated_at = now() WHERE id = $1`, id); err != nil {
		return Record{}, fmt.Errorf("check out attendance %d: %w", id, err)
	}
	return r.getByID(ctx, id)
}

func itoa(i int) string             { return fmt.Sprintf("%d", i) }
func joinAnd(parts []string) string { return joinWith(parts, " AND ") }
func joinWith(parts []string, sep string) string {
	out := ""
	for i, s := range parts {
		if i > 0 {
			out += sep
		}
		out += s
	}
	return out
}
