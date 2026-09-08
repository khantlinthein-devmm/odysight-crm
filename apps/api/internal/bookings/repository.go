package bookings

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/odysight/crm/pkg/pagination"
)

var ErrNotFound = errors.New("booking not found")

// ErrUnknownCleaner indicates a requested cleaner ID does not exist.
var ErrUnknownCleaner = errors.New("cleaner not found")

// Conflict describes a time-overlapping booking that blocks an assignment.
type Conflict struct {
	CleanerID       int64
	BookingID       int64
	BookingNumber   string
	CustomerName    string
	ScheduledFor    time.Time
	DurationMinutes int
	CleanerName     string
}

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const bookingColumns = `id, booking_number, customer_name, customer_email, customer_id, service_type, scheduled_for, duration_minutes, address, assigned_cleaner, status, notes, is_recurring, recurrence, series_id, created_at`

const bookingNumberExpr = `'BK-' || to_char(created_at, 'YYYY') || '-' || lpad(id::text, 4, '0')`

func scanBooking(row pgx.Row) (Booking, error) {
	var b Booking
	var recurrence *string
	err := row.Scan(&b.ID, &b.BookingNumber, &b.CustomerName, &b.CustomerEmail, &b.CustomerID, &b.ServiceType,
		&b.ScheduledFor, &b.DurationMinutes, &b.Address, &b.AssignedCleaner,
		&b.Status, &b.Notes, &b.IsRecurring, &recurrence, &b.SeriesID, &b.CreatedAt)
	if recurrence != nil {
		b.Recurrence = *recurrence
	}
	return b, err
}

const assignmentQuery = `
	SELECT c.id, c.first_name || ' ' || c.last_name, bc.role
	FROM booking_cleaners bc
	JOIN cleaners c ON c.id = bc.cleaner_id
	WHERE bc.booking_id = $1
	ORDER BY (bc.role = 'primary') DESC, bc.id`

// loadAssignments fills b.Cleaners (and the primary display name) in place.
func (r *Repository) loadAssignments(ctx context.Context, b *Booking) error {
	rows, err := r.pool.Query(ctx, assignmentQuery, b.ID)
	if err != nil {
		return fmt.Errorf("load assignments for booking %d: %w", b.ID, err)
	}
	defer rows.Close()
	b.Cleaners = []CleanerBrief{}
	for rows.Next() {
		var c CleanerBrief
		if err := rows.Scan(&c.ID, &c.Name, &c.Role); err != nil {
			return fmt.Errorf("scan assignment for booking %d: %w", b.ID, err)
		}
		b.Cleaners = append(b.Cleaners, c)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate assignments for booking %d: %w", b.ID, err)
	}
	return nil
}

// replaceAssignments deletes and recreates the booking's cleaner links.
// Primary is the first ID, the remainder are crew.
func (r *Repository) replaceAssignments(ctx context.Context, q pgxQuerier, bookingID int64, ids []int64) error {
	if _, err := q.Exec(ctx, `DELETE FROM booking_cleaners WHERE booking_id = $1`, bookingID); err != nil {
		return fmt.Errorf("clear assignments for booking %d: %w", bookingID, err)
	}
	if len(ids) == 0 {
		return nil
	}
	for i, id := range ids {
		role := "crew"
		if i == 0 {
			role = "primary"
		}
		if _, err := q.Exec(ctx,
			`INSERT INTO booking_cleaners (booking_id, cleaner_id, role) VALUES ($1, $2, $3)`,
			bookingID, id, role); err != nil {
			return fmt.Errorf("assign cleaner %d to booking %d: %w", id, bookingID, err)
		}
	}
	return nil
}

