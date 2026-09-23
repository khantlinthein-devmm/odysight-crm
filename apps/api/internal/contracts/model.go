package contracts

import "time"

type Status string

const (
	StatusDraft     Status = "draft"
	StatusActive    Status = "active"
	StatusExpiring  Status = "expiring"
	StatusExpired   Status = "expired"
	StatusCancelled Status = "cancelled"
	StatusRenewed   Status = "renewed"
)

func (s Status) Valid() bool {
	switch s {
	case StatusDraft, StatusActive, StatusExpiring, StatusExpired, StatusCancelled, StatusRenewed:
		return true
	}
	return false
}

type BillingFrequency string

const (
	BillingMonthly   BillingFrequency = "monthly"
	BillingQuarterly BillingFrequency = "quarterly"
	BillingAnnual    BillingFrequency = "annual"
	BillingCustom    BillingFrequency = "custom"
)

func (b BillingFrequency) Valid() bool {
	switch b {
	case BillingMonthly, BillingQuarterly, BillingAnnual, BillingCustom:
		return true
	}
	return false
}

type Contract struct {
	ID               int64
	ContractNumber   string
	CustomerID       int64
	Title            string
	Status           Status
	StartDate        time.Time
	EndDate          time.Time
	RenewalDate      *time.Time
	ContractValue    float64
	BillingFrequency BillingFrequency
	SLATerms         string
	Notes            string
	SiteIDs          []int64
	CreatedAt        time.Time
	UpdatedAt        time.Time
}
