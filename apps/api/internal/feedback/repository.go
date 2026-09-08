package feedback

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/odysight/crm/pkg/pagination"
)

var ErrNotFound = errors.New("feedback not found")

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const feedbackColumns = `f.id, f.booking_id, b.booking_number, f.customer_id,
	COALESCE(c.first_name || ' ' || c.last_name, b.customer_name),
	f.rating, f.comment, f.created_at`

const feedbackFrom = `FROM feedback f
	LEFT JOIN bookings b ON b.id = f.booking_id
	LEFT JOIN customers c ON c.id = f.customer_id`

func scanFeedback(row pgx.Row) (Feedback, error) {
	var fb Feedback
	err := row.Scan(&fb.ID, &fb.BookingID, &fb.BookingNumber, &fb.CustomerID,
		&fb.CustomerName, &fb.Rating, &fb.Comment, &fb.CreatedAt)
	return fb, err
}

func (r *Repository) List(ctx context.Context, params pagination.Params) ([]Feedback, int, error) {
	args := []any{}
	conds := []string{}
	if params.Search != "" {
		like := "%" + params.Search + "%"
		start := len(args) + 1
		args = append(args, like, like, like)
		conds = append(conds, "(b.booking_number ILIKE $"+itoa(start)+" OR b.customer_name ILIKE $"+itoa(start+1)+" OR COALESCE(c.first_name,'')||' '||COALESCE(c.last_name,'') ILIKE $"+itoa(start+2)+")")
	}
	where := ""
	if len(conds) > 0 {
		where = "WHERE " + joinAnd(conds)
	}
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM feedback f LEFT JOIN bookings b ON b.id = f.booking_id LEFT JOIN customers c ON c.id = f.customer_id `+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count feedback: %w", err)
	}
	args = append(args, params.Limit, params.Offset)
	query := `SELECT ` + feedbackColumns + ` ` + feedbackFrom + ` ` + where +
		` ORDER BY f.created_at DESC LIMIT $` + itoa(len(args)-1) + ` OFFSET $` + itoa(len(args))
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query feedback: %w", err)
	}
	defer rows.Close()

	items := []Feedback{}
	for rows.Next() {
		item, err := scanFeedback(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan feedback: %w", err)
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *Repository) GetByID(ctx context.Context, id int64) (Feedback, error) {
	fb, err := scanFeedback(r.pool.QueryRow(ctx,
		`SELECT `+feedbackColumns+` `+feedbackFrom+` WHERE f.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Feedback{}, ErrNotFound
	}
	if err != nil {
		return Feedback{}, fmt.Errorf("get feedback %d: %w", id, err)
	}
	return fb, nil
}

func (r *Repository) Create(ctx context.Context, fb Feedback) (Feedback, error) {
	var id int64
	err := r.pool.QueryRow(ctx,
		`INSERT INTO feedback (booking_id, customer_id, rating, comment)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id`,
		fb.BookingID, fb.CustomerID, fb.Rating, fb.Comment).Scan(&id)
	if err != nil {
		return Feedback{}, fmt.Errorf("create feedback: %w", err)
	}
	return r.GetByID(ctx, id)
}

func (r *Repository) Update(ctx context.Context, id int64, rating *int, comment *string) (Feedback, error) {
	var ratingArg any
	if rating != nil {
		ratingArg = *rating
	}
	var commentArg any
	if comment != nil {
		commentArg = *comment
	}
	tag, err := r.pool.Exec(ctx,
		`UPDATE feedback SET
			rating  = COALESCE($2, rating),
			comment = COALESCE($3, comment)
		 WHERE id = $1`,
		id, ratingArg, commentArg)
	if err != nil {
		return Feedback{}, fmt.Errorf("update feedback %d: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return Feedback{}, ErrNotFound
	}
	return r.GetByID(ctx, id)
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM feedback WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete feedback %d: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
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