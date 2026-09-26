package complaints

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("complaint not found")

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const columns = `c.id, c.complaint_number, c.customer_id, TRIM(cu.first_name || ' ' || cu.last_name),
	c.site_id, COALESCE(s.name, ''), c.booking_id, COALESCE(b.booking_number, ''),
	c.category, c.severity, c.channel, c.description, c.status, c.due_at, c.resolution, c.resolved_at,
	c.reclean_booking_id, COALESCE(rb.booking_number, ''), c.created_at, c.updated_at`

const from = `FROM complaints c
	JOIN customers cu ON cu.id = c.customer_id
	LEFT JOIN sites s ON s.id = c.site_id
	LEFT JOIN bookings b ON b.id = c.booking_id
	LEFT JOIN bookings rb ON rb.id = c.reclean_booking_id`

func scan(row pgx.Row) (Complaint, error) {
	var c Complaint
	err := row.Scan(&c.ID, &c.Number, &c.CustomerID, &c.CustomerName, &c.SiteID, &c.SiteName,
		&c.BookingID, &c.BookingNumber, &c.Category, &c.Severity, &c.Channel, &c.Description,
		&c.Status, &c.DueAt, &c.Resolution, &c.ResolvedAt, &c.RecleanBookingID, &c.RecleanNumber,
		&c.CreatedAt, &c.UpdatedAt)
	return c, err
}

type Filters struct {
	Status     string // "", a status, or "active" (open + in_progress)
	CustomerID int64
	Overdue    bool
	Search     string
	Limit      int
	Offset     int
}

func (r *Repository) List(ctx context.Context, f Filters) ([]Complaint, int, error) {
	var conds []string
	var args []any
	add := func(cond string, v any) {
		args = append(args, v)
		conds = append(conds, strings.ReplaceAll(cond, "$?", "$"+strconv.Itoa(len(args))))
	}
	switch f.Status {
	case "":
	case "active":
		conds = append(conds, "c.status IN ('open', 'in_progress')")
	default:
		add("c.status = $?", f.Status)
	}
	if f.CustomerID > 0 {
		add("c.customer_id = $?", f.CustomerID)
	}
	if f.Overdue {
		conds = append(conds, "c.status IN ('open', 'in_progress') AND c.due_at < now()")
	}
	if q := strings.TrimSpace(f.Search); q != "" {
		add("(c.complaint_number ILIKE $? OR cu.first_name ILIKE $? OR cu.last_name ILIKE $? OR c.description ILIKE $?)", "%"+q+"%")
	}
	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) `+from+` `+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count complaints: %w", err)
	}
	args = append(args, f.Limit, f.Offset)
	rows, err := r.pool.Query(ctx,
		`SELECT `+columns+` `+from+` `+where+
			` ORDER BY (c.status IN ('open', 'in_progress')) DESC, c.due_at ASC, c.id DESC
			  LIMIT $`+strconv.Itoa(len(args)-1)+` OFFSET $`+strconv.Itoa(len(args)), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list complaints: %w", err)
	}
	defer rows.Close()
	out := []Complaint{}
	for rows.Next() {
		c, err := scan(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, c)
	}
	return out, total, rows.Err()
}

func (r *Repository) Get(ctx context.Context, id int64) (Complaint, error) {
	c, err := scan(r.pool.QueryRow(ctx, `SELECT `+columns+` `+from+` WHERE c.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Complaint{}, ErrNotFound
	}
	return c, err
}

// BookingOwner returns the customer and site of a booking.
func (r *Repository) BookingOwner(ctx context.Context, bookingID int64) (customerID *int64, siteID *int64, err error) {
	err = r.pool.QueryRow(ctx, `SELECT customer_id, site_id FROM bookings WHERE id = $1`, bookingID).Scan(&customerID, &siteID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, ErrNotFound
	}
	return customerID, siteID, err
}

// SiteCustomer returns the customer a site belongs to.
func (r *Repository) SiteCustomer(ctx context.Context, siteID int64) (int64, error) {
	var id int64
	err := r.pool.QueryRow(ctx, `SELECT customer_id FROM sites WHERE id = $1`, siteID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound
	}
	return id, err
}

