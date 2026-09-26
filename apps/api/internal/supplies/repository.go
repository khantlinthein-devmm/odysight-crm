// Package supplies keeps the cleaning-supplies catalog, its stock ledger
// and supply cost per site.
package supplies

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound  = errors.New("supply not found")
	ErrDuplicate = errors.New("a supply with that name already exists")
)

// InsufficientStock is returned when usage exceeds what is on hand.
type InsufficientStock struct{ Available float64 }

func (e *InsufficientStock) Error() string {
	return fmt.Sprintf("only %s in stock; record a purchase or adjustment first", trimNum(e.Available))
}

func trimNum(v float64) string {
	return strings.TrimSuffix(strings.TrimRight(fmt.Sprintf("%.2f", v), "0"), ".")
}

type Supply struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	Unit         string    `json:"unit"`
	UnitCost     float64   `json:"unitCost"`
	StockQty     float64   `json:"stockQty"`
	ReorderLevel float64   `json:"reorderLevel"`
	Active       bool      `json:"active"`
	LowStock     bool      `json:"lowStock"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type Movement struct {
	ID            int64     `json:"id"`
	SupplyID      int64     `json:"supplyId"`
	SupplyName    string    `json:"supplyName"`
	Unit          string    `json:"unit"`
	Kind          string    `json:"kind"`
	Quantity      float64   `json:"quantity"`
	UnitCost      float64   `json:"unitCost"`
	Cost          float64   `json:"cost"`
	SiteID        *int64    `json:"siteId"`
	SiteName      string    `json:"siteName"`
	BookingID     *int64    `json:"bookingId"`
	BookingNumber string    `json:"bookingNumber"`
	Note          string    `json:"note"`
	CreatedAt     time.Time `json:"createdAt"`
}

type SiteCost struct {
	SiteID       int64   `json:"siteId"`
	SiteName     string  `json:"siteName"`
	CustomerName string  `json:"customerName"`
	SupplyCost   float64 `json:"supplyCost"`
	Revenue      float64 `json:"revenue"`
	// CostShare is supply cost as a percentage of revenue (0 without revenue).
	CostShare float64 `json:"costShare"`
}

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const supplyColumns = `id, name, unit, unit_cost::float8, stock_qty::float8, reorder_level::float8, active, updated_at`

func scanSupply(row pgx.Row) (Supply, error) {
	var s Supply
	err := row.Scan(&s.ID, &s.Name, &s.Unit, &s.UnitCost, &s.StockQty, &s.ReorderLevel, &s.Active, &s.UpdatedAt)
	s.LowStock = s.Active && s.ReorderLevel > 0 && s.StockQty <= s.ReorderLevel
	return s, err
}

func mapWrite(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrDuplicate
	}
	return err
}

func (r *Repository) List(ctx context.Context, includeInactive bool) ([]Supply, error) {
	where := "WHERE active"
	if includeInactive {
		where = ""
	}
	rows, err := r.pool.Query(ctx, `SELECT `+supplyColumns+` FROM supplies `+where+` ORDER BY lower(name)`)
	if err != nil {
		return nil, fmt.Errorf("list supplies: %w", err)
	}
	defer rows.Close()
	out := []Supply{}
	for rows.Next() {
		s, err := scanSupply(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *Repository) Get(ctx context.Context, id int64) (Supply, error) {
	s, err := scanSupply(r.pool.QueryRow(ctx, `SELECT `+supplyColumns+` FROM supplies WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Supply{}, ErrNotFound
	}
	return s, err
}

func (r *Repository) Create(ctx context.Context, s Supply) (Supply, error) {
	out, err := scanSupply(r.pool.QueryRow(ctx,
		`INSERT INTO supplies (name, unit, unit_cost, stock_qty, reorder_level)
		 VALUES ($1, $2, $3, $4, $5) RETURNING `+supplyColumns,
		s.Name, s.Unit, s.UnitCost, s.StockQty, s.ReorderLevel))
	return out, mapWrite(err)
}

