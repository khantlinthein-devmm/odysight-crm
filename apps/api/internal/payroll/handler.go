package payroll

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/odysight/crm/internal/auth"
	"github.com/odysight/crm/pkg/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func Routes(h *Handler, az *auth.Authorizer) chi.Router {
	r := chi.NewRouter()
	r.With(az.Require(auth.PermPayrollRead)).Get("/", h.Report)
	r.With(az.Require(auth.PermPayrollRead)).Get("/export.csv", h.ExportCSV)
	r.With(az.Require(auth.PermPayrollRead)).Get("/rates", h.Rates)
	r.With(az.Require(auth.PermPayrollManage)).Patch("/rates/{cleanerId}", h.SetRate)
	return r
}

type LineDTO struct {
	CleanerID    int64   `json:"cleanerId"`
	Name         string  `json:"name"`
	PayType      PayType `json:"payType"`
	Rate         float64 `json:"rate"`
	HasRate      bool    `json:"hasRate"`
	DaysWorked   int     `json:"daysWorked"`
	OpenDays     int     `json:"openDays"`
	Hours        float64 `json:"hours"`
	RegularHours float64 `json:"regularHours"`
	OTHours      float64 `json:"otHours"`
	Jobs         int     `json:"jobs"`
	BasePay      float64 `json:"basePay"`
	OTPay        float64 `json:"otPay"`
	Total        float64 `json:"total"`
}

type ReportDTO struct {
	From  string    `json:"from"`
	To    string    `json:"to"`
	Lines []LineDTO `json:"lines"`
	Total float64   `json:"total"`
}

type RateDTO struct {
	CleanerID     int64     `json:"cleanerId"`
	PayType       PayType   `json:"payType"`
	Rate          float64   `json:"rate"`
	OTMultiplier  float64   `json:"otMultiplier"`
	StandardHours float64   `json:"standardHours"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

func rateDTO(r Rate) RateDTO {
	return RateDTO{r.CleanerID, r.PayType, r.Rate, r.OTMultiplier, r.StandardHours, r.UpdatedAt}
}

func (h *Handler) load(w http.ResponseWriter, r *http.Request) (Report, bool) {
	from, to, err := Period(r.URL.Query().Get("from"), r.URL.Query().Get("to"), time.Now())
	if err != nil {
		response.HandleError(w, r, err)
		return Report{}, false
	}
	rep, err := h.service.Report(r.Context(), from, to)
	if err != nil {
		response.HandleError(w, r, err)
		return Report{}, false
	}
	return rep, true
}

// Report handles GET /api/v1/payroll?from=YYYY-MM-DD&to=YYYY-MM-DD.
func (h *Handler) Report(w http.ResponseWriter, r *http.Request) {
	rep, ok := h.load(w, r)
	if !ok {
		return
	}
	out := ReportDTO{From: rep.From, To: rep.To, Total: rep.Total, Lines: make([]LineDTO, 0, len(rep.Lines))}
	for _, l := range rep.Lines {
		out.Lines = append(out.Lines, LineDTO(l))
	}
	response.JSON(w, http.StatusOK, out)
}

// ExportCSV handles GET /api/v1/payroll/export.csv — the same report as a
// spreadsheet (UTF-8 with BOM so Excel shows Thai names correctly).
func (h *Handler) ExportCSV(w http.ResponseWriter, r *http.Request) {
	rep, ok := h.load(w, r)
	if !ok {
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="payroll_%s_%s.csv"`, rep.From, rep.To))
	_, _ = w.Write([]byte("\xef\xbb\xbf"))
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"Cleaner", "Pay type", "Rate", "Days worked", "Open days (no check-out)",
		"Hours", "Regular hours", "OT hours", "Completed jobs", "Base pay", "OT pay", "Total"})
	money := func(v float64) string { return strconv.FormatFloat(v, 'f', 2, 64) }
	for _, l := range rep.Lines {
		payType := string(l.PayType)
		if !l.HasRate {
			payType = "(no rate set)"
		}
		_ = cw.Write([]string{l.Name, payType, money(l.Rate), strconv.Itoa(l.DaysWorked), strconv.Itoa(l.OpenDays),
			money(l.Hours), money(l.RegularHours), money(l.OTHours), strconv.Itoa(l.Jobs),
			money(l.BasePay), money(l.OTPay), money(l.Total)})
	}
	_ = cw.Write([]string{"TOTAL", "", "", "", "", "", "", "", "", "", "", money(rep.Total)})
	cw.Flush()
}

// Rates handles GET /api/v1/payroll/rates.
func (h *Handler) Rates(w http.ResponseWriter, r *http.Request) {
	rates, err := h.service.Rates(r.Context())
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	out := make([]RateDTO, 0, len(rates))
	for _, rt := range rates {
		out = append(out, rateDTO(rt))
	}
	response.JSON(w, http.StatusOK, out)
}

// SetRate handles PATCH /api/v1/payroll/rates/{cleanerId} (create or replace).
func (h *Handler) SetRate(w http.ResponseWriter, r *http.Request) {
	cleanerID, err := strconv.ParseInt(chi.URLParam(r, "cleanerId"), 10, 64)
	if err != nil || cleanerID < 1 {
		response.Error(w, http.StatusBadRequest, "invalid cleaner id")
		return
	}
	var req RateRequest
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	id, _ := auth.IdentityFromContext(r.Context())
	rt, err := h.service.SetRate(r.Context(), cleanerID, id.UserID, req)
	if err != nil {
		response.HandleError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, rateDTO(rt))
}
