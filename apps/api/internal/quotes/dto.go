package quotes

import (
	"math"
	"strings"

	"github.com/odysight/crm/pkg/response"
)

type QuoteItemDTO struct {
	ID          int64   `json:"id"`
	ServiceName string  `json:"serviceName"`
	Description string  `json:"description"`
	Quantity    float64 `json:"quantity"`
	UnitPrice   float64 `json:"unitPrice"`
	LineTotal   float64 `json:"lineTotal"`
	SortOrder   int     `json:"sortOrder"`
}

type QuoteDTO struct {
	ID                  int64          `json:"id"`
	QuoteNumber         string         `json:"quoteNumber"`
	CustomerID          int64          `json:"customerId"`
	SiteID              *int64         `json:"siteId"`
	Status              string         `json:"status"`
	ValidUntil          *string        `json:"validUntil"`
	Subtotal            float64        `json:"subtotal"`
	TaxRate             float64        `json:"taxRate"`
	Total               float64        `json:"total"`
	Currency            string         `json:"currency"`
	Notes               string         `json:"notes"`
	Version             int            `json:"version"`
	AcceptedAt          *string        `json:"acceptedAt"`
	RejectedAt          *string        `json:"rejectedAt"`
	ConvertedBookingID  *int64         `json:"convertedBookingId"`
	ConvertedContractID *int64         `json:"convertedContractId"`
	Items               []QuoteItemDTO `json:"items"`
	CreatedAt           string         `json:"createdAt"`
	UpdatedAt           string         `json:"updatedAt"`
}

