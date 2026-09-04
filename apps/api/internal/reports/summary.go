package reports

import (
	"time"
)

type LeadStatusCount struct {
	Status string `json:"status"`
	Count  int64  `json:"count"`
}

type MonthlyRevenue struct {
	Month     string  `json:"month"`
	Collected float64 `json:"collected"`
}

type Summary struct {
	TotalLeads       int64             `json:"totalLeads"`
	ActiveApplicants int64             `json:"activeApplicants"`
	OpenVisaCases    int64             `json:"openVisaCases"`
	MonthlyRevenue   float64           `json:"monthlyRevenue"`
	LeadsByStatus    []LeadStatusCount `json:"leadsByStatus"`
	RevenueByMonth   []MonthlyRevenue  `json:"revenueByMonth"`
}

var leadStatusOrder = []string{"new", "contacted", "qualified", "proposal", "won", "lost"}

var leadStatusLabel = map[string]string{
	"new":       "New",
	"contacted": "Contacted",
	"qualified": "Qualified",
	"proposal":  "Proposal",
	"won":       "Won",
	"lost":      "Lost",
}

type summaryRow struct {
	totalLeads       int64
	activeApplicants int64
	openVisaCases    int64
	monthlyRevenue   float64
}

func buildSummary(row summaryRow, counts map[string]int64, revenue []MonthlyRevenue) Summary {
	summary := Summary{
		TotalLeads:       row.totalLeads,
		ActiveApplicants: row.activeApplicants,
		OpenVisaCases:    row.openVisaCases,
		MonthlyRevenue:   row.monthlyRevenue,
		LeadsByStatus:    make([]LeadStatusCount, 0, len(leadStatusOrder)),
		RevenueByMonth:   revenue,
	}
	for _, status := range leadStatusOrder {
		summary.LeadsByStatus = append(summary.LeadsByStatus, LeadStatusCount{
			Status: leadStatusLabel[status],
			Count:  counts[status],
		})
	}
	return summary
}

// monthLabel formats a time as a short month name, e.g. "Aug".
func monthLabel(t time.Time) string {
	return t.Format("Jan")
}
