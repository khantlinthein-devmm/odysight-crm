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
	ErrAlreadyExists    = errors.New("booking already has a checklist")
	ErrNoTemplate       = errors.New("no checklist template")
	ErrAlreadySigned    = errors.New("checklist already confirmed by the client")
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
	rows, err := q.Query(ctx, `SELECT id, template_id, label, item_type, require_photo, sort_order FROM checklist_template_items WHERE template_id = ANY($1) ORDER BY sort_order, id`, ids)
	if err != nil {
		return nil, fmt.Errorf("load template items: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var it TemplateItem
		var tid int64
		if err := rows.Scan(&it.ID, &tid, &it.Label, &it.ItemType, &it.RequirePhoto, &it.SortOrder); err != nil {
			return nil, fmt.Errorf("scan template item: %w", err)
		}
		out[tid] = append(out[tid], it)
	}
	return out, rows.Err()
}

func (r *Repository) getTemplate(ctx context.Context, id int64) (Template, error) {
	var t Template
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, service_type, is_active, created_at, updated_at FROM checklist_templates WHERE id = $1`, id).
		Scan(&t.ID, &t.Name, &t.ServiceType, &t.IsActive, &t.CreatedAt, &t.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Template{}, ErrTemplateNotFound
	}
	if err != nil {
		return Template{}, fmt.Errorf("get template %d: %w", id, err)
	}
	itemMap, err := r.templateItems(ctx, r.pool, []int64{t.ID})
	if err != nil {
		return Template{}, err
	}
	t.Items = itemMap[t.ID]
	if t.Items == nil {
		t.Items = []TemplateItem{}
	}
	return t, nil
}

func insertTemplateItems(ctx context.Context, tx pgx.Tx, templateID int64, items []TemplateItem) error {
	for i, it := range items {
		if _, err := tx.Exec(ctx,
			`INSERT INTO checklist_template_items (template_id, label, item_type, require_photo, sort_order) VALUES ($1, $2, $3, $4, $5)`,
			templateID, it.Label, it.ItemType, it.RequirePhoto, i); err != nil {
			return fmt.Errorf("create template item: %w", err)
		}
	}
	return nil
}

func (r *Repository) CreateTemplate(ctx context.Context, name, serviceType string, items []TemplateItem) (Template, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Template{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var id int64
	if err := tx.QueryRow(ctx,
		`INSERT INTO checklist_templates (name, service_type) VALUES ($1, $2) RETURNING id`,
		name, serviceType).Scan(&id); err != nil {
		return Template{}, fmt.Errorf("create template: %w", err)
	}
	if err := insertTemplateItems(ctx, tx, id, items); err != nil {
		return Template{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Template{}, fmt.Errorf("commit template: %w", err)
	}
	return r.getTemplate(ctx, id)
}

// UpdateTemplate replaces a template's name, service, active flag and items.
// Checklists already created from it keep their own copied items.
func (r *Repository) UpdateTemplate(ctx context.Context, id int64, name, serviceType string, isActive bool, items []TemplateItem) (Template, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Template{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx,
		`UPDATE checklist_templates SET name = $2, service_type = $3, is_active = $4, updated_at = now() WHERE id = $1`,
		id, name, serviceType, isActive)
	if err != nil {
		return Template{}, fmt.Errorf("update template %d: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return Template{}, ErrTemplateNotFound
	}
	if _, err := tx.Exec(ctx, `DELETE FROM checklist_template_items WHERE template_id = $1`, id); err != nil {
		return Template{}, fmt.Errorf("clear template items: %w", err)
	}
	if err := insertTemplateItems(ctx, tx, id, items); err != nil {
		return Template{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Template{}, fmt.Errorf("commit template: %w", err)
	}
	return r.getTemplate(ctx, id)
}

// Checklists

const checklistColumns = `c.id, c.booking_id, c.template_id, c.status, c.client_signature, c.client_confirmed_at,
	c.submitted_at, su.name, c.review_status, c.review_note, ru.name, c.reviewed_at, c.created_at, c.updated_at`

const checklistFrom = `FROM booking_checklists c
	LEFT JOIN users su ON su.id = c.submitted_by_user_id
	LEFT JOIN users ru ON ru.id = c.reviewed_by_user_id`

func (r *Repository) GetByBooking(ctx context.Context, bookingID int64) (Checklist, error) {
	var c Checklist
	err := r.pool.QueryRow(ctx, `SELECT `+checklistColumns+` `+checklistFrom+` WHERE c.booking_id = $1`, bookingID).
		Scan(&c.ID, &c.BookingID, &c.TemplateID, &c.Status, &c.ClientSignature, &c.ClientConfirmedAt,
			&c.SubmittedAt, &c.SubmittedBy, &c.ReviewStatus, &c.ReviewNote, &c.ReviewedBy, &c.ReviewedAt, &c.CreatedAt, &c.UpdatedAt)
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
		`SELECT i.id, i.checklist_id, i.label, i.item_type, i.require_photo, i.is_completed, i.score, i.result,
		        COALESCE(u.name, i.completed_by), i.completed_by_user_id, i.completed_at, i.notes,
		        i.before_photo_url, i.after_photo_url, i.sort_order
		 FROM booking_checklist_items i
		 LEFT JOIN users u ON u.id = i.completed_by_user_id
		 WHERE i.checklist_id = $1 ORDER BY i.sort_order, i.id`, checklistID)
	if err != nil {
		return nil, fmt.Errorf("load checklist items: %w", err)
	}
	defer rows.Close()
	items := []ChecklistItem{}
	for rows.Next() {
		var it ChecklistItem
		var score *int16
		if err := rows.Scan(&it.ID, &it.ChecklistID, &it.Label, &it.ItemType, &it.RequirePhoto, &it.IsCompleted, &score, &it.Result,
			&it.CompletedBy, &it.CompletedByUser, &it.CompletedAt, &it.Notes, &it.BeforePhotoURL, &it.AfterPhotoURL, &it.SortOrder); err != nil {
			return nil, fmt.Errorf("scan checklist item: %w", err)
		}
		if score != nil {
			v := int(*score)
			it.Score = &v
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

// Create starts a booking checklist. With a template and no ad-hoc items the
// template's items (label, type, photo rule) are copied in.
func (r *Repository) Create(ctx context.Context, bookingID int64, templateID *int64, items []TemplateItem) (Checklist, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Checklist{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if templateID != nil && len(items) == 0 {
		m, err := r.templateItems(ctx, tx, []int64{*templateID})
		if err != nil {
			return Checklist{}, err
		}
		items = m[*templateID]
		if len(items) == 0 {
			return Checklist{}, ErrTemplateNotFound
		}
	}
	var checklistID int64
	err = tx.QueryRow(ctx,
		`INSERT INTO booking_checklists (booking_id, template_id, status) VALUES ($1, $2, 'pending')
		 ON CONFLICT (booking_id) DO NOTHING RETURNING id`,
		bookingID, templateID).Scan(&checklistID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Checklist{}, ErrAlreadyExists
	}
	if err != nil {
		return Checklist{}, fmt.Errorf("create checklist: %w", err)
	}
	for i, it := range items {
		if _, err := tx.Exec(ctx,
			`INSERT INTO booking_checklist_items (checklist_id, label, item_type, require_photo, sort_order) VALUES ($1, $2, $3, $4, $5)`,
			checklistID, strings.TrimSpace(it.Label), it.ItemType, it.RequirePhoto, i); err != nil {
			return Checklist{}, fmt.Errorf("create checklist item: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return Checklist{}, fmt.Errorf("commit checklist: %w", err)
	}
	return r.GetByBooking(ctx, bookingID)
}

// ItemContext is what an item update needs to know about its checklist.
type ItemContext struct {
	Item      ChecklistItem
	BookingID int64
	Locked    bool
}

func (r *Repository) ItemContext(ctx context.Context, itemID int64) (ItemContext, error) {
	var ic ItemContext
	var score *int16
	err := r.pool.QueryRow(ctx,
		`SELECT i.id, i.checklist_id, i.item_type, i.is_completed, i.score, i.result, i.completed_at,
		        c.booking_id, c.submitted_at IS NOT NULL
		 FROM booking_checklist_items i JOIN booking_checklists c ON c.id = i.checklist_id
		 WHERE i.id = $1`, itemID).
		Scan(&ic.Item.ID, &ic.Item.ChecklistID, &ic.Item.ItemType, &ic.Item.IsCompleted, &score, &ic.Item.Result,
			&ic.Item.CompletedAt, &ic.BookingID, &ic.Locked)
	if errors.Is(err, pgx.ErrNoRows) {
		return ItemContext{}, ErrItemNotFound
	}
	if err != nil {
		return ItemContext{}, fmt.Errorf("get checklist item %d: %w", itemID, err)
	}
	if score != nil {
		v := int(*score)
		ic.Item.Score = &v
	}
	return ic, nil
}

// ItemUpdate is the resolved new state of one item.
type ItemUpdate struct {
	Completed   bool
	Score       *int
	Result      *string
	CompletedAt *time.Time
	ByUserID    *int64
	ByName      *string // legacy free-text name; nil keeps the stored one
	Notes       *string // nil keeps the stored notes
}

func (r *Repository) UpdateItem(ctx context.Context, ic ItemContext, u ItemUpdate) (Checklist, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Checklist{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx,
		`UPDATE booking_checklist_items SET is_completed = $2, score = $3, result = $4, completed_at = $5,
		 completed_by_user_id = $6, completed_by = COALESCE($7, completed_by), notes = COALESCE($8, notes),
		 updated_at = now()
		 WHERE id = $1`,
		ic.Item.ID, u.Completed, u.Score, u.Result, u.CompletedAt, u.ByUserID, u.ByName, u.Notes); err != nil {
		return Checklist{}, fmt.Errorf("update item %d: %w", ic.Item.ID, err)
	}
	var remaining, done int
	if err := tx.QueryRow(ctx,
		`SELECT COUNT(*) FILTER (WHERE NOT is_completed), COUNT(*) FILTER (WHERE is_completed)
		 FROM booking_checklist_items WHERE checklist_id = $1`, ic.Item.ChecklistID).Scan(&remaining, &done); err != nil {
		return Checklist{}, fmt.Errorf("count remaining items: %w", err)
	}
	status := StatusInProgress
	switch {
	case remaining == 0:
		status = StatusCompleted
	case done == 0:
		status = StatusPending
	}
	if _, err := tx.Exec(ctx, `UPDATE booking_checklists SET status = $2, updated_at = now() WHERE id = $1`, ic.Item.ChecklistID, status); err != nil {
		return Checklist{}, fmt.Errorf("update checklist status: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Checklist{}, fmt.Errorf("commit item %d: %w", ic.Item.ID, err)
	}
	return r.GetByBooking(ctx, ic.BookingID)
}

func (r *Repository) Confirm(ctx context.Context, bookingID int64, signature string) (Checklist, error) {
	tag, err := r.pool.Exec(ctx,
		`UPDATE booking_checklists SET client_signature = $2, client_confirmed_at = now(), updated_at = now()
		 WHERE booking_id = $1 AND client_signature IS NULL`,
		bookingID, signature)
	if err != nil {
		return Checklist{}, fmt.Errorf("confirm checklist: %w", err)
	}
	if tag.RowsAffected() == 0 {
		if _, err := r.GetByBooking(ctx, bookingID); err != nil {
			return Checklist{}, err
		}
		return Checklist{}, ErrAlreadySigned
	}
	return r.GetByBooking(ctx, bookingID)
}

// Submit locks the checklist and records who handed it in. A previous
// "rework" review is cleared so the office sees it as waiting again.
func (r *Repository) Submit(ctx context.Context, bookingID, userID int64) (Checklist, error) {
	tag, err := r.pool.Exec(ctx,
		`UPDATE booking_checklists SET submitted_at = now(), submitted_by_user_id = $2,
		 review_status = '', reviewed_by_user_id = NULL, reviewed_at = NULL, updated_at = now()
		 WHERE booking_id = $1 AND submitted_at IS NULL`, bookingID, userID)
	if err != nil {
		return Checklist{}, fmt.Errorf("submit checklist: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return Checklist{}, ErrNotFound
	}
	return r.GetByBooking(ctx, bookingID)
}

// Review records the office decision. Rework unlocks the checklist again.
func (r *Repository) Review(ctx context.Context, bookingID, userID int64, decision, note string) (Checklist, error) {
	q := `UPDATE booking_checklists SET review_status = $3, review_note = $4, reviewed_by_user_id = $2,
	      reviewed_at = now(), updated_at = now()`
	if decision == ReviewRework {
		q += `, submitted_at = NULL`
	}
	tag, err := r.pool.Exec(ctx, q+` WHERE booking_id = $1 AND submitted_at IS NOT NULL`, bookingID, userID, decision, note)
	if err != nil {
		return Checklist{}, fmt.Errorf("review checklist: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return Checklist{}, ErrNotFound
	}
	return r.GetByBooking(ctx, bookingID)
}

// SetItemPhoto records an uploaded before/after photo URL on a checklist item.
func (r *Repository) SetItemPhoto(ctx context.Context, ic ItemContext, kind, url string) (Checklist, error) {
	column := "before_photo_url"
	if kind == "after" {
		column = "after_photo_url"
	}
	if _, err := r.pool.Exec(ctx,
		`UPDATE booking_checklist_items SET `+column+` = $2, updated_at = now() WHERE id = $1`, ic.Item.ID, url); err != nil {
		return Checklist{}, fmt.Errorf("attach photo to item %d: %w", ic.Item.ID, err)
	}
	return r.GetByBooking(ctx, ic.BookingID)
}

// TemplateForBooking picks the active template matching the booking's
// service type, falling back to the general template (blank service type).
func (r *Repository) TemplateForBooking(ctx context.Context, bookingID int64) (int64, error) {
	var id int64
	err := r.pool.QueryRow(ctx,
		`SELECT t.id FROM checklist_templates t
		 LEFT JOIN bookings b ON b.id = $1
		 WHERE t.is_active AND (t.service_type = b.service_type OR t.service_type = '')
		   AND EXISTS (SELECT 1 FROM checklist_template_items i WHERE i.template_id = t.id)
		 ORDER BY (t.service_type = b.service_type) DESC, t.id
		 LIMIT 1`, bookingID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNoTemplate
	}
	if err != nil {
		return 0, fmt.Errorf("pick template for booking %d: %w", bookingID, err)
	}
	return id, nil
}

// Cleaner scoping

// cleanerOfUser matches a login to its cleaner profile: by user_id link, or
// by email for profiles not yet linked (same rule as the bookings module).
const cleanerOfUser = `SELECT cl.id, trim(cl.first_name || ' ' || cl.last_name) AS name
	FROM cleaners cl LEFT JOIN users u ON u.id = $1
	WHERE cl.user_id = $1 OR (cl.user_id IS NULL AND lower(cl.email) = lower(u.email))`

// assignedFilter is true when booking b is on the cleaner's crew, or (legacy
// rows without a crew) names the cleaner in assigned_cleaner.
const assignedFilter = `(EXISTS (SELECT 1 FROM booking_cleaners bc WHERE bc.booking_id = b.id AND bc.cleaner_id = me.id)
	OR (NOT EXISTS (SELECT 1 FROM booking_cleaners bc WHERE bc.booking_id = b.id)
	    AND lower(trim(COALESCE(b.assigned_cleaner, ''))) = lower(me.name) AND me.name <> ''))`

// CleanerAssigned reports whether the login's cleaner profile works booking.
func (r *Repository) CleanerAssigned(ctx context.Context, userID, bookingID int64) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM bookings b, (`+cleanerOfUser+`) me WHERE b.id = $2 AND `+assignedFilter+`)`,
		userID, bookingID).Scan(&ok)
	if err != nil {
		return false, fmt.Errorf("check cleaner assignment: %w", err)
	}
	return ok, nil
}

