package portal

import (
	"context"
	"errors"
	"fmt"
	"strings"

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
		 WHERE lower(email) = lower($1)`, strings.TrimSpace(email)))
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
		`SELECT id, first_name || ' ' || last_name, email, phone, address, area
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
