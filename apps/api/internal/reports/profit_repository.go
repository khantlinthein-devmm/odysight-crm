package reports

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/odysight/crm/pkg/response"
)

type profitBooking struct {
	ID    int64
	Price *float64
}

// ProfitInputs loads everything the booking-profit estimate needs for ids:
// price, the assigned team, the office-entered costs and supply usage
// recorded against each booking.
func (r *Repository) ProfitInputs(ctx context.Context, ids []int64) (
	[]profitBooking, map[int64][]TeamMember, map[int64]BookingCosts, map[int64]float64, error) {
	bookings := []profitBooking{}
	rows, err := r.pool.Query(ctx, `SELECT id, price::float8 FROM bookings WHERE id = ANY($1) ORDER BY id`, ids)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("profit bookings: %w", err)
	}
	for rows.Next() {
		var b profitBooking
		if err := rows.Scan(&b.ID, &b.Price); err != nil {
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

	costs := map[int64]BookingCosts{}
	rows, err = r.pool.Query(ctx,
		`SELECT booking_id, cleaner_fees, transport::float8, other::float8, other_note
		   FROM booking_costs WHERE booking_id = ANY($1)`, ids)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("profit costs: %w", err)
	}
	for rows.Next() {
		var id int64
		var raw []byte
		var c BookingCosts
		if err := rows.Scan(&id, &raw, &c.Transport, &c.Other, &c.OtherNote); err != nil {
			rows.Close()
			return nil, nil, nil, nil, err
		}
		fees, err := decodeFees(raw)
		if err != nil {
			rows.Close()
			return nil, nil, nil, nil, fmt.Errorf("booking %d cleaner fees: %w", id, err)
		}
		c.CleanerFees = fees
		costs[id] = c
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
	return bookings, team, costs, supplies, rows.Err()
}

func decodeFees(raw []byte) (map[int64]float64, error) {
	byKey := map[string]float64{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &byKey); err != nil {
			return nil, err
		}
	}
	out := make(map[int64]float64, len(byKey))
	for k, v := range byKey {
		if id, err := strconv.ParseInt(k, 10, 64); err == nil {
			out[id] = v
		}
	}
	return out, nil
}

// SaveBookingCosts upserts the office-entered costs of one booking.
func (r *Repository) SaveBookingCosts(ctx context.Context, bookingID, userID int64, c BookingCosts) error {
	byKey := make(map[string]float64, len(c.CleanerFees))
	for id, fee := range c.CleanerFees {
		byKey[strconv.FormatInt(id, 10)] = fee
	}
	fees, err := json.Marshal(byKey)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx,
		`INSERT INTO booking_costs (booking_id, cleaner_fees, transport, other, other_note, updated_by, updated_at)
		 VALUES ($1, $2::jsonb, $3, $4, $5, $6, now())
		 ON CONFLICT (booking_id) DO UPDATE SET
		   cleaner_fees = EXCLUDED.cleaner_fees, transport = EXCLUDED.transport, other = EXCLUDED.other,
		   other_note = EXCLUDED.other_note, updated_by = EXCLUDED.updated_by, updated_at = now()`,
		bookingID, string(fees), c.Transport, c.Other, c.OtherNote, userID)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23503" {
		return response.NewAPIError(404, "booking not found")
	}
	if err != nil {
		return fmt.Errorf("save booking %d costs: %w", bookingID, err)
	}
	return nil
}