// StatusRow is a compact checklist state for job cards.
type StatusRow struct {
	BookingID    int64
	Status       Status
	Submitted    bool
	ReviewStatus string
	ReviewNote   string
	ItemsTotal   int
	ItemsDone    int
}

// Statuses returns checklist state for the given bookings. With cleanerUser
// set, only bookings that cleaner works are returned.
func (r *Repository) Statuses(ctx context.Context, bookingIDs []int64, cleanerUser *int64) ([]StatusRow, error) {
	args := []any{bookingIDs}
	scope := ""
	if cleanerUser != nil {
		args = append(args, *cleanerUser)
		scope = ` AND EXISTS (SELECT 1 FROM (` + strings.ReplaceAll(cleanerOfUser, "$1", "$2") + `) me WHERE ` + assignedFilter + `)`
	}
	rows, err := r.pool.Query(ctx,
		`SELECT c.booking_id, c.status, c.submitted_at IS NOT NULL, c.review_status, c.review_note,
		        (SELECT COUNT(*) FROM booking_checklist_items i WHERE i.checklist_id = c.id),
		        (SELECT COUNT(*) FROM booking_checklist_items i WHERE i.checklist_id = c.id AND i.is_completed)
		 FROM booking_checklists c JOIN bookings b ON b.id = c.booking_id
		 WHERE c.booking_id = ANY($1)`+scope, args...)
	if err != nil {
		return nil, fmt.Errorf("checklist statuses: %w", err)
	}
	defer rows.Close()
	out := []StatusRow{}
	for rows.Next() {
		var s StatusRow
		if err := rows.Scan(&s.BookingID, &s.Status, &s.Submitted, &s.ReviewStatus, &s.ReviewNote, &s.ItemsTotal, &s.ItemsDone); err != nil {
			return nil, fmt.Errorf("scan checklist status: %w", err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// Office list

func (r *Repository) List(ctx context.Context, f ListFilter) ([]Summary, int, error) {
	conds := []string{}
	args := []any{}
	add := func(cond string, v any) {
		args = append(args, v)
		conds = append(conds, strings.ReplaceAll(cond, "?", fmt.Sprintf("$%d", len(args))))
	}
	switch f.State {
	case "pending":
		conds = append(conds, `c.submitted_at IS NULL AND c.review_status = '' AND c.status = 'pending'`)
	case "in_progress":
		conds = append(conds, `c.submitted_at IS NULL AND c.review_status = '' AND c.status <> 'pending'`)
	case "submitted":
		conds = append(conds, `c.submitted_at IS NOT NULL AND c.review_status = ''`)
	case ReviewApproved:
		conds = append(conds, `c.review_status = 'approved'`)
	case ReviewRework:
		conds = append(conds, `c.review_status = 'rework'`)
	}
	if f.CleanerID > 0 {
		add(`(EXISTS (SELECT 1 FROM booking_cleaners bc WHERE bc.booking_id = b.id AND bc.cleaner_id = ?)
		  OR (NOT EXISTS (SELECT 1 FROM booking_cleaners bc WHERE bc.booking_id = b.id)
		      AND lower(trim(COALESCE(b.assigned_cleaner, ''))) =
		          (SELECT lower(trim(cl.first_name || ' ' || cl.last_name)) FROM cleaners cl WHERE cl.id = ?)))`, f.CleanerID)
	}
	if f.From != nil {
		add(`b.scheduled_for >= ?`, *f.From)
	}
	if f.To != nil {
		add(`b.scheduled_for < ?`, f.To.AddDate(0, 0, 1))
	}
	if f.Search != "" {
		add(`(b.booking_number ILIKE ? OR b.customer_name ILIKE ?)`, "%"+f.Search+"%")
	}
	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}
	from := `FROM booking_checklists c JOIN bookings b ON b.id = c.booking_id `

	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) `+from+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count checklists: %w", err)
	}
	args = append(args, f.Limit, f.Offset)
	rows, err := r.pool.Query(ctx,
		`SELECT c.id, b.id, b.booking_number, b.customer_name, b.service_type, b.scheduled_for,
		        COALESCE((SELECT string_agg(trim(cl.first_name || ' ' || cl.last_name), ', ' ORDER BY bc.role DESC, cl.first_name)
		                  FROM booking_cleaners bc JOIN cleaners cl ON cl.id = bc.cleaner_id WHERE bc.booking_id = b.id),
		                 NULLIF(trim(b.assigned_cleaner), ''), ''),
		        c.status, c.submitted_at, c.review_status,
		        s.total, s.done, s.avg_score, s.fails, s.photos
		 `+from+`
		 CROSS JOIN LATERAL (
		   SELECT COUNT(*)::int AS total,
		          (COUNT(*) FILTER (WHERE i.is_completed))::int AS done,
		          AVG(i.score)::float8 AS avg_score,
		          (COUNT(*) FILTER (WHERE i.result = 'fail'))::int AS fails,
		          (COUNT(i.before_photo_url) + COUNT(i.after_photo_url))::int AS photos
		   FROM booking_checklist_items i WHERE i.checklist_id = c.id) s
		 `+where+`
		 ORDER BY b.scheduled_for DESC, c.id DESC`+
			fmt.Sprintf(` LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list checklists: %w", err)
	}
	defer rows.Close()
	out := []Summary{}
	for rows.Next() {
		var s Summary
		if err := rows.Scan(&s.ID, &s.BookingID, &s.BookingNumber, &s.CustomerName, &s.ServiceType, &s.ScheduledFor,
			&s.Cleaners, &s.Status, &s.SubmittedAt, &s.ReviewStatus,
			&s.ItemsTotal, &s.ItemsDone, &s.AvgScore, &s.FailCount, &s.PhotoCount); err != nil {
			return nil, 0, fmt.Errorf("scan checklist summary: %w", err)
		}
		out = append(out, s)
	}
	return out, total, rows.Err()
}