// pgxQuerier abstracts *pgxpool.Pool and pgx.Tx so assignments can run inside
// the create/update transactions.
type pgxQuerier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// CleanerNames resolves cleaner IDs to full names, returning ErrUnknownCleaner
// if any ID does not exist.
func (r *Repository) CleanerNames(ctx context.Context, ids []int64) ([]CleanerBrief, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, first_name || ' ' || last_name FROM cleaners WHERE id = ANY($1) ORDER BY id`, ids)
	if err != nil {
		return nil, fmt.Errorf("query cleaner names: %w", err)
	}
	defer rows.Close()
	found := map[int64]string{}
	out := []CleanerBrief{}
	for rows.Next() {
		var id int64
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, fmt.Errorf("scan cleaner name: %w", err)
		}
		found[id] = name
		out = append(out, CleanerBrief{ID: id, Name: name})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate cleaner names: %w", err)
	}
	for _, id := range ids {
		if _, ok := found[id]; !ok {
			return nil, ErrUnknownCleaner
		}
	}
	return out, nil
}

// FindConflicts returns bookings that overlap [start, end) for any of the
// cleaner IDs, excluding excludeID and non-active statuses.
func (r *Repository) FindConflicts(ctx context.Context, cleanerIDs []int64, start, end time.Time, excludeID int64) ([]Conflict, error) {
	if len(cleanerIDs) == 0 {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT bc.cleaner_id, c.first_name || ' ' || c.last_name,
		       b.id, b.booking_number, b.customer_name, b.scheduled_for, b.duration_minutes
		FROM booking_cleaners bc
		JOIN bookings b ON b.id = bc.booking_id
		JOIN cleaners c ON c.id = bc.cleaner_id
		WHERE bc.cleaner_id = ANY($1)
		  AND b.status NOT IN ('cancelled', 'no_show')
		  AND b.id <> $4
		  AND b.scheduled_for < $3
		  AND (b.scheduled_for + (b.duration_minutes * INTERVAL '1 minute')) > $2
		ORDER BY b.scheduled_for`, cleanerIDs, start, end, excludeID)
	if err != nil {
		return nil, fmt.Errorf("query conflicts: %w", err)
	}
	defer rows.Close()
	conflicts := []Conflict{}
	for rows.Next() {
		var c Conflict
		if err := rows.Scan(&c.CleanerID, &c.CleanerName, &c.BookingID, &c.BookingNumber, &c.CustomerName, &c.ScheduledFor, &c.DurationMinutes); err != nil {
			return nil, fmt.Errorf("scan conflict: %w", err)
		}
		conflicts = append(conflicts, c)
	}
	return conflicts, rows.Err()
}

func (r *Repository) List(ctx context.Context, params pagination.Params) ([]Booking, int, error) {
	args := []any{}
	conds := []string{}
	if params.Search != "" {
		like := "%" + params.Search + "%"
		start := len(args) + 1
		orParts := []string{}
		for i, col := range []string{"booking_number", "customer_name", "address", "assigned_cleaner"} {
			orParts = append(orParts, col+" ILIKE $"+itoa(start+i))
			args = append(args, like)
			_ = i
		}
		conds = append(conds, "("+joinOr(orParts)+")")
	}
	if params.Status != "" {
		args = append(args, params.Status)
		conds = append(conds, "status = $"+itoa(len(args)))
	}
	if params.From != "" {
		args = append(args, params.From)
		conds = append(conds, "scheduled_for >= $"+itoa(len(args)))
	}
	if params.To != "" {
		args = append(args, params.To)
		conds = append(conds, "scheduled_for < $"+itoa(len(args)))
	}
	if params.CleanerID > 0 {
		args = append(args, params.CleanerID)
		conds = append(conds,
			"(EXISTS (SELECT 1 FROM booking_cleaners bc WHERE bc.booking_id = bookings.id AND bc.cleaner_id = $"+itoa(len(args))+") "+
				"OR assigned_cleaner = (SELECT first_name || ' ' || last_name FROM cleaners WHERE id = $"+itoa(len(args))+"))")
	}
	where := ""
	if len(conds) > 0 {
		where = "WHERE " + joinAnd(conds)
	}
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM bookings `+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count bookings: %w", err)
	}
	args = append(args, params.Limit, params.Offset)
	query := `SELECT ` + bookingColumns + ` FROM bookings ` + where + ` ORDER BY scheduled_for DESC LIMIT $` + itoa(len(args)-1) + ` OFFSET $` + itoa(len(args))
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query bookings: %w", err)
	}
	defer rows.Close()

	items := []Booking{}
	for rows.Next() {
		item, err := scanBooking(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan booking: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate bookings: %w", err)
	}
	for i := range items {
		if err := r.loadAssignments(ctx, &items[i]); err != nil {
			return nil, 0, err
		}
	}
	return items, total, nil
}

func itoa(i int) string             { return fmt.Sprintf("%d", i) }
func joinOr(parts []string) string  { return joinWith(parts, " OR ") }
func joinAnd(parts []string) string { return joinWith(parts, " AND ") }
func joinWith(parts []string, sep string) string {
	out := ""
	for i, s := range parts {
		if i > 0 {
			out += sep
		}
		out += s
	}
	return out
}

func (r *Repository) GetByID(ctx context.Context, id int64) (Booking, error) {
	b, err := scanBooking(r.pool.QueryRow(ctx,
		`SELECT `+bookingColumns+` FROM bookings WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Booking{}, ErrNotFound
	}
	if err != nil {
		return Booking{}, fmt.Errorf("get booking %d: %w", id, err)
	}
	if err := r.loadAssignments(ctx, &b); err != nil {
		return Booking{}, err
	}
	return b, nil
}

