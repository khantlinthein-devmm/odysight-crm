package cleaners

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/odysight/crm/pkg/pagination"
)

var ErrNotFound = errors.New("cleaner not found")

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const cleanerColumns = `id, first_name, last_name, phone, email, skills, status, area, user_id, lat, lng, is_online, last_seen_at, created_at`

func scanCleaner(row pgx.Row) (Cleaner, error) {
	var c Cleaner
	err := row.Scan(&c.ID, &c.FirstName, &c.LastName, &c.Phone, &c.Email, &c.Skills, &c.Status,
		&c.Area, &c.UserID, &c.Lat, &c.Lng, &c.IsOnline, &c.LastSeenAt, &c.CreatedAt)
	return c, err
}

func (r *Repository) List(ctx context.Context, params pagination.Params) ([]Cleaner, int, error) {
	args := []any{}
	conds := []string{}
	if params.Search != "" {
		like := "%" + params.Search + "%"
		start := len(args) + 1
		orParts := []string{}
		for i, col := range []string{"first_name", "last_name", "email", "phone"} {
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
	if params.Area != "" {
		args = append(args, params.Area)
		conds = append(conds, "lower(area) = lower($"+itoa(len(args))+")")
	}
	if params.OnlineOnly {
		conds = append(conds, "is_online = true")
	}
	where := ""
	if len(conds) > 0 {
		where = "WHERE " + joinAnd(conds)
	}
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM cleaners `+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count cleaners: %w", err)
	}
	args = append(args, params.Limit, params.Offset)
	query := `SELECT `+cleanerColumns+` FROM cleaners ` + where + ` ORDER BY created_at DESC LIMIT $` + itoa(len(args)-1) + ` OFFSET $` + itoa(len(args))
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query cleaners: %w", err)
	}
	defer rows.Close()

	items := []Cleaner{}
	for rows.Next() {
		item, err := scanCleaner(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan cleaner: %w", err)
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

func (r *Repository) GetByID(ctx context.Context, id int64) (Cleaner, error) {
	c, err := scanCleaner(r.pool.QueryRow(ctx,
		`SELECT `+cleanerColumns+` FROM cleaners WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Cleaner{}, ErrNotFound
	}
	if err != nil {
		return Cleaner{}, fmt.Errorf("get cleaner %d: %w", id, err)
	}
	return c, nil
}

func (r *Repository) Create(ctx context.Context, c Cleaner) (Cleaner, error) {
	created, err := scanCleaner(r.pool.QueryRow(ctx,
		`INSERT INTO cleaners (first_name, last_name, phone, email, skills, status, area, user_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING `+cleanerColumns,
		c.FirstName, c.LastName, c.Phone, c.Email, c.Skills, c.Status, c.Area, c.UserID))
	if err != nil {
		return Cleaner{}, fmt.Errorf("create cleaner: %w", err)
	}
	return created, nil
}

type Patch struct {
	FirstName *string
	LastName  *string
	Phone     *string
	Email     *string
	Skills    *string
	Status    *Status
	Area      *string
	UserID    *int64
	IsOnline  *bool
}

func (r *Repository) Update(ctx context.Context, id int64, p Patch) (Cleaner, error) {
	var status any
	if p.Status != nil {
		status = *p.Status
	}

	updated, err := scanCleaner(r.pool.QueryRow(ctx,
		`UPDATE cleaners SET
			first_name = COALESCE($2, first_name),
			last_name  = COALESCE($3, last_name),
			phone      = COALESCE($4, phone),
			email      = COALESCE($5, email),
			skills     = COALESCE($6, skills),
			status     = COALESCE($7, status),
			area       = COALESCE($8, area),
			user_id    = COALESCE($9, user_id),
			is_online  = COALESCE($10, is_online)
		 WHERE id = $1
		 RETURNING `+cleanerColumns,
		id, p.FirstName, p.LastName, p.Phone, p.Email, p.Skills, status, p.Area, p.UserID, p.IsOnline))
	if errors.Is(err, pgx.ErrNoRows) {
		return Cleaner{}, ErrNotFound
	}
	if err != nil {
		return Cleaner{}, fmt.Errorf("update cleaner %d: %w", id, err)
	}
	return updated, nil
}

// FindForUser resolves the cleaner profile belonging to a login user.
// It first tries the user_id link, then falls back to matching the user's
// email (so profiles created before linking still resolve).
func (r *Repository) FindForUser(ctx context.Context, userID int64) (Cleaner, error) {
	c, err := scanCleaner(r.pool.QueryRow(ctx,
		`SELECT `+cleanerColumns+` FROM cleaners WHERE user_id = $1`, userID))
	if err == nil {
		return c, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Cleaner{}, fmt.Errorf("find cleaner for user %d: %w", userID, err)
	}
	var email string
	if err := r.pool.QueryRow(ctx, `SELECT email FROM users WHERE id = $1`, userID).Scan(&email); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Cleaner{}, ErrNotFound
		}
		return Cleaner{}, fmt.Errorf("find user %d email: %w", userID, err)
	}
	c, err = scanCleaner(r.pool.QueryRow(ctx,
		`SELECT `+cleanerColumns+` FROM cleaners WHERE lower(email) = lower($1) AND user_id IS NULL`, email))
	if errors.Is(err, pgx.ErrNoRows) {
		return Cleaner{}, ErrNotFound
	}
	if err != nil {
		return Cleaner{}, fmt.Errorf("find cleaner by email for user %d: %w", userID, err)
	}
	return c, nil
}

