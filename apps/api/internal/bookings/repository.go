package bookings

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("booking not found")

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const bookingColumns = `id, booking_number, customer_name, service_type, scheduled_for, duration_minutes, address, assigned_cleaner, status, notes, created_at`

const bookingNumberExpr = `'BK-' || to_char(created_at, 'YYYY') || '-' || lpad(id::text, 4, '0')`

func scanBooking(row pgx.Row) (Booking, error) {
	var b Booking
	err := row.Scan(&b.ID, &b.BookingNumber, &b.CustomerName, &b.ServiceType,
		&b.ScheduledFor, &b.DurationMinutes, &b.Address, &b.AssignedCleaner,
		&b.Status, &b.Notes, &b.CreatedAt)
	return b, err
}

func (r *Repository) List(ctx context.Context) ([]Booking, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+bookingColumns+` FROM bookings ORDER BY scheduled_for DESC`)
	if err != nil {
		return nil, fmt.Errorf("query bookings: %w", err)
	}
	defer rows.Close()

	bookings := []Booking{}
	for rows.Next() {
		b, err := scanBooking(rows)
		if err != nil {
			return nil, fmt.Errorf("scan booking: %w", err)
		}
		bookings = append(bookings, b)
	}
	return bookings, rows.Err()
}

func (r *Repository) GetByID(ctx context.Context, id int64) (Booking, error) {
	b, err := scanBooking(r.pool.QueryRow(ctx,
		`SELECT `+bookingColumns+` FROM bookings WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Booking{}, ErrNotFound
	}
	if err != nil {
		return Booking{}, fmt.Errorf("get booking %d: %w", id, err)
	}
	return b, nil
}

func (r *Repository) Create(ctx context.Context, b Booking) (Booking, error) {
	var id int64
	if err := r.pool.QueryRow(ctx,
		`INSERT INTO bookings (booking_number, customer_name, service_type, scheduled_for, duration_minutes, address, assigned_cleaner, status, notes)
		 VALUES ('', $1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING id`,
		b.CustomerName, b.ServiceType, b.ScheduledFor, b.DurationMinutes,
		b.Address, b.AssignedCleaner, b.Status, b.Notes).Scan(&id); err != nil {
		return Booking{}, fmt.Errorf("create booking: %w", err)
	}

	created, err := scanBooking(r.pool.QueryRow(ctx,
		`UPDATE bookings SET booking_number = `+bookingNumberExpr+`
		 WHERE id = $1
		 RETURNING `+bookingColumns,
		id))
	if err != nil {
		return Booking{}, fmt.Errorf("assign booking number to booking %d: %w", id, err)
	}
	return created, nil
}

type Patch struct {
	CustomerName    *string
	ServiceType     *ServiceType
	ScheduledFor    *time.Time
	DurationMinutes *int
	Address         *string
	AssignedCleaner *string
	Status          *Status
	Notes           *string
}

func (r *Repository) Update(ctx context.Context, id int64, p Patch) (Booking, error) {
	var serviceType any
	if p.ServiceType != nil {
		serviceType = *p.ServiceType
	}
	var status any
	if p.Status != nil {
		status = *p.Status
	}

	updated, err := scanBooking(r.pool.QueryRow(ctx,
		`UPDATE bookings SET
			customer_name    = COALESCE($2, customer_name),
			service_type     = COALESCE($3, service_type),
			scheduled_for    = COALESCE($4, scheduled_for),
			duration_minutes = COALESCE($5, duration_minutes),
			address          = COALESCE($6, address),
			assigned_cleaner = COALESCE($7, assigned_cleaner),
			status           = COALESCE($8, status),
			notes            = COALESCE($9, notes)
		 WHERE id = $1
		 RETURNING `+bookingColumns,
		id, p.CustomerName, serviceType, p.ScheduledFor, p.DurationMinutes,
		p.Address, p.AssignedCleaner, status, p.Notes))
	if errors.Is(err, pgx.ErrNoRows) {
		return Booking{}, ErrNotFound
	}
	if err != nil {
		return Booking{}, fmt.Errorf("update booking %d: %w", id, err)
	}
	return updated, nil
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM bookings WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete booking %d: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
