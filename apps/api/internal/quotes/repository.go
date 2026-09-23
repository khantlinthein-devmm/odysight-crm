package quotes

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/odysight/crm/pkg/pagination"
)

var ErrNotFound = errors.New("quote not found")

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const quoteColumns = `id, quote_number, customer_id, site_id, status, valid_until, subtotal, tax_rate, total, currency, notes, version, accepted_at, rejected_at, converted_booking_id, converted_contract_id, created_at, updated_at`

func parseDate(v string) (time.Time, error) {
	return time.Parse("2006-01-02", strings.TrimSpace(v))
}

func scanQuote(row pgx.Row) (Quote, error) {
	var q Quote
	err := row.Scan(&q.ID, &q.QuoteNumber, &q.CustomerID, &q.SiteID, &q.Status,
		&q.ValidUntil, &q.Subtotal, &q.TaxRate, &q.Total, &q.Currency, &q.Notes,
		&q.Version, &q.AcceptedAt, &q.RejectedAt, &q.ConvertedBookingID,
		&q.ConvertedContractID, &q.CreatedAt, &q.UpdatedAt)
	return q, err
}

func (r *Repository) loadItems(ctx context.Context, q pgxQuerier, ids []int64) (map[int64][]QuoteItem, error) {
	out := make(map[int64][]QuoteItem, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := q.Query(ctx, `SELECT id, quote_id, service_name, description, quantity, unit_price, line_total, sort_order FROM quote_items WHERE quote_id = ANY($1) ORDER BY sort_order, id`, ids)
	if err != nil {
		return nil, fmt.Errorf("load quote items: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var it QuoteItem
		if err := rows.Scan(&it.ID, &it.QuoteID, &it.ServiceName, &it.Description, &it.Quantity, &it.UnitPrice, &it.LineTotal, &it.SortOrder); err != nil {
			return nil, fmt.Errorf("scan quote item: %w", err)
		}
		out[it.QuoteID] = append(out[it.QuoteID], it)
	}
	return out, rows.Err()
}

type pgxQuerier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func (r *Repository) List(ctx context.Context, params pagination.Params, customerID int64) ([]Quote, int, error) {
	args := []any{}
	conds := []string{}
	if customerID > 0 {
		args = append(args, customerID)
		conds = append(conds, "customer_id = $"+strconv.Itoa(len(args)))
	}
	if params.Search != "" {
		args = append(args, "%"+params.Search+"%")
		conds = append(conds, "(quote_number ILIKE $"+strconv.Itoa(len(args))+" OR notes ILIKE $"+strconv.Itoa(len(args))+")")
	}
	if params.Status != "" {
		args = append(args, params.Status)
		conds = append(conds, "status = $"+strconv.Itoa(len(args)))
	}
	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM quotes `+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count quotes: %w", err)
	}
	args = append(args, params.Limit, params.Offset)
	query := `SELECT ` + quoteColumns + ` FROM quotes ` + where +
		` ORDER BY created_at DESC LIMIT $` + strconv.Itoa(len(args)-1) + ` OFFSET $` + strconv.Itoa(len(args))
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query quotes: %w", err)
	}
	defer rows.Close()

	items := []Quote{}
	for rows.Next() {
		item, err := scanQuote(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan quote: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	ids := make([]int64, 0, len(items))
	for _, q := range items {
		ids = append(ids, q.ID)
	}
	itemMap, err := r.loadItems(ctx, r.pool, ids)
	if err != nil {
		return nil, 0, err
	}
	for i := range items {
		items[i].Items = itemMap[items[i].ID]
		if items[i].Items == nil {
			items[i].Items = []QuoteItem{}
		}
	}
	return items, total, nil
}

func (r *Repository) GetByID(ctx context.Context, id int64) (Quote, error) {
	q, err := scanQuote(r.pool.QueryRow(ctx, `SELECT `+quoteColumns+` FROM quotes WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Quote{}, ErrNotFound
	}
	if err != nil {
		return Quote{}, fmt.Errorf("get quote %d: %w", id, err)
	}
	itemMap, err := r.loadItems(ctx, r.pool, []int64{id})
	if err != nil {
		return Quote{}, err
	}
	q.Items = itemMap[id]
	if q.Items == nil {
		q.Items = []QuoteItem{}
	}
	return q, nil
}

func totals(items []QuoteItemInput, taxRate float64) (float64, float64) {
	sub := 0.0
	for _, it := range items {
		sub += it.Quantity * it.UnitPrice
	}
	sub = round2(sub)
	return sub, round2(sub + sub*taxRate/100)
}

func (r *Repository) Create(ctx context.Context, q Quote, items []QuoteItemInput) (Quote, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Quote{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if q.SiteID != nil {
		var owner int64
		if err := tx.QueryRow(ctx, `SELECT customer_id FROM sites WHERE id = $1`, *q.SiteID).Scan(&owner); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return Quote{}, fmt.Errorf("site not found")
			}
			return Quote{}, fmt.Errorf("check quote site: %w", err)
		}
		if owner != q.CustomerID {
			return Quote{}, fmt.Errorf("site does not belong to this customer")
		}
	}
	var id int64
	if err := tx.QueryRow(ctx,
		`INSERT INTO quotes (quote_number, customer_id, site_id, status, valid_until, subtotal, tax_rate, total, currency, notes, version)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 1) RETURNING id`,
		"TMP", q.CustomerID, q.SiteID, q.Status, q.ValidUntil, q.Subtotal, q.TaxRate, q.Total, q.Currency, q.Notes).Scan(&id); err != nil {
		return Quote{}, fmt.Errorf("create quote: %w", err)
	}
	for i, it := range items {
		line := round2(it.Quantity * it.UnitPrice)
		if _, err := tx.Exec(ctx,
			`INSERT INTO quote_items (quote_id, service_name, description, quantity, unit_price, line_total, sort_order)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			id, strings.TrimSpace(it.ServiceName), strings.TrimSpace(it.Description), it.Quantity, it.UnitPrice, line, i); err != nil {
			return Quote{}, fmt.Errorf("create quote item: %w", err)
		}
	}
	created, err := scanQuote(tx.QueryRow(ctx,
		`UPDATE quotes SET quote_number = 'QT-' || to_char(created_at, 'YYYY') || '-' || lpad(id::text, 4, '0') WHERE id = $1 RETURNING `+quoteColumns, id))
	if err != nil {
		return Quote{}, fmt.Errorf("assign quote number: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Quote{}, fmt.Errorf("commit quote: %w", err)
	}
	return r.GetByID(ctx, created.ID)
}

type Patch struct {
	SiteID              *int64
	ClearSiteID         bool
	Status              *Status
	ValidUntil          *time.Time
	ClearValidUntil     bool
	TaxRate             *float64
	Currency            *string
	Notes               *string
	Items               []QuoteItemInput
	HasItems            bool
	ConvertedBookingID  *int64
	ConvertedContractID *int64
}

func (r *Repository) Update(ctx context.Context, id int64, p Patch) (Quote, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Quote{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var customerID int64
	var version int
	var status string
	if err := tx.QueryRow(ctx, `SELECT customer_id, version, status FROM quotes WHERE id = $1`, id).Scan(&customerID, &version, &status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Quote{}, ErrNotFound
		}
		return Quote{}, fmt.Errorf("get quote %d: %w", id, err)
	}
	if p.SiteID != nil {
		var owner int64
		if err := tx.QueryRow(ctx, `SELECT customer_id FROM sites WHERE id = $1`, *p.SiteID).Scan(&owner); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return Quote{}, fmt.Errorf("site not found")
			}
			return Quote{}, fmt.Errorf("check quote site: %w", err)
		}
		if owner != customerID {
			return Quote{}, fmt.Errorf("site does not belong to this customer")
		}
	}

	var statusVal any
	if p.Status != nil {
		statusVal = *p.Status
	}
	updated, err := scanQuote(tx.QueryRow(ctx,
		`UPDATE quotes SET
			site_id     = CASE WHEN $2 THEN NULL WHEN $3::bigint IS NULL THEN site_id ELSE $3 END,
			status      = COALESCE($4, status),
			valid_until = CASE WHEN $5 THEN NULL ELSE COALESCE($6, valid_until) END,
			tax_rate    = COALESCE($7, tax_rate),
			currency    = COALESCE($8, currency),
			notes       = COALESCE($9, notes),
			converted_booking_id  = COALESCE($10, converted_booking_id),
			converted_contract_id = COALESCE($11, converted_contract_id),
			accepted_at = CASE WHEN $4 = 'accepted' THEN now() ELSE accepted_at END,
			rejected_at = CASE WHEN $4 = 'rejected' THEN now() ELSE rejected_at END,
			version     = version + 1
		 WHERE id = $1
		 RETURNING `+quoteColumns,
		id, p.ClearSiteID, p.SiteID, statusVal, p.ClearValidUntil, p.ValidUntil, p.TaxRate, p.Currency, p.Notes, p.ConvertedBookingID, p.ConvertedContractID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Quote{}, ErrNotFound
	}
	if err != nil {
		return Quote{}, fmt.Errorf("update quote %d: %w", id, err)
	}
	if p.HasItems {
		if _, err := tx.Exec(ctx, `DELETE FROM quote_items WHERE quote_id = $1`, id); err != nil {
			return Quote{}, fmt.Errorf("clear quote items: %w", err)
		}
		sub := 0.0
		for i, it := range p.Items {
			line := round2(it.Quantity * it.UnitPrice)
			sub += line
			if _, err := tx.Exec(ctx,
				`INSERT INTO quote_items (quote_id, service_name, description, quantity, unit_price, line_total, sort_order)
				 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
				id, strings.TrimSpace(it.ServiceName), strings.TrimSpace(it.Description), it.Quantity, it.UnitPrice, line, i); err != nil {
				return Quote{}, fmt.Errorf("replace quote item: %w", err)
			}
		}
		sub = round2(sub)
		total := round2(sub + sub*updated.TaxRate/100)
		if _, err := tx.Exec(ctx, `UPDATE quotes SET subtotal = $2, total = $3 WHERE id = $1`, id, sub, total); err != nil {
			return Quote{}, fmt.Errorf("recalc quote totals: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return Quote{}, fmt.Errorf("commit quote %d: %w", id, err)
	}
	return r.GetByID(ctx, id)
}