// CustomerPhone returns the phone number for a linked customer, or "" when the
// booking has no customer link or the customer has no phone stored.
func (r *Repository) CustomerPhone(ctx context.Context, id int64) (string, error) {
	var phone string
	err := r.pool.QueryRow(ctx, `SELECT COALESCE(phone, '') FROM customers WHERE id = $1`, id).Scan(&phone)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("get customer %d phone: %w", id, err)
	}
	return phone, nil
}

func (r *Repository) Create(ctx context.Context, b Booking) (Booking, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Booking{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	// Unique temp value avoids UNIQUE collisions on concurrent inserts.
	temp := "TMP-" + fmt.Sprintf("%d", time.Now().UnixNano())
	var id int64
	if err := tx.QueryRow(ctx,
		`INSERT INTO bookings (booking_number, customer_name, customer_email, customer_id, service_type, scheduled_for, duration_minutes, address, assigned_cleaner, status, notes, is_recurring, recurrence, series_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		 RETURNING id`,
		temp, b.CustomerName, b.CustomerEmail, b.CustomerID, b.ServiceType, b.ScheduledFor, b.DurationMinutes,
		b.Address, b.AssignedCleaner, b.Status, b.Notes, b.IsRecurring, recurrenceValue(b), b.SeriesID).Scan(&id); err != nil {
		return Booking{}, fmt.Errorf("create booking: %w", err)
	}

	if len(b.Cleaners) > 0 {
		ids := make([]int64, 0, len(b.Cleaners))
		for _, c := range b.Cleaners {
			ids = append(ids, c.ID)
		}
		if err := r.replaceAssignments(ctx, tx, id, ids); err != nil {
			return Booking{}, err
		}
		primary := primaryName(b.Cleaners)
		if _, err := tx.Exec(ctx, `UPDATE bookings SET assigned_cleaner = $1 WHERE id = $2`, primary, id); err != nil {
			return Booking{}, fmt.Errorf("set primary cleaner on booking %d: %w", id, err)
		}
		b.AssignedCleaner = primary
	}

	created, err := scanBooking(tx.QueryRow(ctx,
		`UPDATE bookings SET booking_number = `+bookingNumberExpr+`
		 WHERE id = $1
		 RETURNING `+bookingColumns,
		id))
	if err != nil {
		return Booking{}, fmt.Errorf("assign booking number to booking %d: %w", id, err)
	}
	b.Cleaners = normalizeOrder(b.Cleaners)
	created.Cleaners = b.Cleaners
	if err := tx.Commit(ctx); err != nil {
		return Booking{}, fmt.Errorf("commit booking %d: %w", id, err)
	}
	return created, nil
}

// primaryName returns the first cleaner's full name, or "" if none is primary.
func primaryName(cleaners []CleanerBrief) string {
	if len(cleaners) == 0 {
		return ""
	}
	return cleaners[0].Name
}

// normalizeOrder guarantees primary first, then crew in the given order.
func normalizeOrder(cleaners []CleanerBrief) []CleanerBrief {
	out := make([]CleanerBrief, 0, len(cleaners))
	var crew []CleanerBrief
	for i, c := range cleaners {
		if i == 0 || c.Role == "primary" {
			c.Role = "primary"
			out = append(out, c)
		} else {
			c.Role = "crew"
			crew = append(crew, c)
		}
	}
	return append(out, crew...)
}

type Patch struct {
	CustomerName    *string
	CustomerEmail   *string
	CustomerID      *int64
	ServiceType     *ServiceType
	ScheduledFor    *time.Time
	DurationMinutes *int
	Address         *string
	AssignedCleaner *string
	Status          *Status
	Notes           *string
	IsRecurring     *bool
	Recurrence      *string
	// Cleaners replaces the whole assignment when non-nil (empty = clear).
	Cleaners []CleanerBrief
}

func (r *Repository) Update(ctx context.Context, id int64, p Patch) (Booking, error) {
	var serviceType any
	if p.ServiceType != nil {
		serviceType = *p.ServiceType
	}
	var status any
	if p.Status != nil {
		status = *p.Status
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Booking{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	updated, err := scanBooking(tx.QueryRow(ctx,
		`UPDATE bookings SET
			customer_name    = COALESCE($2, customer_name),
			customer_email   = COALESCE($3, customer_email),
			customer_id      = COALESCE($4, customer_id),
			service_type     = COALESCE($5, service_type),
			scheduled_for    = COALESCE($6, scheduled_for),
			duration_minutes = COALESCE($7, duration_minutes),
			address          = COALESCE($8, address),
			assigned_cleaner = COALESCE($9, assigned_cleaner),
			status           = COALESCE($10, status),
			notes            = COALESCE($11, notes),
			is_recurring     = COALESCE($12, is_recurring),
			recurrence       = CASE WHEN $13::text = '' THEN NULL ELSE COALESCE($13, recurrence) END
		 WHERE id = $1
		 RETURNING `+bookingColumns,
		id, p.CustomerName, p.CustomerEmail, p.CustomerID, serviceType, p.ScheduledFor, p.DurationMinutes,
		p.Address, p.AssignedCleaner, status, p.Notes, p.IsRecurring, p.Recurrence))
	if errors.Is(err, pgx.ErrNoRows) {
		return Booking{}, ErrNotFound
	}
	if err != nil {
		return Booking{}, fmt.Errorf("update booking %d: %w", id, err)
	}

	if p.Cleaners != nil {
		ids := make([]int64, 0, len(p.Cleaners))
		for _, c := range p.Cleaners {
			ids = append(ids, c.ID)
		}
		if err := r.replaceAssignments(ctx, tx, id, ids); err != nil {
			return Booking{}, err
		}
		primary := primaryName(normalizeOrder(p.Cleaners))
		if _, err := tx.Exec(ctx, `UPDATE bookings SET assigned_cleaner = $1 WHERE id = $2`, primary, id); err != nil {
			return Booking{}, fmt.Errorf("set primary cleaner on booking %d: %w", id, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return Booking{}, fmt.Errorf("commit update booking %d: %w", id, err)
	}
	if err := r.loadAssignments(ctx, &updated); err != nil {
		return Booking{}, err
	}
	return updated, nil
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM bookings WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete booking %d: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// DueRecurring returns active recurring bookings whose scheduled time has
// passed the cutoff. Each is a candidate for generating the next occurrence.
func (r *Repository) DueRecurring(ctx context.Context, cutoff time.Time) ([]Booking, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+bookingColumns+` FROM bookings
		 WHERE is_recurring = true AND scheduled_for < $1`, cutoff)
	if err != nil {
		return nil, fmt.Errorf("query due recurring bookings: %w", err)
	}
	defer rows.Close()

	items := []Booking{}
	for rows.Next() {
		item, err := scanBooking(rows)
		if err != nil {
			return nil, fmt.Errorf("scan recurring booking: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recurring bookings: %w", err)
	}
	for i := range items {
		if err := r.loadAssignments(ctx, &items[i]); err != nil {
			return nil, err
		}
	}
	return items, nil
}

// recurrenceValue returns the DB value for a booking's recurrence column:
// NULL for non-recurring bookings, otherwise the frequency string.
func recurrenceValue(b Booking) any {
	if !b.IsRecurring {
		return nil
	}
	return b.Recurrence
}

// DisableRecurring clears the recurring flags on a booking so the generator
// never processes it again (used after a successor has been created).
func (r *Repository) DisableRecurring(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE bookings SET is_recurring = false, recurrence = NULL WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("disable recurring on booking %d: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
