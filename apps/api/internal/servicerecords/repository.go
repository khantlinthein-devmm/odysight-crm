package servicerecords

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/odysight/crm/pkg/pagination"
)

var ErrNotFound = errors.New("service record not found")

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const recordColumns = `id, booking_number, cleaner_name, service_type, rating, status, notes, completed_at, created_at`

func scanRecord(row pgx.Row) (ServiceRecord, error) {
	var r ServiceRecord
	err := row.Scan(&r.ID, &r.BookingNumber, &r.CleanerName, &r.ServiceType,
		&r.Rating, &r.Status, &r.Notes, &r.CompletedAt, &r.CreatedAt)
	return r, err
}

func (r *Repository) List(ctx context.Context, params pagination.Params) ([]ServiceRecord, int, error) {
	args := []any{}
	conds := []string{}
	if params.Search != "" {
		like := "%" + params.Search + "%"
		start := len(args) + 1
		orParts := []string{}
		for i, col := range []string{"booking_number", "cleaner_name"} {
			orParts = append(orParts, col+" ILIKE $"+itoa(start+i))
			args = append(args, like)
			_ = i
		}
		conds = append(conds, "("+joinOr(orParts)+")")
	}
	if params.Status != "" {
		args = append(args, params.Status)
		conds = append(conds, "status = $"+itoa(len(args)))
	}
	where := ""
	if len(conds) > 0 {
		where = "WHERE " + joinAnd(conds)
	}
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM service_records `+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count service_records: %w", err)
	}
	args = append(args, params.Limit, params.Offset)
	query := `SELECT `+recordColumns+` FROM service_records ` + where + ` ORDER BY created_at DESC LIMIT $` + itoa(len(args)-1) + ` OFFSET $` + itoa(len(args))
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query service_records: %w", err)
	}
	defer rows.Close()

	items := []ServiceRecord{}
	for rows.Next() {
		item, err := scanRecord(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan service_record: %w", err)
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func itoa(i int) string { return fmt.Sprintf("%d", i) }
func joinOr(parts []string) string { return joinWith(parts, " OR ") }
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

func (r *Repository) GetByID(ctx context.Context, id int64) (ServiceRecord, error) {
	rec, err := scanRecord(r.pool.QueryRow(ctx,
		`SELECT `+recordColumns+` FROM service_records WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return ServiceRecord{}, ErrNotFound
	}
	if err != nil {
		return ServiceRecord{}, fmt.Errorf("get service record %d: %w", id, err)
	}
	return rec, nil
}

func (r *Repository) Create(ctx context.Context, rec ServiceRecord) (ServiceRecord, error) {
	created, err := scanRecord(r.pool.QueryRow(ctx,
		`INSERT INTO service_records (booking_number, cleaner_name, service_type, rating, status, notes)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING `+recordColumns,
		rec.BookingNumber, rec.CleanerName, rec.ServiceType, rec.Rating, rec.Status, rec.Notes))
	if err != nil {
		return ServiceRecord{}, fmt.Errorf("create service record: %w", err)
	}
	return created, nil
}

type Patch struct {
	BookingNumber *string
	CleanerName   *string
	ServiceType   *string
	Rating        *int
	Status        *Status
	Notes         *string
}

func (r *Repository) Update(ctx context.Context, id int64, p Patch) (ServiceRecord, error) {
	var status any
	if p.Status != nil {
		status = *p.Status
	}

	updated, err := scanRecord(r.pool.QueryRow(ctx,
		`UPDATE service_records SET
			booking_number = COALESCE($2, booking_number),
			cleaner_name   = COALESCE($3, cleaner_name),
			service_type   = COALESCE($4, service_type),
			rating         = COALESCE($5, rating),
			status         = COALESCE($6, status),
			notes          = COALESCE($7, notes),
			completed_at   = CASE WHEN $6::text = 'completed' AND completed_at IS NULL THEN now() ELSE completed_at END
		 WHERE id = $1
		 RETURNING `+recordColumns,
		id, p.BookingNumber, p.CleanerName, p.ServiceType, p.Rating, status, p.Notes))
	if errors.Is(err, pgx.ErrNoRows) {
		return ServiceRecord{}, ErrNotFound
	}
	if err != nil {
		return ServiceRecord{}, fmt.Errorf("update service record %d: %w", id, err)
	}
	return updated, nil
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM service_records WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete service record %d: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