func toDTO(q Quote) QuoteDTO {
	var validUntil *string
	if q.ValidUntil != nil {
		v := q.ValidUntil.Format("2006-01-02")
		validUntil = &v
	}
	var accepted *string
	if q.AcceptedAt != nil {
		v := q.AcceptedAt.Format("2006-01-02T15:04:05Z07:00")
		accepted = &v
	}
	var rejected *string
	if q.RejectedAt != nil {
		v := q.RejectedAt.Format("2006-01-02T15:04:05Z07:00")
		rejected = &v
	}
	items := make([]QuoteItemDTO, 0, len(q.Items))
	for _, it := range q.Items {
		items = append(items, QuoteItemDTO{
			ID: it.ID, ServiceName: it.ServiceName, Description: it.Description,
			Quantity: it.Quantity, UnitPrice: it.UnitPrice, LineTotal: it.LineTotal, SortOrder: it.SortOrder,
		})
	}
	return QuoteDTO{
		ID: q.ID, QuoteNumber: q.QuoteNumber, CustomerID: q.CustomerID, SiteID: q.SiteID,
		Status: string(q.Status), ValidUntil: validUntil, Subtotal: q.Subtotal,
		TaxRate: q.TaxRate, Total: q.Total, Currency: q.Currency, Notes: q.Notes,
		Version: q.Version, AcceptedAt: accepted, RejectedAt: rejected,
		ConvertedBookingID: q.ConvertedBookingID, ConvertedContractID: q.ConvertedContractID,
		Items: items,
		CreatedAt: q.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: q.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

type QuoteItemInput struct {
	ServiceName string  `json:"serviceName"`
	Description string  `json:"description"`
	Quantity    float64 `json:"quantity"`
	UnitPrice   float64 `json:"unitPrice"`
}

type CreateQuoteRequest struct {
	CustomerID int64            `json:"customerId"`
	SiteID     *int64           `json:"siteId"`
	Status     string           `json:"status"`
	ValidUntil *string          `json:"validUntil"`
	TaxRate    *float64         `json:"taxRate"`
	Currency   string           `json:"currency"`
	Notes      string           `json:"notes"`
	Items      []QuoteItemInput `json:"items"`
}

func (r *CreateQuoteRequest) Validate() error {
	r.Status = strings.TrimSpace(r.Status)
	r.Currency = strings.TrimSpace(r.Currency)
	r.Notes = strings.TrimSpace(r.Notes)
	if r.Status == "" {
		r.Status = string(StatusDraft)
	}
	if r.Currency == "" {
		r.Currency = "THB"
	}
	if r.CustomerID < 1 {
		return response.NewAPIError(400, "customerId is required")
	}
	if !Status(r.Status).Valid() {
		return response.NewAPIError(400, "invalid status")
	}
	if r.ValidUntil != nil {
		if _, err := parseDate(strings.TrimSpace(*r.ValidUntil)); err != nil {
			return response.NewAPIError(400, "validUntil must be YYYY-MM-DD")
		}
	}
	if r.TaxRate != nil && (*r.TaxRate < 0 || *r.TaxRate > 100) {
		return response.NewAPIError(400, "taxRate must be between 0 and 100")
	}
	if len(r.Currency) != 3 {
		return response.NewAPIError(400, "currency must be a 3-letter code")
	}
	if len(r.Items) == 0 {
		return response.NewAPIError(400, "at least one item is required")
	}
	for i := range r.Items {
		r.Items[i].ServiceName = strings.TrimSpace(r.Items[i].ServiceName)
		r.Items[i].Description = strings.TrimSpace(r.Items[i].Description)
		if r.Items[i].ServiceName == "" {
			return response.NewAPIError(400, "items[].serviceName is required")
		}
		if r.Items[i].Quantity <= 0 {
			return response.NewAPIError(400, "items[].quantity must be greater than zero")
		}
		if r.Items[i].UnitPrice < 0 {
			return response.NewAPIError(400, "items[].unitPrice must be zero or greater")
		}
	}
	return nil
}

type UpdateQuoteRequest struct {
	SiteID           *int64           `json:"siteId"`
	ClearSiteID      *bool            `json:"clearSiteId"`
	Status           *string          `json:"status"`
	ValidUntil       *string          `json:"validUntil"`
	ClearValidUntil  *bool            `json:"clearValidUntil"`
	TaxRate          *float64         `json:"taxRate"`
	Currency         *string          `json:"currency"`
	Notes            *string          `json:"notes"`
	Items            []QuoteItemInput `json:"items"`
	ConvertedBookingID  *int64        `json:"convertedBookingId"`
	ConvertedContractID *int64        `json:"convertedContractId"`
}

func (r *UpdateQuoteRequest) Validate() error {
	if r.Status != nil && !Status(strings.TrimSpace(*r.Status)).Valid() {
		return response.NewAPIError(400, "invalid status")
	}
	if r.ValidUntil != nil {
		if _, err := parseDate(strings.TrimSpace(*r.ValidUntil)); err != nil {
			return response.NewAPIError(400, "validUntil must be YYYY-MM-DD")
		}
	}
	if r.TaxRate != nil && (*r.TaxRate < 0 || *r.TaxRate > 100) {
		return response.NewAPIError(400, "taxRate must be between 0 and 100")
	}
	if r.Currency != nil && len(strings.TrimSpace(*r.Currency)) != 3 {
		return response.NewAPIError(400, "currency must be a 3-letter code")
	}
	if r.Items != nil {
		if len(r.Items) == 0 {
			return response.NewAPIError(400, "items cannot be empty when provided")
		}
		for i := range r.Items {
			if strings.TrimSpace(r.Items[i].ServiceName) == "" {
				return response.NewAPIError(400, "items[].serviceName is required")
			}
			if r.Items[i].Quantity <= 0 {
				return response.NewAPIError(400, "items[].quantity must be greater than zero")
			}
			if r.Items[i].UnitPrice < 0 {
				return response.NewAPIError(400, "items[].unitPrice must be zero or greater")
			}
		}
	}
	return nil
}

func (r *UpdateQuoteRequest) IsEmpty() bool {
	return r.SiteID == nil && r.ClearSiteID == nil && r.Status == nil &&
		r.ValidUntil == nil && r.ClearValidUntil == nil && r.TaxRate == nil &&
		r.Currency == nil && r.Notes == nil && r.Items == nil &&
		r.ConvertedBookingID == nil && r.ConvertedContractID == nil
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }
