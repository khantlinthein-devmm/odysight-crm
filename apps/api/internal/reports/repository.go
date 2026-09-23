package reports

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) LoadSummary(ctx context.Context, now time.Time) (Summary, error) {
	var row summaryRow
	err := r.pool.QueryRow(ctx,
		`SELECT
			(SELECT COUNT(*) FROM leads),
			(SELECT COUNT(*) FROM customers WHERE status = 'active'),
			(SELECT COUNT(*) FROM bookings WHERE status IN ('pending', 'confirmed', 'in_progress')),
			(SELECT COALESCE(SUM(amount), 0) FROM payments
			 WHERE status = 'paid'
			   AND created_at >= date_trunc('month', $1::timestamptz))`,
		now).Scan(&row.totalLeads, &row.activeCustomers, &row.upcomingBookings, &row.monthlyRevenue)
	if err != nil {
		return Summary{}, fmt.Errorf("load summary totals: %w", err)
	}

	counts, err := r.loadLeadsByStatus(ctx)
	if err != nil {
		return Summary{}, err
	}

	bookingCounts, err := r.loadBookingsByStatus(ctx)
	if err != nil {
		return Summary{}, err
	}

	revenue, err := r.loadRevenueByMonth(ctx, now)
	if err != nil {
		return Summary{}, err
	}

	productivity, err := r.loadCleanerProductivity(ctx)
	if err != nil {
		return Summary{}, err
	}

	return buildSummary(row, counts, revenue, bookingCounts, productivity), nil
}

func (r *Repository) LoadFinancial(ctx context.Context, from, to time.Time, currency string) (FinancialReport, error) {
	report := FinancialReport{
		From:     from,
		To:       to,
		Currency: currency,
		ARAging:  make([]ARAgingBucket, 0, 5),
	}

	revenue, err := r.loadRevenueBuckets(ctx, from, to, currency)
	if err != nil {
		return report, err
	}

	tax, err := r.loadTaxSummary(ctx, from, to, currency)
	if err != nil {
		return report, err
	}

	aging, err := r.loadARAging(ctx, currency)
	if err != nil {
		return report, err
	}

	report.Revenue = revenue
	report.Tax = tax
	report.ARAging = aging
	return report, nil
}

// loadRevenueBuckets aggregates billed vs collected amounts per month.
func (r *Repository) loadRevenueBuckets(ctx context.Context, from, to time.Time, currency string) ([]RevenueBucket, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT m.period,
		        COALESCE(billed.billed, 0),
		        COALESCE(collected.collected, 0)
		 FROM generate_series($1::timestamptz, $2::timestamptz, interval '1 month') AS m(period)
		 LEFT JOIN (
		     SELECT date_trunc('month', issued_at) AS month, SUM(total) AS billed
		     FROM invoices
		     WHERE status <> 'void' AND currency = $3 AND issued_at BETWEEN $1 AND $2
		     GROUP BY 1
		 ) billed ON billed.month = m.period
		 LEFT JOIN (
		     SELECT date_trunc('month', p.created_at) AS month, SUM(p.amount) AS collected
		     FROM payments p
		     JOIN invoices i ON i.invoice_number = p.invoice_number AND i.currency = $3
		     WHERE p.status = 'paid' AND p.created_at BETWEEN $1 AND $2
		     GROUP BY 1
		 ) collected ON collected.month = m.period
		 ORDER BY m.period`,
		from, to, currency)
	if err != nil {
		return nil, fmt.Errorf("query revenue buckets: %w", err)
	}
	defer rows.Close()

	buckets := []RevenueBucket{}
	for rows.Next() {
		var period time.Time
		var billed, collected float64
		if err := rows.Scan(&period, &billed, &collected); err != nil {
			return nil, fmt.Errorf("scan revenue bucket: %w", err)
		}
		buckets = append(buckets, RevenueBucket{
			Period:      period.Format("2006-01"),
			Billed:      billed,
			Collected:   collected,
			Outstanding: billed - collected,
		})
	}
	return buckets, rows.Err()
}

// loadTaxSummary aggregates tax billed, tax collected, and tax outstanding.
func (r *Repository) loadTaxSummary(ctx context.Context, from, to time.Time, currency string) (TaxSummary, error) {
	var s TaxSummary
	rows, err := r.pool.Query(ctx,
		`SELECT tax_rate,
		        COALESCE(SUM(total), 0)             AS billed_total,
		        COALESCE(SUM(tax_amount), 0)        AS tax_billed,
		        COALESCE(SUM(total) FILTER (WHERE status = 'paid'), 0)   AS collected_total,
		        COALESCE(SUM(tax_amount) FILTER (WHERE status = 'paid'), 0) AS tax_collected
		 FROM invoices
		 WHERE status <> 'void' AND currency = $3 AND issued_at BETWEEN $1 AND $2
		 GROUP BY tax_rate
		 ORDER BY tax_rate`,
		from, to, currency)
	if err != nil {
		return TaxSummary{}, fmt.Errorf("query tax summary: %w", err)
	}
	defer rows.Close()

	hasRows := false
	for rows.Next() {
		var rate, billed, taxBilled, collected, taxCollected float64
		if err := rows.Scan(&rate, &billed, &taxBilled, &collected, &taxCollected); err != nil {
			return TaxSummary{}, fmt.Errorf("scan tax summary: %w", err)
		}
		hasRows = true
		s.TaxRate = rate
		s.BilledTotal += billed
		s.TaxBilled += taxBilled
		s.CollectedTotal += collected
		s.TaxCollected += taxCollected
	}
	if err := rows.Err(); err != nil {
		return TaxSummary{}, err
	}
	s.TaxOutstanding = s.TaxBilled - s.TaxCollected
	if !hasRows {
		s.TaxRate = defaultTaxRate(currency)
	}
	return s, nil
}

// loadARAging buckets open (unpaid, non-void) invoices by days past due.
func (r *Repository) loadARAging(ctx context.Context, currency string) ([]ARAgingBucket, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT
		     CASE
		         WHEN $2::timestamptz <= issued_at THEN 'Current'
		         WHEN $2 - issued_at <= interval '30 days' THEN '1-30'
		         WHEN $2 - issued_at <= interval '60 days' THEN '31-60'
		         WHEN $2 - issued_at <= interval '90 days' THEN '61-90'
		         ELSE '90+'
		     END AS bucket,
		     COALESCE(SUM(total), 0),
		     COUNT(*)
		 FROM invoices
		 WHERE currency = $1 AND status = 'issued'
		 GROUP BY 1`,
		currency, time.Now())
	if err != nil {
		return nil, fmt.Errorf("query AR aging: %w", err)
	}
	defer rows.Close()

	byLabel := map[string]*ARAgingBucket{}
	order := []string{"Current", "1-30", "31-60", "61-90", "90+"}
	for rows.Next() {
		var label string
		var amount float64
		var count int64
		if err := rows.Scan(&label, &amount, &count); err != nil {
			return nil, fmt.Errorf("scan AR aging: %w", err)
		}
		byLabel[label] = &ARAgingBucket{Label: label, Amount: amount, Count: count}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]ARAgingBucket, 0, len(order))
	for _, label := range order {
		if b, ok := byLabel[label]; ok {
			out = append(out, *b)
		} else {
			out = append(out, ARAgingBucket{Label: label})
		}
	}
	return out, nil
}

