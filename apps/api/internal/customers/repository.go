package customers

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/odysight/crm/pkg/pagination"
)

var ErrNotFound = errors.New("customer not found")

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const customerColumns = `id, first_name, last_name, email, phone, address, property_type, area, status, lead_id, portal_enabled, tax_id, tax_branch, withholding_rate::float8, line_user_id <> '', created_at`

func scanCustomer(row pgx.Row) (Customer, error) {
	var c Customer
	err := row.Scan(&c.ID, &c.FirstName, &c.LastName, &c.Email, &c.Phone,
		&c.Address, &c.PropertyType, &c.Area, &c.Status, &c.LeadID, &c.PortalEnabled,
		&c.TaxID, &c.TaxBranch, &c.WithholdingRate, &c.LineLinked, &c.CreatedAt)
	return c, err
}

func (r *Repository) List(ctx context.Context, params pagination.Params) ([]Customer, int, error) {
	args := []any{}
	conds := []string{}
	if params.Search != "" {
		like := "%" + params.Search + "%"
		start := len(args) + 1
		orParts := []string{}
		for i, col := range []string{"first_name", "last_name", "email", "phone", "area"} {
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
	where := ""
	if len(conds) > 0 {
		where = "WHERE " + joinAnd(conds)
	}
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM customers `+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count customers: %w", err)
	}
	args = append(args, params.Limit, params.Offset)
	query := `SELECT ` + customerColumns + ` FROM customers ` + where + ` ORDER BY created_at DESC LIMIT $` + itoa(len(args)-1) + ` OFFSET $` + itoa(len(args))
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query customers: %w", err)
	}
	defer rows.Close()

	items := []Customer{}
	for rows.Next() {
		item, err := scanCustomer(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan customer: %w", err)
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func itoa(i int) string             { return fmt.Sprintf("%d", i) }
func joinOr(parts []string) string  { return joinWith(parts, " OR ") }
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

func (r *Repository) GetByID(ctx context.Context, id int64) (Customer, error) {
	c, err := scanCustomer(r.pool.QueryRow(ctx,
		`SELECT `+customerColumns+` FROM customers WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Customer{}, ErrNotFound
	}
	if err != nil {
		return Customer{}, fmt.Errorf("get customer %d: %w", id, err)
	}
	return c, nil
}

// Create inserts the customer together with its Default Site in one
// statement, matching the backfill in migration 000025, so every customer
// has a site to book against.
func (r *Repository) Create(ctx context.Context, c Customer) (Customer, error) {
	created, err := scanCustomer(r.pool.QueryRow(ctx,
		`WITH c AS (
		   INSERT INTO customers (first_name, last_name, email, phone, address, property_type, area, status, lead_id,
		                          tax_id, tax_branch, withholding_rate, line_user_id)
		   VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12,
		           COALESCE((SELECT line_user_id FROM leads WHERE id = $9), ''))
		   RETURNING *
		 ), site AS (
		   INSERT INTO sites (customer_id, name, address, contact_name, phone, email, status, is_default)
		   SELECT id, 'Default Site', COALESCE(NULLIF(address, ''), 'Address on file'),
		          TRIM(first_name || ' ' || last_name), COALESCE(phone, ''), COALESCE(email, ''), 'active', TRUE
		     FROM c
		 )
		 SELECT `+customerColumns+` FROM c`,
		c.FirstName, c.LastName, c.Email, c.Phone, c.Address, c.PropertyType, c.Area, c.Status, c.LeadID,
		c.TaxID, c.TaxBranch, c.WithholdingRate))
	if err != nil {
		return Customer{}, fmt.Errorf("create customer: %w", err)
	}
	return created, nil
}

type Patch struct {
	FirstName       *string
	LastName        *string
	Email           *string
	Phone           *string
	Address         *string
	PropertyType    *PropertyType
	Area            *string
	Status          *Status
	TaxID           *string
	TaxBranch       *string
	WithholdingRate *float64
}

func (r *Repository) Update(ctx context.Context, id int64, p Patch) (Customer, error) {
	var propType any
	if p.PropertyType != nil {
		propType = *p.PropertyType
	}
	var status any
	if p.Status != nil {
		status = *p.Status
	}

	updated, err := scanCustomer(r.pool.QueryRow(ctx,
		`UPDATE customers SET
			first_name    = COALESCE($2, first_name),
			last_name     = COALESCE($3, last_name),
			email         = COALESCE($4, email),
			phone         = COALESCE($5, phone),
			address       = COALESCE($6, address),
			property_type = COALESCE($7, property_type),
			area          = COALESCE($8, area),
			status        = COALESCE($9, status),
			tax_id        = COALESCE($10, tax_id),
			tax_branch    = COALESCE($11, tax_branch),
			withholding_rate = COALESCE($12, withholding_rate)
		 WHERE id = $1
		 RETURNING `+customerColumns,
		id, p.FirstName, p.LastName, p.Email, p.Phone, p.Address, propType, p.Area, status,
		p.TaxID, p.TaxBranch, p.WithholdingRate))
	if errors.Is(err, pgx.ErrNoRows) {
		return Customer{}, ErrNotFound
	}
	if err != nil {
		return Customer{}, fmt.Errorf("update customer %d: %w", id, err)
	}
	return updated, nil
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM customers WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete customer %d: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdatePortalAuth sets the customer's portal password hash (nil clears it)
// and portal_enabled flag.
func (r *Repository) UpdatePortalAuth(ctx context.Context, id int64, passwordHash *string, enabled bool) (Customer, error) {
	updated, err := scanCustomer(r.pool.QueryRow(ctx,
		`UPDATE customers SET
			password_hash  = COALESCE($2, password_hash),
			portal_enabled = $3
		 WHERE id = $1
		 RETURNING `+customerColumns,
		id, passwordHash, enabled))
	if errors.Is(err, pgx.ErrNoRows) {
		return Customer{}, ErrNotFound
	}
	if err != nil {
		return Customer{}, fmt.Errorf("update portal auth for customer %d: %w", id, err)
	}
	return updated, nil
}
