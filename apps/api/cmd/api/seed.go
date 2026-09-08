package main

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/odysight/crm/internal/bookings"
)

const (
	demoPortalEmail    = "somchai@smileclean.com"
	demoPortalPassword = "portal123"
)

// seedPortalDemoCustomer creates (or upgrades) a demo customer that can sign
// in to the customer portal, plus a couple of bookings to browse. Idempotent:
// on repeat boots it simply re-enables portal access and refreshes the demo
// password without duplicating the bookings.
func seedPortalDemoCustomer(ctx context.Context, pool *pgxpool.Pool, bookingRepo *bookings.Repository) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(demoPortalPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	var id int64
	err = pool.QueryRow(ctx, `SELECT id FROM customers WHERE email = $1`, demoPortalEmail).Scan(&id)
	switch {
	case err == nil:
		if _, err := pool.Exec(ctx,
			`UPDATE customers SET portal_enabled = true, password_hash = $2 WHERE id = $1`,
			id, string(hash)); err != nil {
			return err
		}
	case errors.Is(err, pgx.ErrNoRows):
		if err := pool.QueryRow(ctx,
			`INSERT INTO customers (first_name, last_name, email, phone, address, property_type, area, status, portal_enabled, password_hash)
			 VALUES ('Somchai', 'Prasert', $1, '+66 812345678', 'Sukhumvit 38, Bangkok', 'condo', 'Sukhumvit', 'active', true, $2)
			 RETURNING id`, demoPortalEmail, string(hash)).Scan(&id); err != nil {
			return err
		}
	default:
		return err
	}

	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM bookings WHERE customer_id = $1`, id).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	now := time.Now()
	completed, err := bookingRepo.Create(ctx, bookings.Booking{
		CustomerName:    "Somchai Prasert",
		CustomerEmail:   demoPortalEmail,
		CustomerID:      &id,
		ServiceType:     bookings.SvcCondoCleaning,
		ScheduledFor:    now.AddDate(0, 0, -7),
		DurationMinutes: 120,
		Address:         "Sukhumvit 38, Bangkok",
		Status:          bookings.StatusCompleted,
	})
	if err != nil {
		return err
	}
	if _, err := pool.Exec(ctx,
		`UPDATE bookings SET assigned_cleaner = 'Nok Srisuwan' WHERE id = $1`, completed.ID); err != nil {
		return err
	}

	_, err = bookingRepo.Create(ctx, bookings.Booking{
		CustomerName:    "Somchai Prasert",
		CustomerEmail:   demoPortalEmail,
		CustomerID:      &id,
		ServiceType:     bookings.SvcDeepCleaning,
		ScheduledFor:    now.AddDate(0, 0, 5),
		DurationMinutes: 240,
		Address:         "Sukhumvit 38, Bangkok",
		Notes:           "Please bring extra cloths for the kitchen.",
		Status:          bookings.StatusConfirmed,
	})
	return err
}