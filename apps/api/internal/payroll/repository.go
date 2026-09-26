package payroll

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// CleanerRef is a cleaner as payroll lists them.
type CleanerRef struct {
	ID     int64
	Name   string
	Status string
}

func (r *Repository) Cleaners(ctx context.Context) ([]CleanerRef, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, trim(first_name || ' ' || last_name), status FROM cleaners ORDER BY first_name, last_name, id`)
	if err != nil {
		return nil, fmt.Errorf("list cleaners: %w", err)
	}
	defer rows.Close()
	var out []CleanerRef
	for rows.Next() {
		var c CleanerRef
		if err := rows.Scan(&c.ID, &c.Name, &c.Status); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *Repository) Rates(ctx context.Context) (map[int64]Rate, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT cleaner_id, pay_type, rate::float8, ot_multiplier::float8, standard_hours::float8, updated_at
		   FROM cleaner_pay_rates`)
	if err != nil {
		return nil, fmt.Errorf("list pay rates: %w", err)
	}
	defer rows.Close()
	out := map[int64]Rate{}
	for rows.Next() {
		var rt Rate
		if err := rows.Scan(&rt.CleanerID, &rt.PayType, &rt.Rate, &rt.OTMultiplier, &rt.StandardHours, &rt.UpdatedAt); err != nil {
			return nil, err
		}
		out[rt.CleanerID] = rt
	}
	return out, rows.Err()
}

func (r *Repository) UpsertRate(ctx context.Context, rt Rate, userID int64) (Rate, error) {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO cleaner_pay_rates (cleaner_id, pay_type, rate, ot_multiplier, standard_hours, updated_by, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, now())
		 ON CONFLICT (cleaner_id) DO UPDATE SET
		   pay_type = EXCLUDED.pay_type, rate = EXCLUDED.rate, ot_multiplier = EXCLUDED.ot_multiplier,
		   standard_hours = EXCLUDED.standard_hours, updated_by = EXCLUDED.updated_by, updated_at = now()
		 RETURNING updated_at`,
		rt.CleanerID, rt.PayType, rt.Rate, rt.OTMultiplier, rt.StandardHours, userID).Scan(&rt.UpdatedAt)
	if err != nil {
		return Rate{}, fmt.Errorf("save pay rate for cleaner %d: %w", rt.CleanerID, err)
	}
	return rt, nil
}

// Shifts returns each cleaner's attendance days within [from, to] (dates).
func (r *Repository) Shifts(ctx context.Context, from, to string) (map[int64][]Shift, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT cleaner_id, check_in_at, check_out_at FROM attendance
		  WHERE cleaner_id IS NOT NULL AND work_date BETWEEN $1::date AND $2::date`, from, to)
	if err != nil {
		return nil, fmt.Errorf("attendance for payroll: %w", err)
	}
	defer rows.Close()
	out := map[int64][]Shift{}
	for rows.Next() {
		var id int64
		var s Shift
		if err := rows.Scan(&id, &s.CheckIn, &s.CheckOut); err != nil {
			return nil, err
		}
		out[id] = append(out[id], s)
	}
	return out, rows.Err()
}

// CompletedJobs counts each cleaner's completed bookings scheduled within
// [from, to+1day) in local time.
func (r *Repository) CompletedJobs(ctx context.Context, from, to time.Time) (map[int64]int, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT bc.cleaner_id, COUNT(*) FROM bookings b
		   JOIN booking_cleaners bc ON bc.booking_id = b.id
		  WHERE b.status = 'completed' AND b.scheduled_for >= $1 AND b.scheduled_for < $2
		  GROUP BY bc.cleaner_id`, from, to)
	if err != nil {
		return nil, fmt.Errorf("completed jobs for payroll: %w", err)
	}
	defer rows.Close()
	out := map[int64]int{}
	for rows.Next() {
		var id int64
		var n int
		if err := rows.Scan(&id, &n); err != nil {
			return nil, err
		}
		out[id] = n
	}
	return out, rows.Err()
}
