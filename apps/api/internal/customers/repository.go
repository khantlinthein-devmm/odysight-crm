package customers

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("customer not found")

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const customerColumns = `id, first_name, last_name, email, phone, address, property_type, area, status, created_at`

func scanCustomer(row pgx.Row) (Customer, error) {
	var c Customer
	err := row.Scan(&c.ID, &c.FirstName, &c.LastName, &c.Email, &c.Phone,
		&c.Address, &c.PropertyType, &c.Area, &c.Status, &c.CreatedAt)
	return c, err
}

func (r *Repository) List(ctx context.Context) ([]Customer, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+customerColumns+` FROM customers ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("query customers: %w", err)
	}
	defer rows.Close()

	customers := []Customer{}
	for rows.Next() {
		c, err := scanCustomer(rows)
		if err != nil {
			return nil, fmt.Errorf("scan customer: %w", err)
		}
		customers = append(customers, c)
	}
	return customers, rows.Err()
}

func (r *Repository) GetByID(ctx context.Context, id int64) (Customer, error) {
	c, err := scanCustomer(r.pool.QueryRow(ctx,
		`SELECT `+customerColumns+` FROM customers WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Customer{}, ErrNotFound
	}
	if err != nil {
		return Customer{}, fmt.Errorf("get customer %d: %w", id, err)
	}
	return c, nil
}

func (r *Repository) Create(ctx context.Context, c Customer) (Customer, error) {
	created, err := scanCustomer(r.pool.QueryRow(ctx,
		`INSERT INTO customers (first_name, last_name, email, phone, address, property_type, area, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING `+customerColumns,
		c.FirstName, c.LastName, c.Email, c.Phone, c.Address, c.PropertyType, c.Area, c.Status))
	if err != nil {
		return Customer{}, fmt.Errorf("create customer: %w", err)
	}
	return created, nil
}

type Patch struct {
	FirstName    *string
	LastName     *string
	Email        *string
	Phone        *string
	Address      *string
	PropertyType *PropertyType
	Area         *string
	Status       *Status
}

func (r *Repository) Update(ctx context.Context, id int64, p Patch) (Customer, error) {
	var propType any
	if p.PropertyType != nil {
		propType = *p.PropertyType
	}
	var status any
	if p.Status != nil {
		status = *p.Status
	}

	updated, err := scanCustomer(r.pool.QueryRow(ctx,
		`UPDATE customers SET
			first_name    = COALESCE($2, first_name),
			last_name     = COALESCE($3, last_name),
			email         = COALESCE($4, email),
			phone         = COALESCE($5, phone),
			address       = COALESCE($6, address),
			property_type = COALESCE($7, property_type),
			area          = COALESCE($8, area),
			status        = COALESCE($9, status)
		 WHERE id = $1
		 RETURNING `+customerColumns,
		id, p.FirstName, p.LastName, p.Email, p.Phone, p.Address, propType, p.Area, status))
	if errors.Is(err, pgx.ErrNoRows) {
		return Customer{}, ErrNotFound
	}
	if err != nil {
		return Customer{}, fmt.Errorf("update customer %d: %w", id, err)
	}
	return updated, nil
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM customers WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete customer %d: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
