package checklists

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/odysight/crm/pkg/pagination"
)

var (
	ErrNotFound         = errors.New("checklist not found")
	ErrTemplateNotFound = errors.New("checklist template not found")
	ErrItemNotFound     = errors.New("checklist item not found")
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Templates

func (r *Repository) ListTemplates(ctx context.Context, params pagination.Params) ([]Template, int, error) {
	var total int
	where := ""
	args := []any{}
	if params.Search != "" {
		args = append(args, "%"+params.Search+"%")
		where = "WHERE name ILIKE $1 OR service_type ILIKE $1"
	}
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM checklist_templates `+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count templates: %w", err)
	}
	args = append(args, params.Limit, params.Offset)
	query := `SELECT id, name, service_type, is_active, created_at, updated_at FROM checklist_templates ` + where +
		fmt.Sprintf(` ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, len(args)-1, len(args))
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query templates: %w", err)
	}
	defer rows.Close()

	items := []Template{}
	for rows.Next() {
		var t Template
		if err := rows.Scan(&t.ID, &t.Name, &t.ServiceType, &t.IsActive, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan template: %w", err)
		}
		items = append(items, t)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	ids := make([]int64, 0, len(items))
	for _, t := range items {
		ids = append(ids, t.ID)
	}
	itemMap, err := r.templateItems(ctx, r.pool, ids)
	if err != nil {
		return nil, 0, err
	}
	for i := range items {
		items[i].Items = itemMap[items[i].ID]
		if items[i].Items == nil {
			items[i].Items = []TemplateItem{}
		}
	}
	return items, total, nil
}

