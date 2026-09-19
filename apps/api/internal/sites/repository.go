package sites

import (
	"context"
	"errors"
	"fmt"

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

const siteColumns = `s.id, s.customer_id,
	trim(c.first_name || ' ' || c.last_name),
	s.name, s.address, s.area, s.property_type,
	s.contact_name, s.contact_phone, s.contact_email,
	s.notes, s.status, s.lat, s.lng, s.created_at, s.updated_at`

const siteFrom = `FROM sites s JOIN customers c ON c.id = s.customer_id`

func scanSite(row pgx.Row) (Site, error) {
	var s Site
	err := row.Scan(&s.ID, &s.CustomerID, &s.CustomerName, &s.Name, &s.Address, &s.Area,
		&s.PropertyType, &s.ContactName, &s.ContactPhone, &s.ContactEmail,
		&s.Notes, &s.Status, &s.Lat, &s.Lng, &s.CreatedAt, &s.UpdatedAt)
	return s, err
}

// Filters narrows a List query. A zero CustomerID means every customer.
type Filters struct {
	CustomerID int64
}

func (r *Repository) List(ctx context.Context, f Filters, params pagination.Params) ([]Site, int, error) {
	args := []any{}
	conds := []string{}
	if f.CustomerID > 0 {
		args = append(args, f.CustomerID)
		conds = append(conds, "s.customer_id = $"+itoa(len(args)))
	}
	if params.Status != "" {
		args = append(args, params.Status)
		conds = append(conds, "s.status = $"+itoa(len(args)))
	}
	if params.Area != "" {
		args = append(args, params.Area)
		conds = append(conds, "s.area = $"+itoa(len(args)))
	}
	if params.Search != "" {
		like := "%" + params.Search + "%"
		start := len(args) + 1
		args = append(args, like, like, like)
		conds = append(conds, "(s.name ILIKE $"+itoa(start)+
			" OR s.address ILIKE $"+itoa(start+1)+
			" OR trim(c.first_name || ' ' || c.last_name) ILIKE $"+itoa(start+2)+")")
	}
	where := ""
	if len(conds) > 0 {
		where = "WHERE " + joinAnd(conds)
	}
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) `+siteFrom+` `+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count sites: %w", err)
	}
	args = append(args, params.Limit, params.Offset)
	query := `SELECT ` + siteColumns + ` ` + siteFrom + ` ` + where +
		` ORDER BY s.customer_id, s.name LIMIT $` + itoa(len(args)-1) + ` OFFSET $` + itoa(len(args))
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
		`SELECT `+siteColumns+` `+siteFrom+` WHERE s.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Site{}, ErrNotFound
	}
	if err != nil {
		return Site{}, fmt.Errorf("get site %d: %w", id, err)
	}
	return s, nil
}

// OwnerOf returns the customer a site belongs to, used to reject cross-customer
// assignments. Returns ErrNotFound when the site does not exist.
func (r *Repository) OwnerOf(ctx context.Context, siteID int64) (int64, error) {
	var customerID int64
	err := r.pool.QueryRow(ctx, `SELECT customer_id FROM sites WHERE id = $1`, siteID).Scan(&customerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("get owner of site %d: %w", siteID, err)
	}
	return customerID, nil
}

func (r *Repository) Create(ctx context.Context, s Site) (Site, error) {
	var id int64
	err := r.pool.QueryRow(ctx,
		`INSERT INTO sites (customer_id, name, address, area, property_type,
			contact_name, contact_phone, contact_email, notes, status, lat, lng)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		 RETURNING id`,
		s.CustomerID, s.Name, s.Address, s.Area, s.PropertyType,
		s.ContactName, s.ContactPhone, s.ContactEmail, s.Notes, s.Status, s.Lat, s.Lng).Scan(&id)
	if err != nil {
		return Site{}, fmt.Errorf("create site: %w", err)
	}
	return r.GetByID(ctx, id)
}

func (r *Repository) Update(ctx context.Context, id int64, req UpdateSiteRequest) (Site, error) {
	tag, err := r.pool.Exec(ctx,
		`UPDATE sites SET
			name          = COALESCE($2, name),
			address       = COALESCE($3, address),
			area          = COALESCE($4, area),
			property_type = COALESCE($5, property_type),
			contact_name  = COALESCE($6, contact_name),
			contact_phone = COALESCE($7, contact_phone),
			contact_email = COALESCE($8, contact_email),
			notes         = COALESCE($9, notes),
			status        = COALESCE($10, status),
			lat           = COALESCE($11, lat),
			lng           = COALESCE($12, lng)
		 WHERE id = $1`,
		id, req.Name, req.Address, req.Area, req.PropertyType,
		req.ContactName, req.ContactPhone, req.ContactEmail,
		req.Notes, req.Status, req.Lat, req.Lng)
	if err != nil {
		return Site{}, fmt.Errorf("update site %d: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return Site{}, ErrNotFound
	}
	return r.GetByID(ctx, id)
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM sites WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete site %d: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
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
