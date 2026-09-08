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

type BookingStatusCount struct {
	Status string `json:"status"`
	Count  int64  `json:"count"`
}

type CleanerProductivity struct {
	CleanerName       string `json:"cleanerName"`
	CompletedBookings int64  `json:"completedBookings"`
	UpcomingBookings  int64  `json:"upcomingBookings"`
}

type Summary struct {
	TotalLeads          int64               `json:"totalLeads"`
	ActiveCustomers     int64               `json:"activeCustomers"`
	UpcomingBookings    int64               `json:"upcomingBookings"`
	MonthlyRevenue      float64             `json:"monthlyRevenue"`
	LeadsByStatus       []LeadStatusCount   `json:"leadsByStatus"`
	RevenueByMonth      []MonthlyRevenue    `json:"revenueByMonth"`
	BookingsByStatus    []BookingStatusCount `json:"bookingsByStatus"`
	LeadConversionRate  *float64            `json:"leadConversionRate"`
	CleanerProductivity []CleanerProductivity `json:"cleanerProductivity"`
}

var leadStatusOrder = []string{"new", "contacted", "quote_sent", "booked", "won", "lost"}

var leadStatusLabel = map[string]string{
	"new":        "New",
	"contacted":  "Contacted",
	"quote_sent": "Quote Sent",
	"booked":     "Booked",
	"won":        "Won",
	"lost":       "Lost",
}

var bookingStatusOrder = []string{"pending", "confirmed", "in_progress", "completed", "cancelled", "no_show", "rescheduled"}

var bookingStatusLabel = map[string]string{
	"pending":     "Pending",
	"confirmed":   "Confirmed",
	"in_progress": "In Progress",
	"completed":   "Completed",
	"cancelled":   "Cancelled",
	"no_show":     "No Show",
	"rescheduled": "Rescheduled",
}

type summaryRow struct {
	totalLeads       int64
	activeCustomers  int64
	upcomingBookings int64
	monthlyRevenue   float64
}

func buildSummary(row summaryRow, counts map[string]int64, revenue []MonthlyRevenue, bookingCounts map[string]int64, productivity []CleanerProductivity) Summary {
	summary := Summary{
		TotalLeads:          row.totalLeads,
		ActiveCustomers:     row.activeCustomers,
		UpcomingBookings:    row.upcomingBookings,
		MonthlyRevenue:      row.monthlyRevenue,
		LeadsByStatus:       make([]LeadStatusCount, 0, len(leadStatusOrder)),
		RevenueByMonth:      revenue,
		BookingsByStatus:    make([]BookingStatusCount, 0, len(bookingStatusOrder)),
		CleanerProductivity: productivity,
	}
	if row.totalLeads > 0 {
		rate := (float64(counts["won"]) / float64(row.totalLeads)) * 100
		summary.LeadConversionRate = &rate
	}
	for _, status := range leadStatusOrder {
		summary.LeadsByStatus = append(summary.LeadsByStatus, LeadStatusCount{
			Status: leadStatusLabel[status],
			Count:  counts[status],
		})
	}
	for _, status := range bookingStatusOrder {
		summary.BookingsByStatus = append(summary.BookingsByStatus, BookingStatusCount{
			Status: bookingStatusLabel[status],
			Count:  bookingCounts[status],
		})
	}
	return summary
}

// monthLabel formats a time as a short month name, e.g. "Aug".
func monthLabel(t time.Time) string {
	return t.Format("Jan")
}
