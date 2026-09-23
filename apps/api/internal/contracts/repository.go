package contracts

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

var ErrNotFound = errors.New("contract not found")

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const contractColumns = `id, contract_number, customer_id, title, status, start_date, end_date, renewal_date, contract_value, billing_frequency, sla_terms, notes, created_at, updated_at`

func parseDate(v string) (time.Time, error) {
	return time.Parse("2006-01-02", strings.TrimSpace(v))
}

func scanContract(row pgx.Row) (Contract, error) {
	var c Contract
	err := row.Scan(&c.ID, &c.ContractNumber, &c.CustomerID, &c.Title, &c.Status,
		&c.StartDate, &c.EndDate, &c.RenewalDate, &c.ContractValue, &c.BillingFrequency,
		&c.SLATerms, &c.Notes, &c.CreatedAt, &c.UpdatedAt)
	return c, err
}

func (r *Repository) loadSiteIDs(ctx context.Context, q pgxQuerier, ids []int64) (map[int64][]int64, error) {
	out := make(map[int64][]int64, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := q.Query(ctx, `SELECT contract_id, site_id FROM contract_sites WHERE contract_id = ANY($1) ORDER BY site_id`, ids)
	if err != nil {
		return nil, fmt.Errorf("load contract sites: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var cid, sid int64
		if err := rows.Scan(&cid, &sid); err != nil {
			return nil, fmt.Errorf("scan contract site: %w", err)
		}
		out[cid] = append(out[cid], sid)
	}
	return out, rows.Err()
}

type pgxQuerier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func (r *Repository) List(ctx context.Context, params pagination.Params, customerID int64) ([]Contract, int, error) {
	args := []any{}
	conds := []string{}
	if customerID > 0 {
		args = append(args, customerID)
		conds = append(conds, "customer_id = $"+strconv.Itoa(len(args)))
	}
	if params.Search != "" {
		args = append(args, "%"+params.Search+"%")
		conds = append(conds, "(contract_number ILIKE $"+strconv.Itoa(len(args))+" OR title ILIKE $"+strconv.Itoa(len(args))+")")
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
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM contracts `+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count contracts: %w", err)
	}
	args = append(args, params.Limit, params.Offset)
	query := `SELECT ` + contractColumns + ` FROM contracts ` + where +
		` ORDER BY end_date DESC LIMIT $` + strconv.Itoa(len(args)-1) + ` OFFSET $` + strconv.Itoa(len(args))
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query contracts: %w", err)
	}
	defer rows.Close()

	items := []Contract{}
	for rows.Next() {
		item, err := scanContract(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan contract: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	ids := make([]int64, 0, len(items))
	for _, c := range items {
		ids = append(ids, c.ID)
	}
	siteMap, err := r.loadSiteIDs(ctx, r.pool, ids)
	if err != nil {
		return nil, 0, err
	}
	for i := range items {
		items[i].SiteIDs = siteMap[items[i].ID]
		if items[i].SiteIDs == nil {
			items[i].SiteIDs = []int64{}
		}
	}
	return items, total, nil
}

func (r *Repository) GetByID(ctx context.Context, id int64) (Contract, error) {
	c, err := scanContract(r.pool.QueryRow(ctx, `SELECT `+contractColumns+` FROM contracts WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Contract{}, ErrNotFound
	}
	if err != nil {
		return Contract{}, fmt.Errorf("get contract %d: %w", id, err)
	}
	siteMap, err := r.loadSiteIDs(ctx, r.pool, []int64{id})
	if err != nil {
		return Contract{}, err
	}
	c.SiteIDs = siteMap[id]
	if c.SiteIDs == nil {
		c.SiteIDs = []int64{}
	}
	return c, nil
}

// assertSitesBelong ensures every site belongs to the contract's customer.
func assertSitesBelong(ctx context.Context, q queryRow, customerID int64, siteIDs []int64) error {
	if len(siteIDs) == 0 {
		return nil
	}
	var count int
	if err := q.QueryRow(ctx, `SELECT COUNT(*) FROM sites WHERE id = ANY($1) AND customer_id = $2`, siteIDs, customerID).Scan(&count); err != nil {
		return fmt.Errorf("check contract sites: %w", err)
	}
	if count != len(siteIDs) {
		return fmt.Errorf("one or more sites do not belong to this customer")
	}
	return nil
}

type queryRow interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func (r *Repository) Create(ctx context.Context, c Contract) (Contract, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Contract{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := assertSitesBelong(ctx, tx, c.CustomerID, c.SiteIDs); err != nil {
		return Contract{}, err
	}
	temp := "TMP"
	var id int64
	if err := tx.QueryRow(ctx,
		`INSERT INTO contracts (contract_number, customer_id, title, status, start_date, end_date, renewal_date, contract_value, billing_frequency, sla_terms, notes)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) RETURNING id`,
		temp, c.CustomerID, c.Title, c.Status, c.StartDate, c.EndDate, c.RenewalDate, c.ContractValue, c.BillingFrequency, c.SLATerms, c.Notes).Scan(&id); err != nil {
		return Contract{}, fmt.Errorf("create contract: %w", err)
	}
	for _, sid := range c.SiteIDs {
		if _, err := tx.Exec(ctx, `INSERT INTO contract_sites (contract_id, site_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, id, sid); err != nil {
			return Contract{}, fmt.Errorf("link contract site: %w", err)
		}
	}
	created, err := scanContract(tx.QueryRow(ctx,
		`UPDATE contracts SET contract_number = 'CT-' || to_char(created_at, 'YYYY') || '-' || lpad(id::text, 4, '0') WHERE id = $1 RETURNING `+contractColumns, id))
	if err != nil {
		return Contract{}, fmt.Errorf("assign contract number: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Contract{}, fmt.Errorf("commit contract: %w", err)
	}
	siteMap, err := r.loadSiteIDs(ctx, r.pool, []int64{id})
	if err != nil {
		return Contract{}, err
	}
	created.SiteIDs = siteMap[id]
	if created.SiteIDs == nil {
		created.SiteIDs = []int64{}
	}
	return created, nil
}

type Patch struct {
	Title            *string
	Status           *Status
	StartDate        *time.Time
	EndDate          *time.Time
	RenewalDate      *time.Time
	ClearRenewal     bool
	ContractValue    *float64
	BillingFrequency *BillingFrequency
	SLATerms         *string
	Notes            *string
	SiteIDs          []int64
	ReplaceSites     bool
}

func (r *Repository) Update(ctx context.Context, id int64, p Patch) (Contract, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Contract{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var customerID int64
	var startDate, endDate time.Time
	if err := tx.QueryRow(ctx, `SELECT customer_id, start_date, end_date FROM contracts WHERE id = $1`, id).Scan(&customerID, &startDate, &endDate); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Contract{}, ErrNotFound
		}
		return Contract{}, fmt.Errorf("get contract %d: %w", id, err)
	}
	if p.StartDate != nil {
		startDate = *p.StartDate
	}
	if p.EndDate != nil {
		endDate = *p.EndDate
	}
	if endDate.Before(startDate) {
		return Contract{}, fmt.Errorf("endDate must be on or after startDate")
	}
	if p.ReplaceSites {
		if err := assertSitesBelong(ctx, tx, customerID, p.SiteIDs); err != nil {
			return Contract{}, err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM contract_sites WHERE contract_id = $1`, id); err != nil {
			return Contract{}, fmt.Errorf("clear contract sites: %w", err)
		}
		for _, sid := range p.SiteIDs {
			if _, err := tx.Exec(ctx, `INSERT INTO contract_sites (contract_id, site_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, id, sid); err != nil {
				return Contract{}, fmt.Errorf("link contract site: %w", err)
			}
		}
	}

	var status any
	if p.Status != nil {
		status = *p.Status
	}
	var freq any
	if p.BillingFrequency != nil {
		freq = *p.BillingFrequency
	}
	updated, err := scanContract(tx.QueryRow(ctx,
		`UPDATE contracts SET
			title             = COALESCE($2, title),
			status            = COALESCE($3, status),
			start_date        = COALESCE($4, start_date),
			end_date          = COALESCE($5, end_date),
			renewal_date      = CASE WHEN $6 THEN NULL ELSE COALESCE($7, renewal_date) END,
			contract_value    = COALESCE($8, contract_value),
			billing_frequency = COALESCE($9, billing_frequency),
			sla_terms         = COALESCE($10, sla_terms),
			notes             = COALESCE($11, notes)
		 WHERE id = $1
		 RETURNING `+contractColumns,
		id, p.Title, status, p.StartDate, p.EndDate, p.ClearRenewal, p.RenewalDate, p.ContractValue, freq, p.SLATerms, p.Notes))
	if errors.Is(err, pgx.ErrNoRows) {
		return Contract{}, ErrNotFound
	}
	if err != nil {
		return Contract{}, fmt.Errorf("update contract %d: %w", id, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Contract{}, fmt.Errorf("commit contract %d: %w", id, err)
	}
	siteMap, err := r.loadSiteIDs(ctx, r.pool, []int64{id})
	if err != nil {
		return Contract{}, err
	}
	updated.SiteIDs = siteMap[id]
	if updated.SiteIDs == nil {
		updated.SiteIDs = []int64{}
	}
	return updated, nil
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM contracts WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete contract %d: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Renew retires a contract as renewed and opens its successor draft in one
// transaction: the old row keeps its history, the new row starts a fresh
// lifecycle for the next period.
func (r *Repository) Renew(ctx context.Context, id int64, start, end time.Time) (Contract, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Contract{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	src, err := scanContract(tx.QueryRow(ctx, `SELECT `+contractColumns+` FROM contracts WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Contract{}, ErrNotFound
	}
	if err != nil {
		return Contract{}, fmt.Errorf("get contract %d: %w", id, err)
	}
	switch src.Status {
	case StatusActive, StatusExpiring, StatusExpired:
	default:
		return Contract{}, fmt.Errorf("only active, expiring or expired contracts can be renewed")
	}
	if _, err := tx.Exec(ctx, `UPDATE contracts SET status = 'renewed' WHERE id = $1`, id); err != nil {
		return Contract{}, fmt.Errorf("retire contract %d: %w", id, err)
	}
	var newID int64
	if err := tx.QueryRow(ctx,
		`INSERT INTO contracts (contract_number, customer_id, title, status, start_date, end_date, contract_value, billing_frequency, sla_terms, notes)
		 VALUES ('TMP', $1, $2, 'draft', $3, $4, $5, $6, $7, $8) RETURNING id`,
		src.CustomerID, src.Title, start, end, src.ContractValue, src.BillingFrequency, src.SLATerms, src.Notes).Scan(&newID); err != nil {
		return Contract{}, fmt.Errorf("create successor contract: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO contract_sites (contract_id, site_id)
		 SELECT $1, site_id FROM contract_sites WHERE contract_id = $2 ON CONFLICT DO NOTHING`,
		newID, id); err != nil {
		return Contract{}, fmt.Errorf("copy contract sites: %w", err)
	}
	next, err := scanContract(tx.QueryRow(ctx,
		`UPDATE contracts SET contract_number = 'CT-' || to_char(created_at, 'YYYY') || '-' || lpad(id::text, 4, '0') WHERE id = $1 RETURNING `+contractColumns, newID))
	if err != nil {
		return Contract{}, fmt.Errorf("assign contract number: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Contract{}, fmt.Errorf("commit contract renewal: %w", err)
	}
	siteMap, err := r.loadSiteIDs(ctx, r.pool, []int64{newID})
	if err != nil {
		return Contract{}, err
	}
	next.SiteIDs = siteMap[newID]
	if next.SiteIDs == nil {
		next.SiteIDs = []int64{}
	}
	return next, nil
}
