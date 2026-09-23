package reports

// SiteRevenue aggregates bookings and billed revenue per site.
type SiteRevenue struct {
	SiteID     int64   `json:"siteId"`
	CustomerID int64   `json:"customerId"`
	SiteName   string  `json:"siteName"`
	Bookings   int64   `json:"bookings"`
	Completed  int64   `json:"completed"`
	Billed     float64 `json:"billed"`
}

// ContractRevenue aggregates bookings and billed revenue per contract.
type ContractRevenue struct {
	ContractID     int64   `json:"contractId"`
	ContractNumber string  `json:"contractNumber"`
	Title          string  `json:"title"`
	Status         string  `json:"status"`
	ContractValue  float64 `json:"contractValue"`
	Bookings       int64   `json:"bookings"`
	Billed         float64 `json:"billed"`
}

// QuoteWinRate summarises quote outcomes.
type QuoteWinRate struct {
	Total    int64   `json:"total"`
	Accepted int64   `json:"accepted"`
	Rate     *float64 `json:"rate"`
}

// ChecklistStats summarises proof-of-work completion.
type ChecklistStats struct {
	Total        int64   `json:"total"`
	Completed    int64   `json:"completed"`
	ItemsTotal   int64   `json:"itemsTotal"`
	ItemsDone    int64   `json:"itemsDone"`
	CompletionPC *float64 `json:"completionRate"`
}

// CommercialReport is the output of GET /api/v1/reports/commercial.
type CommercialReport struct {
	RevenueBySite     []SiteRevenue     `json:"revenueBySite"`
	RevenueByContract []ContractRevenue `json:"revenueByContract"`
	QuoteWinRate      QuoteWinRate      `json:"quoteWinRate"`
	Checklists        ChecklistStats    `json:"checklists"`
}