func (r *Repository) LoadCommercial(ctx context.Context) (CommercialReport, error) {
	report := CommercialReport{
		RevenueBySite:     []SiteRevenue{},
		RevenueByContract: []ContractRevenue{},
	}

	siteRows, err := r.pool.Query(ctx,
		`SELECT s.id, s.customer_id, s.name,
		        COUNT(DISTINCT b.id) AS bookings,
		        COUNT(DISTINCT CASE WHEN b.status = 'completed' THEN b.id END) AS completed,
		        COALESCE(SUM(i.total) FILTER (WHERE i.status <> 'void'), 0) AS billed
		 FROM sites s
		 LEFT JOIN bookings b ON b.site_id = s.id
		 LEFT JOIN invoices i ON i.booking_id = b.id
		 GROUP BY s.id, s.customer_id, s.name
		 ORDER BY billed DESC, bookings DESC`)
	if err != nil {
		return report, fmt.Errorf("query revenue by site: %w", err)
	}
	for siteRows.Next() {
		var row SiteRevenue
		if err := siteRows.Scan(&row.SiteID, &row.CustomerID, &row.SiteName, &row.Bookings, &row.Completed, &row.Billed); err != nil {
			siteRows.Close()
			return report, fmt.Errorf("scan revenue by site: %w", err)
		}
		report.RevenueBySite = append(report.RevenueBySite, row)
	}
	siteRows.Close()
	if err := siteRows.Err(); err != nil {
		return report, err
	}

	contractRows, err := r.pool.Query(ctx,
		`SELECT c.id, c.contract_number, c.title, c.status, c.contract_value,
		        (SELECT COUNT(*) FROM bookings b WHERE b.contract_id = c.id) AS bookings,
		        (SELECT COALESCE(SUM(total), 0) FROM invoices i WHERE i.contract_id = c.id AND i.status <> 'void') AS billed
		 FROM contracts c
		 ORDER BY c.end_date DESC`)
	if err != nil {
		return report, fmt.Errorf("query revenue by contract: %w", err)
	}
	for contractRows.Next() {
		var row ContractRevenue
		if err := contractRows.Scan(&row.ContractID, &row.ContractNumber, &row.Title, &row.Status, &row.ContractValue, &row.Bookings, &row.Billed); err != nil {
			contractRows.Close()
			return report, fmt.Errorf("scan revenue by contract: %w", err)
		}
		report.RevenueByContract = append(report.RevenueByContract, row)
	}
	contractRows.Close()
	if err := contractRows.Err(); err != nil {
		return report, err
	}

	var total, accepted int64
	if err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*), COUNT(*) FILTER (WHERE status = 'accepted') FROM quotes`).Scan(&total, &accepted); err != nil {
		return report, fmt.Errorf("query quote win rate: %w", err)
	}
	report.QuoteWinRate = QuoteWinRate{Total: total, Accepted: accepted}
	if total > 0 {
		rate := float64(accepted) / float64(total) * 100
		report.QuoteWinRate.Rate = &rate
	}

	var lists, listsDone, itemsTotal, itemsDone int64
	if err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*), COUNT(*) FILTER (WHERE status = 'completed'),
		        COALESCE((SELECT COUNT(*) FROM booking_checklist_items), 0),
		        COALESCE((SELECT COUNT(*) FROM booking_checklist_items WHERE is_completed), 0)
		 FROM booking_checklists`).Scan(&lists, &listsDone, &itemsTotal, &itemsDone); err != nil {
		return report, fmt.Errorf("query checklist stats: %w", err)
	}
	report.Checklists = ChecklistStats{Total: lists, Completed: listsDone, ItemsTotal: itemsTotal, ItemsDone: itemsDone}
	if itemsTotal > 0 {
		pc := float64(itemsDone) / float64(itemsTotal) * 100
		report.Checklists.CompletionPC = &pc
	}
	return report, nil
}

