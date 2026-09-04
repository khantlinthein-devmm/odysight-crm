package servicerecords

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("service record not found")

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const recordColumns = `id, booking_number, cleaner_name, service_type, rating, status, notes, completed_at, created_at`

func scanRecord(row pgx.Row) (ServiceRecord, error) {
	var r ServiceRecord
	err := row.Scan(&r.ID, &r.BookingNumber, &r.CleanerName, &r.ServiceType,
		&r.Rating, &r.Status, &r.Notes, &r.CompletedAt, &r.CreatedAt)
	return r, err
}

func (r *Repository) List(ctx context.Context) ([]ServiceRecord, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+recordColumns+` FROM service_records ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("query service records: %w", err)
	}
	defer rows.Close()

	records := []ServiceRecord{}
	for rows.Next() {
		rec, err := scanRecord(rows)
		if err != nil {
			return nil, fmt.Errorf("scan service record: %w", err)
		}
		records = append(records, rec)
	}
	return records, rows.Err()
}

func (r *Repository) GetByID(ctx context.Context, id int64) (ServiceRecord, error) {
	rec, err := scanRecord(r.pool.QueryRow(ctx,
		`SELECT `+recordColumns+` FROM service_records WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return ServiceRecord{}, ErrNotFound
	}
	if err != nil {
		return ServiceRecord{}, fmt.Errorf("get service record %d: %w", id, err)
	}
	return rec, nil
}

func (r *Repository) Create(ctx context.Context, rec ServiceRecord) (ServiceRecord, error) {
	created, err := scanRecord(r.pool.QueryRow(ctx,
		`INSERT INTO service_records (booking_number, cleaner_name, service_type, rating, status, notes)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING `+recordColumns,
		rec.BookingNumber, rec.CleanerName, rec.ServiceType, rec.Rating, rec.Status, rec.Notes))
	if err != nil {
		return ServiceRecord{}, fmt.Errorf("create service record: %w", err)
	}
	return created, nil
}

type Patch struct {
	BookingNumber *string
	CleanerName   *string
	ServiceType   *string
	Rating        *int
	Status        *Status
	Notes         *string
}

func (r *Repository) Update(ctx context.Context, id int64, p Patch) (ServiceRecord, error) {
	var status any
	if p.Status != nil {
		status = *p.Status
	}

	updated, err := scanRecord(r.pool.QueryRow(ctx,
		`UPDATE service_records SET
			booking_number = COALESCE($2, booking_number),
			cleaner_name   = COALESCE($3, cleaner_name),
			service_type   = COALESCE($4, service_type),
			rating         = COALESCE($5, rating),
			status         = COALESCE($6, status),
			notes          = COALESCE($7, notes),
			completed_at   = CASE WHEN $6::text = 'completed' AND completed_at IS NULL THEN now() ELSE completed_at END
		 WHERE id = $1
		 RETURNING `+recordColumns,
		id, p.BookingNumber, p.CleanerName, p.ServiceType, p.Rating, status, p.Notes))
	if errors.Is(err, pgx.ErrNoRows) {
		return ServiceRecord{}, ErrNotFound
	}
	if err != nil {
		return ServiceRecord{}, fmt.Errorf("update service record %d: %w", id, err)
	}
	return updated, nil
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM service_records WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete service record %d: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
