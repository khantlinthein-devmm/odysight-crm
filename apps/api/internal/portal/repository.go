package portal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrInvalidCredentials = errors.New("invalid portal credentials")
var ErrNotFound = errors.New("portal customer not found")

// customerWithHash is the full row used to verify portal credentials.
// PasswordHash is nullable: customers without a portal password yet have NULL.
type customerWithHash struct {
	ID            int64
	Name          string
	Email         string
	Phone         string
	Address       string
	Area          string
	PasswordHash  *string
	PortalEnabled bool
	Status        string
}

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const portalCustomerColumns = `id, first_name, last_name, email, phone, address, area`

func scanCustomer(row pgx.Row) (customerWithHash, error) {
	var fn, ln string
	var c customerWithHash
	err := row.Scan(&c.ID, &fn, &ln, &c.Email, &c.Phone, &c.Address, &c.Area, &c.PasswordHash, &c.PortalEnabled, &c.Status)
	c.Name = strings.TrimSpace(fn + " " + ln)
	return c, err
}

// FindByEmail returns the customer with the given email address, including
// the stored password hash and portal flags. Matches case-insensitively.
func (r *Repository) FindByEmail(ctx context.Context, email string) (customerWithHash, error) {
	c, err := scanCustomer(r.pool.QueryRow(ctx,
		`SELECT `+portalCustomerColumns+`, password_hash, portal_enabled, status
		 FROM customers
		 WHERE email <> '' AND lower(email) = lower($1)`, strings.TrimSpace(email)))
	if errors.Is(err, pgx.ErrNoRows) {
		return customerWithHash{}, ErrInvalidCredentials
	}
	if err != nil {
		return customerWithHash{}, fmt.Errorf("find portal customer by email: %w", err)
	}
	return c, nil
}

// GetByID returns a customer's portal profile by id.
func (r *Repository) GetByID(ctx context.Context, id int64) (Customer, error) {
	var c Customer
	err := r.pool.QueryRow(ctx,
		`SELECT id, trim(first_name || ' ' || last_name), email, phone, address, area
		 FROM customers WHERE id = $1`, id).
		Scan(&c.ID, &c.Name, &c.Email, &c.Phone, &c.Address, &c.Area)
	if errors.Is(err, pgx.ErrNoRows) {
		return Customer{}, ErrNotFound
	}
	if err != nil {
		return Customer{}, fmt.Errorf("get portal customer %d: %w", id, err)
	}
	return c, nil
}

