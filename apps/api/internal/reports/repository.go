package reports

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

func (r *Repository) LoadSummary(ctx context.Context, now time.Time) (Summary, error) {
	var row summaryRow
	err := r.pool.QueryRow(ctx,
		`SELECT
			(SELECT COUNT(*) FROM leads),
			(SELECT COUNT(*) FROM customers WHERE status = 'active'),
			(SELECT COUNT(*) FROM bookings WHERE status IN ('pending', 'confirmed', 'in_progress')),
			(SELECT COALESCE(SUM(amount), 0) FROM payments
			 WHERE status = 'paid'
			   AND created_at >= date_trunc('month', $1::timestamptz))`,
		now).Scan(&row.totalLeads, &row.activeCustomers, &row.upcomingBookings, &row.monthlyRevenue)
	if err != nil {
		return Summary{}, fmt.Errorf("load summary totals: %w", err)
	}

	counts, err := r.loadLeadsByStatus(ctx)
	if err != nil {
		return Summary{}, err
	}

	bookingCounts, err := r.loadBookingsByStatus(ctx)
	if err != nil {
		return Summary{}, err
	}

	revenue, err := r.loadRevenueByMonth(ctx, now)
	if err != nil {
		return Summary{}, err
	}

	return buildSummary(row, counts, revenue, bookingCounts), nil
}

func (r *Repository) loadLeadsByStatus(ctx context.Context) (map[string]int64, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT status, COUNT(*) FROM leads GROUP BY status`)
	if err != nil {
		return nil, fmt.Errorf("query leads by status: %w", err)
	}
	defer rows.Close()

	counts := map[string]int64{}
	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("scan lead status count: %w", err)
		}
		counts[status] = count
	}
	return counts, rows.Err()
}

func (r *Repository) loadBookingsByStatus(ctx context.Context) (map[string]int64, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT status, COUNT(*) FROM bookings GROUP BY status`)
	if err != nil {
		return nil, fmt.Errorf("query bookings by status: %w", err)
	}
	defer rows.Close()

	counts := map[string]int64{}
	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("scan booking status count: %w", err)
		}
		counts[status] = count
	}
	return counts, rows.Err()
}

func (r *Repository) loadRevenueByMonth(ctx context.Context, now time.Time) ([]MonthlyRevenue, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT m.month_start, COALESCE(SUM(p.amount), 0) AS collected
		 FROM generate_series(
			 date_trunc('month', $1::timestamptz) - interval '5 months',
			 date_trunc('month', $1::timestamptz),
			 interval '1 month'
		 ) AS m(month_start)
		 LEFT JOIN payments p
			ON p.status = 'paid'
		   AND p.created_at >= m.month_start
		   AND p.created_at < m.month_start + interval '1 month'
		 GROUP BY m.month_start
		 ORDER BY m.month_start`,
		now)
	if err != nil {
		return nil, fmt.Errorf("query revenue by month: %w", err)
	}
	defer rows.Close()

	revenue := []MonthlyRevenue{}
	for rows.Next() {
		var monthStart time.Time
		var collected float64
		if err := rows.Scan(&monthStart, &collected); err != nil {
			return nil, fmt.Errorf("scan monthly revenue: %w", err)
		}
		revenue = append(revenue, MonthlyRevenue{
			Month:     monthLabel(monthStart),
			Collected: collected,
		})
	}
	return revenue, rows.Err()
}