type pgxQuerier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func (r *Repository) templateItems(ctx context.Context, q pgxQuerier, ids []int64) (map[int64][]TemplateItem, error) {
	out := make(map[int64][]TemplateItem, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := q.Query(ctx, `SELECT id, template_id, label, sort_order FROM checklist_template_items WHERE template_id = ANY($1) ORDER BY sort_order, id`, ids)
	if err != nil {
		return nil, fmt.Errorf("load template items: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id, tid int64
		var label string
		var sort int
		if err := rows.Scan(&id, &tid, &label, &sort); err != nil {
			return nil, fmt.Errorf("scan template item: %w", err)
		}
		out[tid] = append(out[tid], TemplateItem{ID: id, Label: label, SortOrder: sort})
	}
	return out, rows.Err()
}

func (r *Repository) CreateTemplate(ctx context.Context, name, serviceType string, labels []string) (Template, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Template{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var t Template
	if err := tx.QueryRow(ctx,
		`INSERT INTO checklist_templates (name, service_type) VALUES ($1, $2) RETURNING id, name, service_type, is_active, created_at, updated_at`,
		name, serviceType).Scan(&t.ID, &t.Name, &t.ServiceType, &t.IsActive, &t.CreatedAt, &t.UpdatedAt); err != nil {
		return Template{}, fmt.Errorf("create template: %w", err)
	}
	for i, label := range labels {
		if _, err := tx.Exec(ctx, `INSERT INTO checklist_template_items (template_id, label, sort_order) VALUES ($1, $2, $3)`, t.ID, label, i); err != nil {
			return Template{}, fmt.Errorf("create template item: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return Template{}, fmt.Errorf("commit template: %w", err)
	}
	itemMap, err := r.templateItems(ctx, r.pool, []int64{t.ID})
	if err != nil {
		return Template{}, err
	}
	t.Items = itemMap[t.ID]
	return t, nil
}

// Checklists

func (r *Repository) GetByBooking(ctx context.Context, bookingID int64) (Checklist, error) {
	var c Checklist
	err := r.pool.QueryRow(ctx,
		`SELECT id, booking_id, template_id, status, client_signature, client_confirmed_at, created_at, updated_at
		 FROM booking_checklists WHERE booking_id = $1`, bookingID).
		Scan(&c.ID, &c.BookingID, &c.TemplateID, &c.Status, &c.ClientSignature, &c.ClientConfirmedAt, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Checklist{}, ErrNotFound
	}
	if err != nil {
		return Checklist{}, fmt.Errorf("get checklist for booking %d: %w", bookingID, err)
	}
	items, err := r.checklistItems(ctx, c.ID)
	if err != nil {
		return Checklist{}, err
	}
	c.Items = items
	return c, nil
}

func (r *Repository) checklistItems(ctx context.Context, checklistID int64) ([]ChecklistItem, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, checklist_id, label, is_completed, completed_by, completed_at, notes, before_photo_url, after_photo_url, sort_order
		 FROM booking_checklist_items WHERE checklist_id = $1 ORDER BY sort_order, id`, checklistID)
	if err != nil {
		return nil, fmt.Errorf("load checklist items: %w", err)
	}
	defer rows.Close()
	items := []ChecklistItem{}
	for rows.Next() {
		var it ChecklistItem
		if err := rows.Scan(&it.ID, &it.ChecklistID, &it.Label, &it.IsCompleted, &it.CompletedBy, &it.CompletedAt, &it.Notes, &it.BeforePhotoURL, &it.AfterPhotoURL, &it.SortOrder); err != nil {
			return nil, fmt.Errorf("scan checklist item: %w", err)
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

func (r *Repository) Create(ctx context.Context, bookingID int64, templateID *int64, labels []string) (Checklist, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Checklist{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if templateID != nil && len(labels) == 0 {
		rows, err := tx.Query(ctx, `SELECT label FROM checklist_template_items WHERE template_id = $1 ORDER BY sort_order, id`, *templateID)
		if err != nil {
			return Checklist{}, fmt.Errorf("load template items: %w", err)
		}
		for rows.Next() {
			var label string
			if err := rows.Scan(&label); err != nil {
				rows.Close()
				return Checklist{}, fmt.Errorf("scan template item: %w", err)
			}
			labels = append(labels, label)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return Checklist{}, err
		}
		if len(labels) == 0 {
			return Checklist{}, ErrTemplateNotFound
		}
	}
	var c Checklist
	err = tx.QueryRow(ctx,
		`INSERT INTO booking_checklists (booking_id, template_id, status) VALUES ($1, $2, 'pending')
		 ON CONFLICT (booking_id) DO NOTHING RETURNING id, booking_id, template_id, status, client_signature, client_confirmed_at, created_at, updated_at`,
		bookingID, templateID).Scan(&c.ID, &c.BookingID, &c.TemplateID, &c.Status, &c.ClientSignature, &c.ClientConfirmedAt, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Checklist{}, fmt.Errorf("booking already has a checklist")
	}
	if err != nil {
		return Checklist{}, fmt.Errorf("create checklist: %w", err)
	}
	for i, label := range labels {
		if _, err := tx.Exec(ctx, `INSERT INTO booking_checklist_items (checklist_id, label, sort_order) VALUES ($1, $2, $3)`, c.ID, strings.TrimSpace(label), i); err != nil {
			return Checklist{}, fmt.Errorf("create checklist item: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return Checklist{}, fmt.Errorf("commit checklist: %w", err)
	}
	items, err := r.checklistItems(ctx, c.ID)
	if err != nil {
		return Checklist{}, err
	}
	c.Items = items
	return c, nil
}

func (r *Repository) CompleteItem(ctx context.Context, itemID int64, completed bool, completedBy *string, notes *string, beforeURL *string, afterURL *string) (Checklist, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Checklist{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var checklistID int64
	var bookingID int64
	err = tx.QueryRow(ctx,
		`SELECT bci.checklist_id, bc.booking_id FROM booking_checklist_items bci
		 JOIN booking_checklists bc ON bc.id = bci.checklist_id WHERE bci.id = $1`, itemID).
		Scan(&checklistID, &bookingID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Checklist{}, ErrItemNotFound
	}
	if err != nil {
		return Checklist{}, fmt.Errorf("get checklist item %d: %w", itemID, err)
	}
	var completedAt any
	if completed {
		completedAt = time.Now()
	}
	if _, err := tx.Exec(ctx,
		`UPDATE booking_checklist_items SET is_completed = $2, completed_by = COALESCE($3, completed_by),
		 completed_at = $4, notes = COALESCE($5, notes),
		 before_photo_url = COALESCE($6, before_photo_url), after_photo_url = COALESCE($7, after_photo_url)
		 WHERE id = $1`,
		itemID, completed, completedBy, completedAt, notes, beforeURL, afterURL); err != nil {
		return Checklist{}, fmt.Errorf("complete item %d: %w", itemID, err)
	}
	var remaining int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM booking_checklist_items WHERE checklist_id = $1 AND is_completed = FALSE`, checklistID).Scan(&remaining); err != nil {
		return Checklist{}, fmt.Errorf("count remaining items: %w", err)
	}
	status := "in_progress"
	if remaining == 0 {
		status = "completed"
	}
	if _, err := tx.Exec(ctx, `UPDATE booking_checklists SET status = $2 WHERE id = $1`, checklistID, status); err != nil {
		return Checklist{}, fmt.Errorf("update checklist status: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Checklist{}, fmt.Errorf("commit item %d: %w", itemID, err)
	}
	return r.GetByBooking(ctx, bookingID)
}

func (r *Repository) Confirm(ctx context.Context, bookingID int64, signature string) (Checklist, error) {
	tag, err := r.pool.Exec(ctx,
		`UPDATE booking_checklists SET client_signature = $2, client_confirmed_at = now() WHERE booking_id = $1`,
		bookingID, signature)
	if err != nil {
		return Checklist{}, fmt.Errorf("confirm checklist: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return Checklist{}, ErrNotFound
	}
	return r.GetByBooking(ctx, bookingID)
}

// SetItemPhoto records an uploaded before/after photo URL on a checklist item.
func (r *Repository) SetItemPhoto(ctx context.Context, itemID int64, kind, url string) (Checklist, error) {
	var checklistID int64
	var bookingID int64
	err := r.pool.QueryRow(ctx,
		`SELECT bci.checklist_id, bc.booking_id FROM booking_checklist_items bci
		 JOIN booking_checklists bc ON bc.id = bci.checklist_id WHERE bci.id = $1`,
		itemID).Scan(&checklistID, &bookingID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Checklist{}, ErrItemNotFound
	}
	if err != nil {
		return Checklist{}, fmt.Errorf("get checklist item %d: %w", itemID, err)
	}
	column := "before_photo_url"
	if kind == "after" {
		column = "after_photo_url"
	}
	if _, err := r.pool.Exec(ctx,
		`UPDATE booking_checklist_items SET `+column+` = $2 WHERE id = $1`, itemID, url); err != nil {
		return Checklist{}, fmt.Errorf("attach photo to item %d: %w", itemID, err)
	}
	return r.GetByBooking(ctx, bookingID)
}
