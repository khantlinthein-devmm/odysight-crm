package reports

import (
	"context"
	"fmt"
)

type profitBooking struct {
	ID              int64
	Price           *float64
	DurationMinutes int
}

// ProfitInputs loads everything the booking-profit estimate needs for ids:
// price and duration, the assigned team, every pay rate, and supply usage
// recorded against each booking.
func (r *Repository) ProfitInputs(ctx context.Context, ids []int64) (
	[]profitBooking, map[int64][]TeamMember, map[int64]PayRate, map[int64]float64, error) {
	bookings := []profitBooking{}
	rows, err := r.pool.Query(ctx,
		`SELECT id, price::float8, duration_minutes FROM bookings WHERE id = ANY($1) ORDER BY id`, ids)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("profit bookings: %w", err)
	}
	for rows.Next() {
		var b profitBooking
		if err := rows.Scan(&b.ID, &b.Price, &b.DurationMinutes); err != nil {
			rows.Close()
			return nil, nil, nil, nil, err
		}
		bookings = append(bookings, b)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, nil, nil, nil, err
	}

	team := map[int64][]TeamMember{}
	rows, err = r.pool.Query(ctx,
		`SELECT bc.booking_id, c.id, TRIM(c.first_name || ' ' || c.last_name), bc.role
		   FROM booking_cleaners bc JOIN cleaners c ON c.id = bc.cleaner_id
		  WHERE bc.booking_id = ANY($1)
		  ORDER BY bc.booking_id, (bc.role = 'primary') DESC, bc.id`, ids)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("profit teams: %w", err)
	}
	for rows.Next() {
		var bookingID int64
		var m TeamMember
		if err := rows.Scan(&bookingID, &m.CleanerID, &m.Name, &m.Role); err != nil {
			rows.Close()
			return nil, nil, nil, nil, err
		}
		team[bookingID] = append(team[bookingID], m)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, nil, nil, nil, err
	}

	rates := map[int64]PayRate{}
	rows, err = r.pool.Query(ctx,
		`SELECT cleaner_id, pay_type, rate::float8, standard_hours::float8 FROM cleaner_pay_rates`)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("profit pay rates: %w", err)
	}
	for rows.Next() {
		var id int64
		var rt PayRate
		if err := rows.Scan(&id, &rt.PayType, &rt.Rate, &rt.StandardHours); err != nil {
			rows.Close()
			return nil, nil, nil, nil, err
		}
		rates[id] = rt
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, nil, nil, nil, err
	}

	// Usage movements store a negative quantity (stock going out).
	supplies := map[int64]float64{}
	rows, err = r.pool.Query(ctx,
		`SELECT booking_id, SUM(-quantity * unit_cost)::float8
		   FROM supply_movements
		  WHERE kind = 'usage' AND booking_id = ANY($1)
		  GROUP BY booking_id`, ids)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("profit supplies: %w", err)
	}
	for rows.Next() {
		var id int64
		var cost float64
		if err := rows.Scan(&id, &cost); err != nil {
			rows.Close()
			return nil, nil, nil, nil, err
		}
		supplies[id] = cost
	}
	rows.Close()
	return bookings, team, rates, supplies, rows.Err()
}