// defaultTaxRate is used when no invoices exist in the period.
func defaultTaxRate(currency string) float64 {
	// THB VAT is 7% (VAT on services). Other currencies use 0 by default.
	if currency == "THB" {
		return 7
	}
	return 0
}

func (r *Repository) loadCleanerProductivity(ctx context.Context) ([]CleanerProductivity, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT CASE
		           WHEN cl.first_name IS NOT NULL THEN trim(cl.first_name || ' ' || cl.last_name)
		           WHEN NULLIF(b.assigned_cleaner, '') IS NOT NULL THEN b.assigned_cleaner
		           ELSE 'Unassigned'
		       END AS cleaner_name,
		        COUNT(*) FILTER (WHERE b.status = 'completed') AS completed,
		        COUNT(*) FILTER (WHERE b.status IN ('pending', 'confirmed', 'in_progress')) AS upcoming
		 FROM bookings b
		 LEFT JOIN booking_cleaners bc
		        ON bc.booking_id = b.id AND bc.role = 'primary'
		 LEFT JOIN cleaners cl ON cl.id = bc.cleaner_id
		 WHERE bc.cleaner_id IS NOT NULL OR b.assigned_cleaner IS NOT NULL
		 GROUP BY cleaner_name
		 ORDER BY completed DESC, upcoming DESC, cleaner_name ASC
		 LIMIT 10`)
	if err != nil {
		return nil, fmt.Errorf("query cleaner productivity: %w", err)
	}
	defer rows.Close()

	productivity := []CleanerProductivity{}
	for rows.Next() {
		var p CleanerProductivity
		if err := rows.Scan(&p.CleanerName, &p.CompletedBookings, &p.UpcomingBookings); err != nil {
			return nil, fmt.Errorf("scan cleaner productivity: %w", err)
		}
		productivity = append(productivity, p)
	}
	return productivity, rows.Err()
}

func (r *Repository) loadLeadsByStatus(ctx context.Context) (map[string]int64, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT status, COUNT(*) FROM leads GROUP BY status`)
	if err != nil {
		return nil, fmt.Errorf("query leads by status: %w", err)
	}
	defer rows.Close()

	counts := map[string]int64{}
	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("scan lead status count: %w", err)
		}
		counts[status] = count
	}
	return counts, rows.Err()
}

func (r *Repository) loadBookingsByStatus(ctx context.Context) (map[string]int64, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT status, COUNT(*) FROM bookings GROUP BY status`)
	if err != nil {
		return nil, fmt.Errorf("query bookings by status: %w", err)
	}
	defer rows.Close()

	counts := map[string]int64{}
	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("scan booking status count: %w", err)
		}
		counts[status] = count
	}
	return counts, rows.Err()
}

func (r *Repository) loadRevenueByMonth(ctx context.Context, now time.Time) ([]MonthlyRevenue, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT m.month_start, COALESCE(SUM(p.amount), 0) AS collected
		 FROM generate_series(
			 date_trunc('month', $1::timestamptz) - interval '5 months',
			 date_trunc('month', $1::timestamptz),
			 interval '1 month'
		 ) AS m(month_start)
		 LEFT JOIN payments p
			ON p.status = 'paid'
		   AND p.created_at >= m.month_start
		   AND p.created_at < m.month_start + interval '1 month'
		 GROUP BY m.month_start
		 ORDER BY m.month_start`,
		now)
	if err != nil {
		return nil, fmt.Errorf("query revenue by month: %w", err)
	}
	defer rows.Close()

	revenue := []MonthlyRevenue{}
	for rows.Next() {
		var monthStart time.Time
		var collected float64
		if err := rows.Scan(&monthStart, &collected); err != nil {
			return nil, fmt.Errorf("scan monthly revenue: %w", err)
		}
		revenue = append(revenue, MonthlyRevenue{
			Month:     monthLabel(monthStart),
			Collected: collected,
		})
	}
	return revenue, rows.Err()
}
