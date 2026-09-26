package customers

import "time"

type Status string

const (
	StatusActive   Status = "active"
	StatusInactive Status = "inactive"
	StatusBlocked  Status = "blocked"
)

func (s Status) Valid() bool {
	switch s {
	case StatusActive, StatusInactive, StatusBlocked:
		return true
	}
	return false
}

type PropertyType string

const (
	PropHouse     PropertyType = "house"
	PropCondo     PropertyType = "condo"
	PropOffice    PropertyType = "office"
	PropApartment PropertyType = "apartment"
	PropOther     PropertyType = "other"
)

func (p PropertyType) Valid() bool {
	switch p {
	case PropHouse, PropCondo, PropOffice, PropApartment, PropOther:
		return true
	}
	return false
}

type Customer struct {
	ID            int64
	FirstName     string
	LastName      string
	Email         string
	Phone         string
	Address       string
	PropertyType  PropertyType
	Area          string
	Status        Status
	LeadID        *int64
	PasswordHash  string
	PortalEnabled bool
	// Thai tax identity: 13-digit TIN, branch ("00000" = head office) and the
	// withholding-tax percent this customer deducts when paying.
	TaxID           string
	TaxBranch       string
	WithholdingRate float64
	CreatedAt       time.Time
}
