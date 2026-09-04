package cleaners

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("cleaner not found")

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const cleanerColumns = `id, first_name, last_name, phone, email, skills, status, created_at`

func scanCleaner(row pgx.Row) (Cleaner, error) {
	var c Cleaner
	err := row.Scan(&c.ID, &c.FirstName, &c.LastName, &c.Phone, &c.Email, &c.Skills, &c.Status, &c.CreatedAt)
	return c, err
}

func (r *Repository) List(ctx context.Context) ([]Cleaner, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+cleanerColumns+` FROM cleaners ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("query cleaners: %w", err)
	}
	defer rows.Close()

	cleaners := []Cleaner{}
	for rows.Next() {
		c, err := scanCleaner(rows)
		if err != nil {
			return nil, fmt.Errorf("scan cleaner: %w", err)
		}
		cleaners = append(cleaners, c)
	}
	return cleaners, rows.Err()
}

func (r *Repository) GetByID(ctx context.Context, id int64) (Cleaner, error) {
	c, err := scanCleaner(r.pool.QueryRow(ctx,
		`SELECT `+cleanerColumns+` FROM cleaners WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Cleaner{}, ErrNotFound
	}
	if err != nil {
		return Cleaner{}, fmt.Errorf("get cleaner %d: %w", id, err)
	}
	return c, nil
}

func (r *Repository) Create(ctx context.Context, c Cleaner) (Cleaner, error) {
	created, err := scanCleaner(r.pool.QueryRow(ctx,
		`INSERT INTO cleaners (first_name, last_name, phone, email, skills, status)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING `+cleanerColumns,
		c.FirstName, c.LastName, c.Phone, c.Email, c.Skills, c.Status))
	if err != nil {
		return Cleaner{}, fmt.Errorf("create cleaner: %w", err)
	}
	return created, nil
}

type Patch struct {
	FirstName *string
	LastName  *string
	Phone     *string
	Email     *string
	Skills    *string
	Status    *Status
}

func (r *Repository) Update(ctx context.Context, id int64, p Patch) (Cleaner, error) {
	var status any
	if p.Status != nil {
		status = *p.Status
	}

	updated, err := scanCleaner(r.pool.QueryRow(ctx,
		`UPDATE cleaners SET
			first_name = COALESCE($2, first_name),
			last_name  = COALESCE($3, last_name),
			phone      = COALESCE($4, phone),
			email      = COALESCE($5, email),
			skills     = COALESCE($6, skills),
			status     = COALESCE($7, status)
		 WHERE id = $1
		 RETURNING `+cleanerColumns,
		id, p.FirstName, p.LastName, p.Phone, p.Email, p.Skills, status))
	if errors.Is(err, pgx.ErrNoRows) {
		return Cleaner{}, ErrNotFound
	}
	if err != nil {
		return Cleaner{}, fmt.Errorf("update cleaner %d: %w", id, err)
	}
	return updated, nil
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM cleaners WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete cleaner %d: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
