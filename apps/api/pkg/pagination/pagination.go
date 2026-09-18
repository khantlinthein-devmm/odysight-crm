package pagination

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Params carries list query options.
type Params struct {
	Limit     int
	Offset    int
	Search    string
	Status    string
	Sort      string // e.g. "-created_at" or "scheduled_for"
	From      string // ISO timestamp lower bound (inclusive)
	To        string // ISO timestamp upper bound (exclusive)
	CleanerID int64  // optional filter for a related resource id (e.g. cleaner)
	Area      string // optional area filter (e.g. "Bang Na") for dispatch
	// Available restricts bookings to the unassigned pool (status=pending,
	// no cleaner assigned). Used by GET /bookings/available.
	Available bool
	// OnlineOnly restricts cleaners to those currently online.
	// Used by dispatch to find cleaners in an area (?area=X&online=true).
	OnlineOnly bool
}

// Page is a paginated envelope.
type Page[T any] struct {
	Data   []T `json:"data"`
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

// Parse extracts pagination/filter params with safe defaults.
func Parse(r *http.Request, defLimit int, allowedSort map[string]bool) Params {
	if defLimit <= 0 {
		defLimit = 20
	}
	q := r.URL.Query()
	limit := atoiBounded(q.Get("limit"), defLimit, 1, 200)
	offset := atoiBounded(q.Get("offset"), 0, 0, 1_000_000)
	search := strings.TrimSpace(q.Get("search"))
	status := strings.TrimSpace(q.Get("status"))
	sort := strings.TrimSpace(q.Get("sort"))
	if sort != "" && allowedSort != nil && !allowedSort[strings.TrimPrefix(sort, "-")] {
		sort = ""
	}
	cleanerID := int64(0)
	if v, err := strconv.ParseInt(q.Get("cleaner"), 10, 64); err == nil && v > 0 {
		cleanerID = v
	}
	area := strings.TrimSpace(q.Get("area"))
	available := q.Get("available") == "true" || q.Get("available") == "1"
	onlineOnly := q.Get("online") == "true" || q.Get("online") == "1"
	return Params{Limit: limit, Offset: offset, Search: search, Status: status, Sort: sort,
		From: trimToISO(q.Get("from")), To: trimToISO(q.Get("to")), CleanerID: cleanerID,
		Area: area, Available: available, OnlineOnly: onlineOnly}
}

func trimToISO(s string) string {
	v := strings.TrimSpace(s)
	if v == "" {
		return ""
	}
	if _, err := time.Parse(time.RFC3339, v); err != nil {
		return ""
	}
	return v
}

func atoiBounded(s string, def, min, max int) int {
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return def
	}
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
