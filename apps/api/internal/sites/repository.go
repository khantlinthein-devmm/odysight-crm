package sites

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/odysight/crm/pkg/pagination"
)

var ErrNotFound = errors.New("site not found")

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const siteColumns = `id, customer_id, name, address, contact_name, phone, email, notes, status, latitude, longitude, is_default, created_at, updated_at`

func scanSite(row pgx.Row) (Site, error) {
	var s Site
	err := row.Scan(&s.ID, &s.CustomerID, &s.Name, &s.Address, &s.ContactName,
		&s.Phone, &s.Email, &s.Notes, &s.Status, &s.Latitude, &s.Longitude,
		&s.IsDefault, &s.CreatedAt, &s.UpdatedAt)
	return s, err
}

func (r *Repository) List(ctx context.Context, params pagination.Params, customerID int64) ([]Site, int, error) {
	args := []any{}
	conds := []string{}
	if customerID > 0 {
		args = append(args, customerID)
		conds = append(conds, "customer_id = $"+strconv.Itoa(len(args)))
	}
	if params.Search != "" {
		like := "%" + params.Search + "%"
		start := len(args) + 1
		orParts := []string{}
		for i, col := range []string{"name", "address", "contact_name"} {
			orParts = append(orParts, col+" ILIKE $"+strconv.Itoa(start+i))
			args = append(args, like)
		}
		conds = append(conds, "("+strings.Join(orParts, " OR ")+")")
	}
	if params.Status != "" {
		args = append(args, params.Status)
		conds = append(conds, "status = $"+strconv.Itoa(len(args)))
	}
	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM sites `+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count sites: %w", err)
	}
	args = append(args, params.Limit, params.Offset)
	query := `SELECT ` + siteColumns + ` FROM sites ` + where +
		` ORDER BY is_default DESC, created_at DESC LIMIT $` + strconv.Itoa(len(args)-1) + ` OFFSET $` + strconv.Itoa(len(args))
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query sites: %w", err)
	}
	defer rows.Close()

	items := []Site{}
	for rows.Next() {
		item, err := scanSite(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan site: %w", err)
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *Repository) GetByID(ctx context.Context, id int64) (Site, error) {
	s, err := scanSite(r.pool.QueryRow(ctx,
		`SELECT `+siteColumns+` FROM sites WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Site{}, ErrNotFound
	}
	if err != nil {
		return Site{}, fmt.Errorf("get site %d: %w", id, err)
	}
	return s, nil
}

func (r *Repository) Create(ctx context.Context, s Site) (Site, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Site{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if s.IsDefault {
		if _, err := tx.Exec(ctx, `UPDATE sites SET is_default = FALSE WHERE customer_id = $1`, s.CustomerID); err != nil {
			return Site{}, fmt.Errorf("clear default site: %w", err)
		}
	}
	created, err := scanSite(tx.QueryRow(ctx,
		`INSERT INTO sites (customer_id, name, address, contact_name, phone, email, notes, status, latitude, longitude, is_default)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		 RETURNING `+siteColumns,
		s.CustomerID, s.Name, s.Address, s.ContactName, s.Phone, s.Email, s.Notes, s.Status, s.Latitude, s.Longitude, s.IsDefault))
	if err != nil {
		return Site{}, fmt.Errorf("create site: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Site{}, fmt.Errorf("commit site: %w", err)
	}
	return created, nil
}

type Patch struct {
	Name        *string
	Address     *string
	ContactName *string
	Phone       *string
	Email       *string
	Notes       *string
	Status      *Status
	Latitude    *float64
	Longitude   *float64
	IsDefault   *bool
}

func (r *Repository) Update(ctx context.Context, id int64, p Patch) (Site, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Site{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var customerID int64
	if err := tx.QueryRow(ctx, `SELECT customer_id FROM sites WHERE id = $1`, id).Scan(&customerID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Site{}, ErrNotFound
		}
		return Site{}, fmt.Errorf("get site customer %d: %w", id, err)
	}
	if p.IsDefault != nil && *p.IsDefault {
		if _, err := tx.Exec(ctx, `UPDATE sites SET is_default = FALSE WHERE customer_id = $1 AND id <> $2`, customerID, id); err != nil {
			return Site{}, fmt.Errorf("clear default site: %w", err)
		}
	}

	var status any
	if p.Status != nil {
		status = *p.Status
	}
	updated, err := scanSite(tx.QueryRow(ctx,
		`UPDATE sites SET
			name         = COALESCE($2, name),
			address      = COALESCE($3, address),
			contact_name = COALESCE($4, contact_name),
			phone        = COALESCE($5, phone),
			email        = COALESCE($6, email),
			notes        = COALESCE($7, notes),
			status       = COALESCE($8, status),
			latitude     = COALESCE($9, latitude),
			longitude    = COALESCE($10, longitude),
			is_default   = COALESCE($11, is_default)
		 WHERE id = $1
		 RETURNING `+siteColumns,
		id, p.Name, p.Address, p.ContactName, p.Phone, p.Email, p.Notes, status, p.Latitude, p.Longitude, p.IsDefault))
	if errors.Is(err, pgx.ErrNoRows) {
		return Site{}, ErrNotFound
	}
	if err != nil {
		return Site{}, fmt.Errorf("update site %d: %w", id, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Site{}, fmt.Errorf("commit site %d: %w", id, err)
	}
	return updated, nil
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	var isDefault bool
	if err := r.pool.QueryRow(ctx, `SELECT is_default FROM sites WHERE id = $1`, id).Scan(&isDefault); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("get site %d: %w", id, err)
	}
	if isDefault {
		var others int
		if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM sites WHERE customer_id = (SELECT customer_id FROM sites WHERE id = $1) AND id <> $1`, id).Scan(&others); err != nil {
			return fmt.Errorf("count sibling sites: %w", err)
		}
		if others > 0 {
			return fmt.Errorf("cannot delete the default site while other sites exist: promote another site first")
		}
	}
	tag, err := r.pool.Exec(ctx, `DELETE FROM sites WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete site %d: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