// UpdateLocation records a GPS ping from the cleaner mobile app.
func (r *Repository) UpdateLocation(ctx context.Context, id int64, lat, lng *float64, area *string, isOnline *bool) (Cleaner, error) {
	updated, err := scanCleaner(r.pool.QueryRow(ctx,
		`UPDATE cleaners SET
			lat          = COALESCE($2, lat),
			lng          = COALESCE($3, lng),
			area         = COALESCE($4, area),
			is_online    = COALESCE($5, is_online),
			last_seen_at = now()
		 WHERE id = $1
		 RETURNING `+cleanerColumns,
		id, lat, lng, area, isOnline))
	if errors.Is(err, pgx.ErrNoRows) {
		return Cleaner{}, ErrNotFound
	}
	if err != nil {
		return Cleaner{}, fmt.Errorf("update location for cleaner %d: %w", id, err)
	}
	return updated, nil
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM cleaners WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete cleaner %d: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ListPhones returns all labeled phone numbers for a cleaner, ordered by id.
func (r *Repository) ListPhones(ctx context.Context, cleanerID int64) ([]PhoneNumber, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, label, phone FROM cleaner_phones WHERE cleaner_id = $1 ORDER BY id`, cleanerID)
	if err != nil {
		return nil, fmt.Errorf("list phones for cleaner %d: %w", cleanerID, err)
	}
	defer rows.Close()

	items := []PhoneNumber{}
	for rows.Next() {
		var p PhoneNumber
		if err := rows.Scan(&p.ID, &p.Label, &p.Phone); err != nil {
			return nil, fmt.Errorf("scan phone for cleaner %d: %w", cleanerID, err)
		}
		items = append(items, p)
	}
	return items, rows.Err()
}

// AddPhone inserts a labeled phone number for a cleaner.
// It returns ErrNotFound if the cleaner does not exist.
func (r *Repository) AddPhone(ctx context.Context, cleanerID int64, label, phone string) (PhoneNumber, error) {
	var exists int
	if err := r.pool.QueryRow(ctx, `SELECT 1 FROM cleaners WHERE id = $1`, cleanerID).Scan(&exists); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return PhoneNumber{}, ErrNotFound
		}
		return PhoneNumber{}, fmt.Errorf("check cleaner %d: %w", cleanerID, err)
	}
	var p PhoneNumber
	if err := r.pool.QueryRow(ctx,
		`INSERT INTO cleaner_phones (cleaner_id, label, phone) VALUES ($1, $2, $3) RETURNING id, label, phone`,
		cleanerID, label, phone).Scan(&p.ID, &p.Label, &p.Phone); err != nil {
		return PhoneNumber{}, fmt.Errorf("add phone for cleaner %d: %w", cleanerID, err)
	}
	return p, nil
}

// RemovePhone deletes a labeled phone number scoped to its cleaner.
// It returns ErrNotFound if the row does not exist.
func (r *Repository) RemovePhone(ctx context.Context, cleanerID, phoneID int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM cleaner_phones WHERE id = $1 AND cleaner_id = $2`, phoneID, cleanerID)
	if err != nil {
		return fmt.Errorf("delete phone %d for cleaner %d: %w", phoneID, cleanerID, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
