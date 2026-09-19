package expenses

import "time"

// Expense is one office/operations spend row backing the finance page.
type Expense struct {
	ID        int64
	SpentOn   string // YYYY-MM-DD
	Category  string
	Amount    float64
	Note      string
	CreatedBy int64 // 0 once the creating user is deleted (created_by is nulled)
	CreatedAt time.Time
}
