package applicants

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("applicant not found")

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const applicantColumns = `id, first_name, last_name, email, phone, nationality, visa_type, status, created_at`

func scanApplicant(row pgx.Row) (Applicant, error) {
	var a Applicant
	err := row.Scan(&a.ID, &a.FirstName, &a.LastName, &a.Email, &a.Phone, &a.Nationality, &a.VisaType, &a.Status, &a.CreatedAt)
	return a, err
}

func (r *Repository) List(ctx context.Context) ([]Applicant, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+applicantColumns+` FROM applicants ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("query applicants: %w", err)
	}
	defer rows.Close()

	applicants := []Applicant{}
	for rows.Next() {
		a, err := scanApplicant(rows)
		if err != nil {
			return nil, fmt.Errorf("scan applicant: %w", err)
		}
		applicants = append(applicants, a)
	}
	return applicants, rows.Err()
}

func (r *Repository) GetByID(ctx context.Context, id int64) (Applicant, error) {
	a, err := scanApplicant(r.pool.QueryRow(ctx,
		`SELECT `+applicantColumns+` FROM applicants WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Applicant{}, ErrNotFound
	}
	if err != nil {
		return Applicant{}, fmt.Errorf("get applicant %d: %w", id, err)
	}
	return a, nil
}

func (r *Repository) Create(ctx context.Context, a Applicant) (Applicant, error) {
	created, err := scanApplicant(r.pool.QueryRow(ctx,
		`INSERT INTO applicants (first_name, last_name, email, phone, nationality, visa_type, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING `+applicantColumns,
		a.FirstName, a.LastName, a.Email, a.Phone, a.Nationality, a.VisaType, a.Status))
	if err != nil {
		return Applicant{}, fmt.Errorf("create applicant: %w", err)
	}
	return created, nil
}

// Patch carries only the fields that should change; nil means "leave as is".
type Patch struct {
	FirstName   *string
	LastName    *string
	Email       *string
	Phone       *string
	Nationality *string
	VisaType    *string
	Status      *Status
}

func (r *Repository) Update(ctx context.Context, id int64, p Patch) (Applicant, error) {
	var status any
	if p.Status != nil {
		status = *p.Status
	}

	updated, err := scanApplicant(r.pool.QueryRow(ctx,
		`UPDATE applicants SET
			first_name  = COALESCE($2, first_name),
			last_name   = COALESCE($3, last_name),
			email       = COALESCE($4, email),
			phone       = COALESCE($5, phone),
			nationality = COALESCE($6, nationality),
			visa_type   = COALESCE($7, visa_type),
			status      = COALESCE($8, status)
		 WHERE id = $1
		 RETURNING `+applicantColumns,
		id, p.FirstName, p.LastName, p.Email, p.Phone, p.Nationality, p.VisaType, status))
	if errors.Is(err, pgx.ErrNoRows) {
		return Applicant{}, ErrNotFound
	}
	if err != nil {
		return Applicant{}, fmt.Errorf("update applicant %d: %w", id, err)
	}
	return updated, nil
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM applicants WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete applicant %d: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