// Bookings lists a customer's own bookings, most recent first.
func (r *Repository) Bookings(ctx context.Context, customerID int64) ([]Booking, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, booking_number, service_type, scheduled_for, duration_minutes, address, COALESCE(assigned_cleaner, ''), status, notes
		 FROM bookings
		 WHERE customer_id = $1
		 ORDER BY scheduled_for DESC`, customerID)
	if err != nil {
		return nil, fmt.Errorf("query portal bookings: %w", err)
	}
	defer rows.Close()

	items := []Booking{}
	for rows.Next() {
		var b Booking
		if err := rows.Scan(&b.ID, &b.BookingNumber, &b.ServiceType,
			&b.ScheduledFor, &b.DurationMinutes, &b.Address, &b.Assignee, &b.Status, &b.Notes); err != nil {
			return nil, fmt.Errorf("scan portal booking: %w", err)
		}
		items = append(items, b)
	}
	return items, rows.Err()
}

// AccessState reports whether a customer may currently use the portal:
// enabled flag set and not blocked. Used to re-check on every request so
// disabling/blocking takes effect before the JWT expires.
func (r *Repository) AccessState(ctx context.Context, id int64) (enabled bool, blocked bool, err error) {
	var status string
	err = r.pool.QueryRow(ctx,
		`SELECT portal_enabled, status FROM customers WHERE id = $1`, id).
		Scan(&enabled, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, false, ErrNotFound
	}
	if err != nil {
		return false, false, fmt.Errorf("check portal access for customer %d: %w", id, err)
	}
	return enabled, status == "blocked", nil
}

// PasswordHashByID returns the stored password hash for a customer, or empty
// string when the customer has no portal password yet.
func (r *Repository) PasswordHashByID(ctx context.Context, id int64) (string, error) {
	var hash *string
	err := r.pool.QueryRow(ctx,
		`SELECT password_hash FROM customers WHERE id = $1`, id).Scan(&hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("get portal password hash: %w", err)
	}
	if hash == nil {
		return "", nil
	}
	return *hash, nil
}

// SetPassword updates a customer's portal password hash.
func (r *Repository) SetPassword(ctx context.Context, id int64, hash string) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE customers SET password_hash = $1 WHERE id = $2`, hash, id)
	if err != nil {
		return fmt.Errorf("update portal password for customer %d: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// EnsureBookingOwned verifies that a booking belongs to the given customer
// and returns the booking's status. Returns ErrNotFound when not owned.
func (r *Repository) EnsureBookingOwned(ctx context.Context, customerID, bookingID int64) (string, error) {
	var status string
	err := r.pool.QueryRow(ctx,
		`SELECT status FROM bookings WHERE id = $1 AND customer_id = $2`, bookingID, customerID).
		Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("check booking %d ownership: %w", bookingID, err)
	}
	return status, nil
}

// ListSites returns the customer's own service locations, default first.
func (r *Repository) ListSites(ctx context.Context, customerID int64) ([]Site, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, address, is_default FROM sites
		 WHERE customer_id = $1 AND status = 'active'
		 ORDER BY is_default DESC, created_at DESC`, customerID)
	if err != nil {
		return nil, fmt.Errorf("query portal sites: %w", err)
	}
	defer rows.Close()
	items := []Site{}
	for rows.Next() {
		var s Site
		if err := rows.Scan(&s.ID, &s.Name, &s.Address, &s.IsDefault); err != nil {
			return nil, fmt.Errorf("scan portal site: %w", err)
		}
		items = append(items, s)
	}
	return items, rows.Err()
}

// ActiveServices reads the workspace service catalog from settings and
// returns only active entries. Falls back to empty (never errors the portal)
// when the settings row is missing.
func (r *Repository) ActiveServices(ctx context.Context) ([]ServiceItem, error) {
	var raw []byte
	err := r.pool.QueryRow(ctx, `SELECT value FROM settings WHERE key = 'services'`).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return []ServiceItem{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query portal services: %w", err)
	}
	var all []ServiceItem
	if err := json.Unmarshal(raw, &all); err != nil {
		return nil, fmt.Errorf("decode portal services: %w", err)
	}
	out := []ServiceItem{}
	for _, s := range all {
		if s.Active {
			out = append(out, s)
		}
	}
	return out, nil
}

// portalCustomerBookingInfo loads the snapshot fields needed to create a
// customer-owned booking.
type portalCustomerBookingInfo struct {
	Name    string
	Email   string
	Address string
	Area    string
}

// CreatePortalBooking inserts a pending booking owned by the given customer.
// Site ownership is enforced: a site belonging to another customer is
// rejected. Address falls back to the site address, then the customer
// address, so one-time portal bookings work without a site.
func (r *Repository) CreatePortalBooking(ctx context.Context, customerID int64, req CreateBookingRequest, scheduledFor time.Time, duration int) (Booking, error) {
	var info portalCustomerBookingInfo
	err := r.pool.QueryRow(ctx,
		`SELECT trim(first_name || ' ' || last_name), email, address, area FROM customers WHERE id = $1`,
		customerID).Scan(&info.Name, &info.Email, &info.Address, &info.Area)
	if errors.Is(err, pgx.ErrNoRows) {
		return Booking{}, ErrNotFound
	}
	if err != nil {
		return Booking{}, fmt.Errorf("load portal customer %d: %w", customerID, err)
	}
	address := strings.TrimSpace(req.Address)
	var siteID *int64
	if req.SiteID != nil {
		var owner int64
		var siteAddr string
		err := r.pool.QueryRow(ctx, `SELECT customer_id, address FROM sites WHERE id = $1`, *req.SiteID).
			Scan(&owner, &siteAddr)
		if errors.Is(err, pgx.ErrNoRows) {
			return Booking{}, fmt.Errorf("site not found")
		}
		if err != nil {
			return Booking{}, fmt.Errorf("check portal site: %w", err)
		}
		if owner != customerID {
			return Booking{}, fmt.Errorf("site does not belong to this customer")
		}
		siteID = req.SiteID
		if address == "" {
			address = siteAddr
		}
	}
	if address == "" {
		address = info.Address
	}
	temp := "TMP-" + fmt.Sprintf("%d", time.Now().UnixNano())
	var id int64
	var bookingNumber string
	var created time.Time
	err = r.pool.QueryRow(ctx,
		`INSERT INTO bookings (booking_number, customer_name, customer_email, customer_id, site_id,
		 service_type, scheduled_for, duration_minutes, address, area, assigned_cleaner, status, notes)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, '', 'pending', $11)
		 RETURNING id, booking_number, scheduled_for`,
		temp, info.Name, info.Email, customerID, siteID,
		strings.TrimSpace(req.ServiceType), scheduledFor, duration, address, info.Area, strings.TrimSpace(req.Notes),
	).Scan(&id, &bookingNumber, &created)
	if err != nil {
		return Booking{}, fmt.Errorf("create portal booking: %w", err)
	}
	// Replace the temp number with the human-readable BK-YYYY-NNNN value.
	err = r.pool.QueryRow(ctx,
		`UPDATE bookings SET booking_number = 'BK-' || to_char(created_at,'YYYY') || '-' || lpad(id::text,4,'0')
		 WHERE id = $1
		 RETURNING booking_number, scheduled_for, duration_minutes, address, status, notes`,
		id).Scan(&bookingNumber, &created, &duration, &address, new(string), new(string))
	// Re-read the full row for a clean response (cheap, single row).
	var b Booking
	err = r.pool.QueryRow(ctx,
		`SELECT id, booking_number, service_type, scheduled_for, duration_minutes, address,
		 COALESCE(assigned_cleaner, ''), status, notes FROM bookings WHERE id = $1`, id).
		Scan(&b.ID, &b.BookingNumber, &b.ServiceType, &b.ScheduledFor, &b.DurationMinutes,
			&b.Address, &b.Assignee, &b.Status, &b.Notes)
	if err != nil {
		return Booking{}, fmt.Errorf("reload portal booking %d: %w", id, err)
	}
	return b, nil
}

// CancelPortalBooking cancels the customer's own pending/confirmed booking.
func (r *Repository) CancelPortalBooking(ctx context.Context, customerID, bookingID int64) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE bookings SET status = 'cancelled' WHERE id = $1 AND customer_id = $2
		 AND status IN ('pending', 'confirmed')`, bookingID, customerID)
	if err != nil {
		return fmt.Errorf("cancel portal booking %d: %w", bookingID, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
