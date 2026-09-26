package customers

import (
	"strings"
	"time"

	"github.com/odysight/crm/pkg/response"
	"github.com/odysight/crm/pkg/validate"
)

type CustomerDTO struct {
	ID              int64     `json:"id"`
	FirstName       string    `json:"firstName"`
	LastName        string    `json:"lastName"`
	Email           string    `json:"email"`
	Phone           string    `json:"phone"`
	Address         string    `json:"address"`
	PropertyType    string    `json:"propertyType"`
	Area            string    `json:"area"`
	Status          string    `json:"status"`
	LeadID          *int64    `json:"leadId"`
	PortalEnabled   bool      `json:"portalEnabled"`
	TaxID           string    `json:"taxId"`
	TaxBranch       string    `json:"taxBranch"`
	WithholdingRate float64   `json:"withholdingRate"`
	CreatedAt       time.Time `json:"createdAt"`
}

func toDTO(c Customer) CustomerDTO {
	return CustomerDTO{
		ID:              c.ID,
		FirstName:       c.FirstName,
		LastName:        c.LastName,
		Email:           c.Email,
		Phone:           c.Phone,
		Address:         c.Address,
		PropertyType:    string(c.PropertyType),
		Area:            c.Area,
		Status:          string(c.Status),
		LeadID:          c.LeadID,
		PortalEnabled:   c.PortalEnabled,
		TaxID:           c.TaxID,
		TaxBranch:       c.TaxBranch,
		WithholdingRate: c.WithholdingRate,
		CreatedAt:       c.CreatedAt,
	}
}

// ToDTO exports toDTO for other packages (e.g. lead conversion responses).
func ToDTO(c Customer) CustomerDTO {
	return toDTO(c)
}

type CreateCustomerRequest struct {
	FirstName       string  `json:"firstName"`
	LastName        string  `json:"lastName"`
	Email           string  `json:"email"`
	Phone           string  `json:"phone"`
	Address         string  `json:"address"`
	PropertyType    string  `json:"propertyType"`
	Area            string  `json:"area"`
	Status          string  `json:"status"`
	LeadID          *int64  `json:"leadId"`
	TaxID           string  `json:"taxId"`
	TaxBranch       string  `json:"taxBranch"`
	WithholdingRate float64 `json:"withholdingRate"`
}

func (r *CreateCustomerRequest) Validate() error {
	r.FirstName = strings.TrimSpace(r.FirstName)
	r.LastName = strings.TrimSpace(r.LastName)
	r.Email = strings.ToLower(strings.TrimSpace(r.Email))
	r.Phone = strings.TrimSpace(r.Phone)
	r.Address = strings.TrimSpace(r.Address)
	r.PropertyType = strings.TrimSpace(r.PropertyType)
	r.Area = strings.TrimSpace(r.Area)

	if r.FirstName == "" {
		return response.NewAPIError(400, "firstName is required")
	}
	// Last name and email are optional: walk-in customers often have neither.
	if r.Email != "" && !validate.Email(r.Email) {
		return response.NewAPIError(400, "email must be a valid address")
	}
	if !validate.Phone(r.Phone) {
		return response.NewAPIError(400, "a valid phone is required")
	}
	if r.Address == "" {
		return response.NewAPIError(400, "address is required")
	}
	if !PropertyType(r.PropertyType).Valid() {
		return response.NewAPIError(400, "invalid propertyType")
	}
	if r.Area == "" {
		return response.NewAPIError(400, "area is required")
	}
	if !Status(r.Status).Valid() {
		return response.NewAPIError(400, "invalid status")
	}
	var err error
	if r.TaxID, r.TaxBranch, err = normalizeTax(r.TaxID, r.TaxBranch); err != nil {
		return err
	}
	return validateWithholding(r.WithholdingRate)
}

// normalizeTax strips separators from a Thai TIN and branch code. A TIN must
// be 13 digits when given; a branch defaults to head office ("00000").
func normalizeTax(taxID, branch string) (string, string, error) {
	taxID = digitsOnly(taxID)
	branch = digitsOnly(branch)
	if taxID == "" {
		return "", branch, nil
	}
	if len(taxID) != 13 {
		return "", "", response.NewAPIError(400, "taxId must be a 13-digit Thai tax ID")
	}
	if branch == "" {
		branch = "00000"
	}
	if len(branch) > 5 {
		return "", "", response.NewAPIError(400, "taxBranch must be up to 5 digits")
	}
	return taxID, strings.Repeat("0", 5-len(branch)) + branch, nil
}

func digitsOnly(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func validateWithholding(rate float64) error {
	if rate < 0 || rate > 15 {
		return response.NewAPIError(400, "withholdingRate must be between 0 and 15 percent")
	}
	return nil
}

type UpdateCustomerRequest struct {
	FirstName       *string  `json:"firstName"`
	LastName        *string  `json:"lastName"`
	Email           *string  `json:"email"`
	Phone           *string  `json:"phone"`
	Address         *string  `json:"address"`
	PropertyType    *string  `json:"propertyType"`
	Area            *string  `json:"area"`
	Status          *string  `json:"status"`
	TaxID           *string  `json:"taxId"`
	TaxBranch       *string  `json:"taxBranch"`
	WithholdingRate *float64 `json:"withholdingRate"`
}

const errNoFields = "at least one field must be provided"

func (r *UpdateCustomerRequest) Validate() error {
	if r.FirstName != nil && strings.TrimSpace(*r.FirstName) == "" {
		return response.NewAPIError(400, "firstName cannot be empty")
	}
	// Last name and email may be cleared: both are optional.
	if r.LastName != nil {
		lastName := strings.TrimSpace(*r.LastName)
		r.LastName = &lastName
	}
	if r.Email != nil {
		email := strings.ToLower(strings.TrimSpace(*r.Email))
		if email != "" && !validate.Email(email) {
			return response.NewAPIError(400, "email must be a valid address")
		}
		r.Email = &email
	}
	if r.Phone != nil && !validate.Phone(*r.Phone) {
		return response.NewAPIError(400, "a valid phone is required")
	}
	if r.Address != nil && strings.TrimSpace(*r.Address) == "" {
		return response.NewAPIError(400, "address cannot be empty")
	}
	if r.PropertyType != nil && !PropertyType(*r.PropertyType).Valid() {
		return response.NewAPIError(400, "invalid propertyType")
	}
	if r.Area != nil && strings.TrimSpace(*r.Area) == "" {
		return response.NewAPIError(400, "area cannot be empty")
	}
	if r.Status != nil && !Status(*r.Status).Valid() {
		return response.NewAPIError(400, "invalid status")
	}
	if r.TaxID != nil || r.TaxBranch != nil {
		taxID, branch := "", ""
		if r.TaxID != nil {
			taxID = *r.TaxID
		}
		if r.TaxBranch != nil {
			branch = *r.TaxBranch
		}
		t, b, err := normalizeTax(taxID, branch)
		if err != nil {
			return err
		}
		if r.TaxID != nil {
			r.TaxID = &t
		}
		if r.TaxBranch != nil || r.TaxID != nil {
			r.TaxBranch = &b
		}
	}
	if r.WithholdingRate != nil {
		return validateWithholding(*r.WithholdingRate)
	}
	return nil
}

func (r *UpdateCustomerRequest) IsEmpty() bool {
	return r.FirstName == nil && r.LastName == nil && r.Email == nil &&
		r.Phone == nil && r.Address == nil && r.PropertyType == nil &&
		r.Area == nil && r.Status == nil &&
		r.TaxID == nil && r.TaxBranch == nil && r.WithholdingRate == nil
}
