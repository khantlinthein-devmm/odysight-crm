package cleanerdocs

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound        = errors.New("document not found")
	ErrCleanerNotFound = errors.New("cleaner not found")
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const selectCols = `id, cleaner_id, doc_type, doc_number_enc, issue_date, expiry_date, notes,
	file_name, original_name, content_type, size_bytes, uploaded_by, reminder_stage, created_at, updated_at`

func scanDocument(row pgx.Row) (Document, error) {
	var d Document
	err := row.Scan(&d.ID, &d.CleanerID, &d.Type, &d.NumberEnc, &d.IssueDate, &d.ExpiryDate, &d.Notes,
		&d.FileName, &d.OriginalName, &d.ContentType, &d.SizeBytes, &d.UploadedBy, &d.ReminderStage,
		&d.CreatedAt, &d.UpdatedAt)
	return d, err
}

func (r *Repository) CleanerExists(ctx context.Context, cleanerID int64) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM cleaners WHERE id = $1)`, cleanerID).Scan(&ok)
	if err != nil {
		return false, fmt.Errorf("check cleaner %d: %w", cleanerID, err)
	}
	return ok, nil
}

func (r *Repository) ListByCleaner(ctx context.Context, cleanerID int64) ([]Document, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+selectCols+` FROM cleaner_documents
		WHERE cleaner_id = $1 ORDER BY expiry_date ASC NULLS LAST, id`, cleanerID)
	if err != nil {
		return nil, fmt.Errorf("list cleaner documents: %w", err)
	}
	defer rows.Close()
	out := []Document{}
	for rows.Next() {
		d, err := scanDocument(rows)
		if err != nil {
			return nil, fmt.Errorf("scan cleaner document: %w", err)
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *Repository) Get(ctx context.Context, id int64) (Document, error) {
	d, err := scanDocument(r.pool.QueryRow(ctx, `SELECT `+selectCols+` FROM cleaner_documents WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Document{}, ErrNotFound
	}
	if err != nil {
		return Document{}, fmt.Errorf("get cleaner document %d: %w", id, err)
	}
	return d, nil
}

func (r *Repository) Create(ctx context.Context, d Document) (Document, error) {
	created, err := scanDocument(r.pool.QueryRow(ctx,
		`INSERT INTO cleaner_documents (cleaner_id, doc_type, doc_number_enc, issue_date, expiry_date, notes,
			file_name, original_name, content_type, size_bytes, uploaded_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING `+selectCols,
		d.CleanerID, d.Type, d.NumberEnc, d.IssueDate, d.ExpiryDate, d.Notes,
		d.FileName, d.OriginalName, d.ContentType, d.SizeBytes, d.UploadedBy))
	if err != nil {
		return Document{}, fmt.Errorf("create cleaner document: %w", err)
	}
	return created, nil
}

// Update writes every editable column of d (the service does
// read-modify-write so nullable dates can be cleared).
func (r *Repository) Update(ctx context.Context, d Document) (Document, error) {
	updated, err := scanDocument(r.pool.QueryRow(ctx,
		`UPDATE cleaner_documents SET doc_type = $2, doc_number_enc = $3, issue_date = $4, expiry_date = $5,
			notes = $6, reminder_stage = $7
		WHERE id = $1 RETURNING `+selectCols,
		d.ID, d.Type, d.NumberEnc, d.IssueDate, d.ExpiryDate, d.Notes, d.ReminderStage))
	if errors.Is(err, pgx.ErrNoRows) {
		return Document{}, ErrNotFound
	}
	if err != nil {
		return Document{}, fmt.Errorf("update cleaner document %d: %w", d.ID, err)
	}
	return updated, nil
}

// Delete removes the row and returns its stored file name for cleanup.
func (r *Repository) Delete(ctx context.Context, id int64) (string, error) {
	var fileName string
	err := r.pool.QueryRow(ctx, `DELETE FROM cleaner_documents WHERE id = $1 RETURNING file_name`, id).Scan(&fileName)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("delete cleaner document %d: %w", id, err)
	}
	return fileName, nil
}

// Expiring lists documents of non-inactive cleaners expiring on or before
// until (already-expired ones included), soonest first.
func (r *Repository) Expiring(ctx context.Context, until time.Time) ([]ExpiringItem, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT d.id, d.cleaner_id, trim(c.first_name || ' ' || c.last_name), d.doc_type, d.expiry_date, d.reminder_stage
		FROM cleaner_documents d
		JOIN cleaners c ON c.id = d.cleaner_id
		WHERE d.expiry_date IS NOT NULL AND d.expiry_date <= $1 AND c.status <> 'inactive'
		ORDER BY d.expiry_date, d.id`, until)
	if err != nil {
		return nil, fmt.Errorf("list expiring documents: %w", err)
	}
	defer rows.Close()
	out := []ExpiringItem{}
	for rows.Next() {
		var e ExpiringItem
		if err := rows.Scan(&e.DocumentID, &e.CleanerID, &e.CleanerName, &e.Type, &e.ExpiryDate, &e.ReminderStage); err != nil {
			return nil, fmt.Errorf("scan expiring document: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *Repository) SetReminderStage(ctx context.Context, id int64, stage int16) error {
	_, err := r.pool.Exec(ctx, `UPDATE cleaner_documents SET reminder_stage = $2 WHERE id = $1`, id, stage)
	if err != nil {
		return fmt.Errorf("stamp reminder stage %d: %w", id, err)
	}
	return nil
}

// FileNames returns every stored file name still referenced by a row.
func (r *Repository) FileNames(ctx context.Context) (map[string]bool, error) {
	rows, err := r.pool.Query(ctx, `SELECT file_name FROM cleaner_documents WHERE file_name <> ''`)
	if err != nil {
		return nil, fmt.Errorf("list document files: %w", err)
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		out[n] = true
	}
	return out, rows.Err()
}

// LogAccess records a read of sensitive data in audit_logs. The generic audit
// middleware only covers mutations, so document views are logged explicitly.
func (r *Repository) LogAccess(ctx context.Context, userID int64, action, resource string, resourceID int64) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO audit_logs (user_id, action, resource, resource_id) VALUES ($1, $2, $3, $4)`,
		userID, action, resource, resourceID)
	return err
}
