package documents

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("document not found")

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const documentColumns = `id, name, type, applicant_name, file_size_kb, status, uploaded_at`

func scanDocument(row pgx.Row) (Document, error) {
	var d Document
	err := row.Scan(&d.ID, &d.Name, &d.Type, &d.ApplicantName, &d.FileSizeKb, &d.Status, &d.UploadedAt)
	return d, err
}

func (r *Repository) List(ctx context.Context) ([]Document, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+documentColumns+` FROM documents ORDER BY uploaded_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("query documents: %w", err)
	}
	defer rows.Close()

	documents := []Document{}
	for rows.Next() {
		d, err := scanDocument(rows)
		if err != nil {
			return nil, fmt.Errorf("scan document: %w", err)
		}
		documents = append(documents, d)
	}
	return documents, rows.Err()
}

func (r *Repository) GetByID(ctx context.Context, id int64) (Document, error) {
	d, err := scanDocument(r.pool.QueryRow(ctx,
		`SELECT `+documentColumns+` FROM documents WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Document{}, ErrNotFound
	}
	if err != nil {
		return Document{}, fmt.Errorf("get document %d: %w", id, err)
	}
	return d, nil
}

func (r *Repository) Create(ctx context.Context, d Document) (Document, error) {
	created, err := scanDocument(r.pool.QueryRow(ctx,
		`INSERT INTO documents (name, type, applicant_name, file_size_kb, status)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING `+documentColumns,
		d.Name, d.Type, d.ApplicantName, d.FileSizeKb, d.Status))
	if err != nil {
		return Document{}, fmt.Errorf("create document: %w", err)
	}
	return created, nil
}

// Patch carries only the fields that should change; nil means "leave as is".
type Patch struct {
	Name   *string
	Type   *string
	Status *Status
}

func (r *Repository) Update(ctx context.Context, id int64, p Patch) (Document, error) {
	var status any
	if p.Status != nil {
		status = *p.Status
	}

	updated, err := scanDocument(r.pool.QueryRow(ctx,
		`UPDATE documents SET
			name   = COALESCE($2, name),
			type   = COALESCE($3, type),
			status = COALESCE($4, status)
		 WHERE id = $1
		 RETURNING `+documentColumns,
		id, p.Name, p.Type, status))
	if errors.Is(err, pgx.ErrNoRows) {
		return Document{}, ErrNotFound
	}
	if err != nil {
		return Document{}, fmt.Errorf("update document %d: %w", id, err)
	}
	return updated, nil
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM documents WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete document %d: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
