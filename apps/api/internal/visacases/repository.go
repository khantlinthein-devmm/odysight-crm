package visacases

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("visa case not found")

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const caseColumns = `id, case_number, applicant_name, visa_type, destination, assigned_to, status, created_at`

const caseNumberExpr = `'VC-' || to_char(created_at, 'YYYY') || '-' || lpad(id::text, 4, '0')`

func scanCase(row pgx.Row) (VisaCase, error) {
	var c VisaCase
	err := row.Scan(&c.ID, &c.CaseNumber, &c.ApplicantName, &c.VisaType, &c.Destination, &c.AssignedTo, &c.Status, &c.CreatedAt)
	return c, err
}

func (r *Repository) List(ctx context.Context) ([]VisaCase, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+caseColumns+` FROM visa_cases ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("query visa cases: %w", err)
	}
	defer rows.Close()

	cases := []VisaCase{}
	for rows.Next() {
		c, err := scanCase(rows)
		if err != nil {
			return nil, fmt.Errorf("scan visa case: %w", err)
		}
		cases = append(cases, c)
	}
	return cases, rows.Err()
}

func (r *Repository) GetByID(ctx context.Context, id int64) (VisaCase, error) {
	c, err := scanCase(r.pool.QueryRow(ctx,
		`SELECT `+caseColumns+` FROM visa_cases WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return VisaCase{}, ErrNotFound
	}
	if err != nil {
		return VisaCase{}, fmt.Errorf("get visa case %d: %w", id, err)
	}
	return c, nil
}

// Create inserts the case and assigns the case number from the new row's id.
func (r *Repository) Create(ctx context.Context, c VisaCase) (VisaCase, error) {
	var id int64
	if err := r.pool.QueryRow(ctx,
		`INSERT INTO visa_cases (case_number, applicant_name, visa_type, destination, assigned_to, status)
		 VALUES ('', $1, $2, $3, $4, $5)
		 RETURNING id`,
		c.ApplicantName, c.VisaType, c.Destination, c.AssignedTo, c.Status).Scan(&id); err != nil {
		return VisaCase{}, fmt.Errorf("create visa case: %w", err)
	}

	created, err := scanCase(r.pool.QueryRow(ctx,
		`UPDATE visa_cases SET case_number = `+caseNumberExpr+`
		 WHERE id = $1
		 RETURNING `+caseColumns,
		id))
	if err != nil {
		return VisaCase{}, fmt.Errorf("assign case number to visa case %d: %w", id, err)
	}
	return created, nil
}

// Patch carries only the fields that should change; nil means "leave as is".
type Patch struct {
	ApplicantName *string
	VisaType      *string
	Destination   *string
	AssignedTo    *string
	Status        *Status
}

func (r *Repository) Update(ctx context.Context, id int64, p Patch) (VisaCase, error) {
	var status any
	if p.Status != nil {
		status = *p.Status
	}

	updated, err := scanCase(r.pool.QueryRow(ctx,
		`UPDATE visa_cases SET
			applicant_name = COALESCE($2, applicant_name),
			visa_type      = COALESCE($3, visa_type),
			destination    = COALESCE($4, destination),
			assigned_to    = COALESCE($5, assigned_to),
			status         = COALESCE($6, status)
		 WHERE id = $1
		 RETURNING `+caseColumns,
		id, p.ApplicantName, p.VisaType, p.Destination, p.AssignedTo, status))
	if errors.Is(err, pgx.ErrNoRows) {
		return VisaCase{}, ErrNotFound
	}
	if err != nil {
		return VisaCase{}, fmt.Errorf("update visa case %d: %w", id, err)
	}
	return updated, nil
}
