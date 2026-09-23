package contracts

import (
	"strings"

	"github.com/odysight/crm/pkg/response"
)

type ContractDTO struct {
	ID               int64   `json:"id"`
	ContractNumber   string  `json:"contractNumber"`
	CustomerID       int64   `json:"customerId"`
	Title            string  `json:"title"`
	Status           string  `json:"status"`
	StartDate        string  `json:"startDate"`
	EndDate          string  `json:"endDate"`
	RenewalDate      *string `json:"renewalDate"`
	ContractValue    float64 `json:"contractValue"`
	BillingFrequency string  `json:"billingFrequency"`
	SLATerms         string  `json:"slaTerms"`
	Notes            string  `json:"notes"`
	SiteIDs          []int64 `json:"siteIds"`
	CreatedAt        string  `json:"createdAt"`
	UpdatedAt        string  `json:"updatedAt"`
}

func toDTO(c Contract) ContractDTO {
	var renewal *string
	if c.RenewalDate != nil {
		v := c.RenewalDate.Format("2006-01-02")
		renewal = &v
	}
	siteIDs := c.SiteIDs
	if siteIDs == nil {
		siteIDs = []int64{}
	}
	return ContractDTO{
		ID:               c.ID,
		ContractNumber:   c.ContractNumber,
		CustomerID:       c.CustomerID,
		Title:            c.Title,
		Status:           string(c.Status),
		StartDate:        c.StartDate.Format("2006-01-02"),
		EndDate:          c.EndDate.Format("2006-01-02"),
		RenewalDate:      renewal,
		ContractValue:    c.ContractValue,
		BillingFrequency: string(c.BillingFrequency),
		SLATerms:         c.SLATerms,
		Notes:            c.Notes,
		SiteIDs:          siteIDs,
		CreatedAt:        c.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:        c.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

type CreateContractRequest struct {
	CustomerID       int64   `json:"customerId"`
	Title            string  `json:"title"`
	Status           string  `json:"status"`
	StartDate        string  `json:"startDate"`
	EndDate          string  `json:"endDate"`
	RenewalDate      *string `json:"renewalDate"`
	ContractValue    float64 `json:"contractValue"`
	BillingFrequency string  `json:"billingFrequency"`
	SLATerms         string  `json:"slaTerms"`
	Notes            string  `json:"notes"`
	SiteIDs          []int64 `json:"siteIds"`
}

func (r *CreateContractRequest) Validate() error {
	r.Title = strings.TrimSpace(r.Title)
	r.Status = strings.TrimSpace(r.Status)
	r.StartDate = strings.TrimSpace(r.StartDate)
	r.EndDate = strings.TrimSpace(r.EndDate)
	r.BillingFrequency = strings.TrimSpace(r.BillingFrequency)
	r.SLATerms = strings.TrimSpace(r.SLATerms)
	r.Notes = strings.TrimSpace(r.Notes)
	if r.Status == "" {
		r.Status = string(StatusDraft)
	}
	if r.BillingFrequency == "" {
		r.BillingFrequency = string(BillingMonthly)
	}

	if r.CustomerID < 1 {
		return response.NewAPIError(400, "customerId is required")
	}
	if !Status(r.Status).Valid() {
		return response.NewAPIError(400, "invalid status")
	}
	if !BillingFrequency(r.BillingFrequency).Valid() {
		return response.NewAPIError(400, "invalid billingFrequency")
	}
	start, err := parseDate(r.StartDate)
	if err != nil {
		return response.NewAPIError(400, "startDate must be YYYY-MM-DD")
	}
	end, err := parseDate(r.EndDate)
	if err != nil {
		return response.NewAPIError(400, "endDate must be YYYY-MM-DD")
	}
	if end.Before(start) {
		return response.NewAPIError(400, "endDate must be on or after startDate")
	}
	if r.RenewalDate != nil {
		if _, err := parseDate(strings.TrimSpace(*r.RenewalDate)); err != nil {
			return response.NewAPIError(400, "renewalDate must be YYYY-MM-DD")
		}
	}
	if r.ContractValue < 0 {
		return response.NewAPIError(400, "contractValue must be zero or greater")
	}
	return nil
}

type UpdateContractRequest struct {
	Title            *string  `json:"title"`
	Status           *string  `json:"status"`
	StartDate        *string  `json:"startDate"`
	EndDate          *string  `json:"endDate"`
	RenewalDate      *string  `json:"renewalDate"`
	ContractValue    *float64 `json:"contractValue"`
	BillingFrequency *string  `json:"billingFrequency"`
	SLATerms         *string  `json:"slaTerms"`
	Notes            *string  `json:"notes"`
	SiteIDs          []int64  `json:"siteIds"`
	ClearSiteIDs     *bool    `json:"clearSiteIds"`
}

func (r *UpdateContractRequest) Validate() error {
	if r.Status != nil && !Status(strings.TrimSpace(*r.Status)).Valid() {
		return response.NewAPIError(400, "invalid status")
	}
	if r.BillingFrequency != nil && !BillingFrequency(strings.TrimSpace(*r.BillingFrequency)).Valid() {
		return response.NewAPIError(400, "invalid billingFrequency")
	}
	if r.StartDate != nil {
		if _, err := parseDate(strings.TrimSpace(*r.StartDate)); err != nil {
			return response.NewAPIError(400, "startDate must be YYYY-MM-DD")
		}
	}
	if r.EndDate != nil {
		if _, err := parseDate(strings.TrimSpace(*r.EndDate)); err != nil {
			return response.NewAPIError(400, "endDate must be YYYY-MM-DD")
		}
	}
	if r.RenewalDate != nil {
		if _, err := parseDate(strings.TrimSpace(*r.RenewalDate)); err != nil {
			return response.NewAPIError(400, "renewalDate must be YYYY-MM-DD")
		}
	}
	if r.ContractValue != nil && *r.ContractValue < 0 {
		return response.NewAPIError(400, "contractValue must be zero or greater")
	}
	return nil
}

func (r *UpdateContractRequest) IsEmpty() bool {
	return r.Title == nil && r.Status == nil && r.StartDate == nil &&
		r.EndDate == nil && r.RenewalDate == nil && r.ContractValue == nil &&
		r.BillingFrequency == nil && r.SLATerms == nil && r.Notes == nil &&
		r.SiteIDs == nil && r.ClearSiteIDs == nil
}