func (r *Repository) Create(ctx context.Context, c Complaint, userID int64) (Complaint, error) {
	var id int64
	err := r.pool.QueryRow(ctx,
		`INSERT INTO complaints (complaint_number, customer_id, site_id, booking_id, category, severity, channel,
		                         description, due_at, created_by)
		 VALUES ('CP-' || to_char(now(), 'YYYY') || '-' || lpad(nextval('complaint_seq')::text, 4, '0'),
		         $1, $2, $3, $4, $5, $6, $7, $8, NULLIF($9, 0))
		 RETURNING id`,
		c.CustomerID, c.SiteID, c.BookingID, c.Category, c.Severity, c.Channel, c.Description, c.DueAt, userID).Scan(&id)
	if err != nil {
		return Complaint{}, fmt.Errorf("create complaint: %w", err)
	}
	return r.Get(ctx, id)
}

type Patch struct {
	Status      *Status
	Severity    *string
	Category    *string
	Description *string
	Resolution  *string
	DueAt       *time.Time
}

func (r *Repository) Update(ctx context.Context, id int64, p Patch, userID int64) (Complaint, error) {
	var status any
	if p.Status != nil {
		status = string(*p.Status)
	}
	tag, err := r.pool.Exec(ctx,
		`UPDATE complaints SET
		   status      = COALESCE($2, status),
		   severity    = COALESCE($3, severity),
		   category    = COALESCE($4, category),
		   description = COALESCE($5, description),
		   resolution  = COALESCE($6, resolution),
		   due_at      = COALESCE($7, due_at),
		   resolved_at = CASE WHEN COALESCE($2, status) IN ('resolved', 'closed')
		                      THEN COALESCE(resolved_at, now()) ELSE NULL END,
		   resolved_by = CASE WHEN COALESCE($2, status) IN ('resolved', 'closed')
		                      THEN COALESCE(resolved_by, NULLIF($8, 0)) ELSE NULL END,
		   updated_at  = now()
		 WHERE id = $1`,
		id, status, p.Severity, p.Category, p.Description, p.Resolution, p.DueAt, userID)
	if err != nil {
		return Complaint{}, fmt.Errorf("update complaint %d: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return Complaint{}, ErrNotFound
	}
	return r.Get(ctx, id)
}

// LinkReclean records the re-clean booking and moves an open complaint to
// in_progress.
func (r *Repository) LinkReclean(ctx context.Context, id, bookingID int64) (Complaint, error) {
	if _, err := r.pool.Exec(ctx,
		`UPDATE complaints SET reclean_booking_id = $2,
		        status = CASE WHEN status = 'open' THEN 'in_progress' ELSE status END, updated_at = now()
		  WHERE id = $1`, id, bookingID); err != nil {
		return Complaint{}, fmt.Errorf("link re-clean for complaint %d: %w", id, err)
	}
	return r.Get(ctx, id)
}

// SourceBooking is what a re-clean copies from the original job.
type SourceBooking struct {
	CustomerName    string
	CustomerEmail   string
	ServiceType     string
	DurationMinutes int
	Address         string
	Area            string
	CleanerIDs      []int64
}

func (r *Repository) SourceBooking(ctx context.Context, bookingID int64) (SourceBooking, error) {
	var s SourceBooking
	err := r.pool.QueryRow(ctx,
		`SELECT customer_name, COALESCE(customer_email, ''), service_type, duration_minutes, address, COALESCE(area, ''),
		        COALESCE(ARRAY(SELECT cleaner_id FROM booking_cleaners WHERE booking_id = b.id
		                        ORDER BY (role = 'primary') DESC, id), '{}')
		   FROM bookings b WHERE id = $1`, bookingID).
		Scan(&s.CustomerName, &s.CustomerEmail, &s.ServiceType, &s.DurationMinutes, &s.Address, &s.Area, &s.CleanerIDs)
	if errors.Is(err, pgx.ErrNoRows) {
		return SourceBooking{}, ErrNotFound
	}
	return s, err
}

// CustomerBasics loads what a re-clean without an original booking needs.
func (r *Repository) CustomerBasics(ctx context.Context, customerID int64) (SourceBooking, error) {
	var s SourceBooking
	err := r.pool.QueryRow(ctx,
		`SELECT TRIM(first_name || ' ' || last_name), COALESCE(email, ''), address, COALESCE(area, '')
		   FROM customers WHERE id = $1`, customerID).Scan(&s.CustomerName, &s.CustomerEmail, &s.Address, &s.Area)
	if errors.Is(err, pgx.ErrNoRows) {
		return SourceBooking{}, ErrNotFound
	}
	return s, err
}
