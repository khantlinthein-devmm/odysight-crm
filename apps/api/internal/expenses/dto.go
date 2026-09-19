package expenses

import (
	"strings"
	"time"

	"github.com/odysight/crm/pkg/response"
)

// dateFormat is the YYYY-MM-DD layout used for spend dates.
const dateFormat = "2006-01-02"

// maxAmount keeps values inside the expenses.amount NUMERIC(12,2) column, so an
// oversized figure is a 400 rather than a Postgres overflow.
const maxAmount = 9_999_999_999.99

const (
	maxCategoryLen = 80
	maxNoteLen     = 500
)

type ExpenseDTO struct {
	ID        int64     `json:"id"`
	SpentOn   string    `json:"spentOn"`
	Category  string    `json:"category"`
	Amount    float64   `json:"amount"`
	Note      string    `json:"note"`
	CreatedBy int64     `json:"createdBy"`
	CreatedAt time.Time `json:"createdAt"`
}

func toDTO(e Expense) ExpenseDTO {
	return ExpenseDTO{
		ID:        e.ID,
		SpentOn:   e.SpentOn,
		Category:  e.Category,
		Amount:    e.Amount,
		Note:      e.Note,
		CreatedBy: e.CreatedBy,
		CreatedAt: e.CreatedAt,
	}
}

type CreateExpenseRequest struct {
	SpentOn  string  `json:"spentOn"`
	Category string  `json:"category"`
	Amount   float64 `json:"amount"`
	Note     string  `json:"note"`
}

func (r *CreateExpenseRequest) Validate() error {
	r.SpentOn = strings.TrimSpace(r.SpentOn)
	r.Category = strings.TrimSpace(r.Category)
	r.Note = strings.TrimSpace(r.Note)
	if err := validateDate(r.SpentOn, true); err != nil {
		return err
	}
	if err := validateCategory(r.Category, true); err != nil {
		return err
	}
	if err := validateAmount(r.Amount); err != nil {
		return err
	}
	return validateNote(r.Note)
}

type UpdateExpenseRequest struct {
	SpentOn  *string  `json:"spentOn"`
	Category *string  `json:"category"`
	Amount   *float64 `json:"amount"`
	Note     *string  `json:"note"`
}

func (r *UpdateExpenseRequest) Validate() error {
	if r.SpentOn != nil {
		v := strings.TrimSpace(*r.SpentOn)
		r.SpentOn = &v
		if err := validateDate(v, true); err != nil {
			return err
		}
	}
	if r.Category != nil {
		v := strings.TrimSpace(*r.Category)
		r.Category = &v
		if err := validateCategory(v, true); err != nil {
			return err
		}
	}
	if r.Amount != nil {
		if err := validateAmount(*r.Amount); err != nil {
			return err
		}
	}
	if r.Note != nil {
		v := strings.TrimSpace(*r.Note)
		r.Note = &v
		if err := validateNote(v); err != nil {
			return err
		}
	}
	return nil
}

func (r *UpdateExpenseRequest) IsEmpty() bool {
	return r.SpentOn == nil && r.Category == nil && r.Amount == nil && r.Note == nil
}

func validateDate(v string, required bool) error {
	if v == "" {
		if required {
			return response.NewAPIError(400, "spentOn is required")
		}
		return nil
	}
	if _, err := time.Parse(dateFormat, v); err != nil {
		return response.NewAPIError(400, "spentOn must be a YYYY-MM-DD date")
	}
	return nil
}

func validateCategory(v string, required bool) error {
	if v == "" {
		if required {
			return response.NewAPIError(400, "category is required")
		}
		return nil
	}
	if len(v) > maxCategoryLen {
		return response.NewAPIError(400, "category is too long")
	}
	return nil
}

func validateAmount(v float64) error {
	if v <= 0 {
		return response.NewAPIError(400, "amount must be greater than 0")
	}
	if v > maxAmount {
		return response.NewAPIError(400, "amount is too large")
	}
	return nil
}

func validateNote(v string) error {
	if len(v) > maxNoteLen {
		return response.NewAPIError(400, "note is too long")
	}
	return nil
}