type Patch struct {
	Name         *string
	Unit         *string
	UnitCost     *float64
	ReorderLevel *float64
	Active       *bool
}

func (r *Repository) Update(ctx context.Context, id int64, p Patch) (Supply, error) {
	s, err := scanSupply(r.pool.QueryRow(ctx,
		`UPDATE supplies SET name = COALESCE($2, name), unit = COALESCE($3, unit),
		        unit_cost = COALESCE($4, unit_cost), reorder_level = COALESCE($5, reorder_level),
		        active = COALESCE($6, active), updated_at = now()
		  WHERE id = $1 RETURNING `+supplyColumns,
		id, p.Name, p.Unit, p.UnitCost, p.ReorderLevel, p.Active))
	if errors.Is(err, pgx.ErrNoRows) {
		return Supply{}, ErrNotFound
	}
	return s, mapWrite(err)
}

// BookingSite returns a booking's site, falling back to its customer's
// Default Site (else their first site) for one-time jobs booked without
// one; nil when the customer has no site at all.
func (r *Repository) BookingSite(ctx context.Context, bookingID int64) (*int64, error) {
	var site *int64
	err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(b.site_id,
		        (SELECT s.id FROM sites s WHERE s.customer_id = b.customer_id
		          ORDER BY s.is_default DESC, s.id LIMIT 1))
		   FROM bookings b WHERE b.id = $1`, bookingID).Scan(&site)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return site, err
}

// Record writes a movement and applies it to stock in one transaction. A
// purchase with a unit cost also becomes the item's current unit cost.
func (r *Repository) Record(ctx context.Context, m Movement, userID int64) (Movement, Supply, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Movement{}, Supply{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var cost, stock float64
	if err := tx.QueryRow(ctx, `SELECT unit_cost::float8, stock_qty::float8 FROM supplies WHERE id = $1 FOR UPDATE`,
		m.SupplyID).Scan(&cost, &stock); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Movement{}, Supply{}, ErrNotFound
		}
		return Movement{}, Supply{}, err
	}
	if m.Kind == "usage" && stock+m.Quantity < -0.0001 {
		return Movement{}, Supply{}, &InsufficientStock{Available: stock}
	}
	if m.UnitCost <= 0 {
		m.UnitCost = cost
	}
	if err := tx.QueryRow(ctx,
		`INSERT INTO supply_movements (supply_id, kind, quantity, unit_cost, site_id, booking_id, note, created_by)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, NULLIF($8, 0)) RETURNING id, created_at`,
		m.SupplyID, m.Kind, m.Quantity, m.UnitCost, m.SiteID, m.BookingID, m.Note, userID).Scan(&m.ID, &m.CreatedAt); err != nil {
		return Movement{}, Supply{}, fmt.Errorf("record supply movement: %w", err)
	}
	setCost := m.Kind == "purchase"
	s, err := scanSupply(tx.QueryRow(ctx,
		`UPDATE supplies SET stock_qty = stock_qty + $2,
		        unit_cost = CASE WHEN $3 THEN $4 ELSE unit_cost END, updated_at = now()
		  WHERE id = $1 RETURNING `+supplyColumns, m.SupplyID, m.Quantity, setCost, m.UnitCost))
	if err != nil {
		return Movement{}, Supply{}, fmt.Errorf("apply supply movement: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Movement{}, Supply{}, err
	}
	m.Cost = round2(abs(m.Quantity) * m.UnitCost)
	m.SupplyName, m.Unit = s.Name, s.Unit
	return m, s, nil
}

type MovementFilters struct {
	From, To time.Time
	SiteID   int64
	Kind     string
}

func (r *Repository) Movements(ctx context.Context, f MovementFilters) ([]Movement, error) {
	conds := []string{"m.created_at >= $1", "m.created_at < $2"}
	args := []any{f.From, f.To}
	if f.SiteID > 0 {
		args = append(args, f.SiteID)
		conds = append(conds, fmt.Sprintf("m.site_id = $%d", len(args)))
	}
	if f.Kind != "" {
		args = append(args, f.Kind)
		conds = append(conds, fmt.Sprintf("m.kind = $%d", len(args)))
	}
	rows, err := r.pool.Query(ctx,
		`SELECT m.id, m.supply_id, s.name, s.unit, m.kind, m.quantity::float8, m.unit_cost::float8,
		        m.site_id, COALESCE(si.name, ''), m.booking_id, COALESCE(b.booking_number, ''), m.note, m.created_at
		   FROM supply_movements m
		   JOIN supplies s ON s.id = m.supply_id
		   LEFT JOIN sites si ON si.id = m.site_id
		   LEFT JOIN bookings b ON b.id = m.booking_id
		  WHERE `+strings.Join(conds, " AND ")+`
		  ORDER BY m.created_at DESC LIMIT 500`, args...)
	if err != nil {
		return nil, fmt.Errorf("list supply movements: %w", err)
	}
	defer rows.Close()
	out := []Movement{}
	for rows.Next() {
		var m Movement
		if err := rows.Scan(&m.ID, &m.SupplyID, &m.SupplyName, &m.Unit, &m.Kind, &m.Quantity, &m.UnitCost,
			&m.SiteID, &m.SiteName, &m.BookingID, &m.BookingNumber, &m.Note, &m.CreatedAt); err != nil {
			return nil, err
		}
		m.Cost = round2(abs(m.Quantity) * m.UnitCost)
		out = append(out, m)
	}
	return out, rows.Err()
}

// SiteCosts sums supply usage cost and invoiced revenue (excluding VAT and
// void invoices) per site for [from, to).
func (r *Repository) SiteCosts(ctx context.Context, from, to time.Time) ([]SiteCost, error) {
	rows, err := r.pool.Query(ctx,
		`WITH cost AS (
		   SELECT site_id, SUM(-quantity * unit_cost) AS supply_cost
		     FROM supply_movements
		    WHERE kind = 'usage' AND site_id IS NOT NULL AND created_at >= $1 AND created_at < $2
		    GROUP BY site_id
		 ), rev AS (
		   SELECT b.site_id, SUM(i.subtotal) AS revenue
		     FROM invoices i JOIN bookings b ON b.id = i.booking_id
		    WHERE i.status <> 'void' AND b.site_id IS NOT NULL AND i.issued_at >= $1 AND i.issued_at < $2
		    GROUP BY b.site_id
		 )
		 SELECT s.id, s.name, TRIM(c.first_name || ' ' || c.last_name),
		        COALESCE(cost.supply_cost, 0)::float8, COALESCE(rev.revenue, 0)::float8
		   FROM sites s
		   JOIN customers c ON c.id = s.customer_id
		   LEFT JOIN cost ON cost.site_id = s.id
		   LEFT JOIN rev ON rev.site_id = s.id
		  WHERE cost.site_id IS NOT NULL OR rev.site_id IS NOT NULL
		  ORDER BY COALESCE(cost.supply_cost, 0) DESC, s.name`, from, to)
	if err != nil {
		return nil, fmt.Errorf("site supply costs: %w", err)
	}
	defer rows.Close()
	out := []SiteCost{}
	for rows.Next() {
		var sc SiteCost
		if err := rows.Scan(&sc.SiteID, &sc.SiteName, &sc.CustomerName, &sc.SupplyCost, &sc.Revenue); err != nil {
			return nil, err
		}
		sc.SupplyCost, sc.Revenue = round2(sc.SupplyCost), round2(sc.Revenue)
		if sc.Revenue > 0 {
			sc.CostShare = round2(sc.SupplyCost / sc.Revenue * 100)
		}
		out = append(out, sc)
	}
	return out, rows.Err()
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func round2(v float64) float64 {
	if v < 0 {
		return -round2(-v)
	}
	return float64(int64(v*100+0.5)) / 100
}
