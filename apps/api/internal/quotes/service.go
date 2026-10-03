package quotes

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/odysight/crm/internal/invoices"

	"github.com/odysight/crm/pkg/dberror"
	"github.com/odysight/crm/pkg/pagination"
	"github.com/odysight/crm/pkg/response"
)

type Service struct {
	repo *Repository
	docs Documents
}

// Documents renders quotation PDFs (the invoices service, which owns the
// document layout).
type Documents interface {
	QuotePDF(ctx context.Context, q invoices.QuoteDoc) ([]byte, error)
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// WithDocuments turns on the PDF download.
func (s *Service) WithDocuments(d Documents) *Service {
	s.docs = d
	return s
}

// document gathers what is printed on the quote.
func (s *Service) document(ctx context.Context, id int64) (Quote, invoices.QuoteDoc, error) {
	q, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Quote{}, invoices.QuoteDoc{}, mapRepoError(err)
	}
	rc, err := s.repo.Recipient(ctx, id)
	if err != nil {
		return Quote{}, invoices.QuoteDoc{}, mapRepoError(err)
	}
	address := rc.Address
	if rc.SiteAddress != "" {
		address = rc.SiteAddress
	}
	doc := invoices.QuoteDoc{
		Number: q.QuoteNumber, Date: q.CreatedAt, ValidUntil: q.ValidUntil, Status: string(q.Status),
		CustomerName: rc.Name, Address: address, TaxID: rc.TaxID, TaxBranch: rc.TaxBranch, Phone: rc.Phone,
		SiteName: rc.SiteName,
		Subtotal: q.Subtotal, TaxRate: q.TaxRate, Total: q.Total, Currency: q.Currency, Notes: q.Notes,
	}
	for _, it := range q.Items {
		doc.Lines = append(doc.Lines, invoices.QuoteLine{
			Name: it.ServiceName, Description: it.Description,
			Quantity: it.Quantity, UnitPrice: it.UnitPrice, Amount: it.LineTotal,
		})
	}
	return q, doc, nil
}

// PDF renders the quotation; it returns the quote number for the file name.
func (s *Service) PDF(ctx context.Context, id int64) (string, []byte, error) {
	if s.docs == nil {
		return "", nil, response.NewAPIError(503, "quotation PDFs are not available")
	}
	q, doc, err := s.document(ctx, id)
	if err != nil {
		return "", nil, err
	}
	pdf, err := s.docs.QuotePDF(ctx, doc)
	return q.QuoteNumber, pdf, err
}

func (s *Service) List(ctx context.Context, params pagination.Params, customerID int64) ([]Quote, int, error) {
	return s.repo.List(ctx, params, customerID)
}

func (s *Service) Get(ctx context.Context, id int64) (Quote, error) {
	q, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Quote{}, mapRepoError(err)
	}
	return q, nil
}

func (s *Service) Create(ctx context.Context, req CreateQuoteRequest) (Quote, error) {
	if err := req.Validate(); err != nil {
		return Quote{}, err
	}
	taxRate := 0.0
	if req.TaxRate != nil {
		taxRate = *req.TaxRate
	}
	sub, total := totals(req.Items, taxRate)
	q := Quote{
		CustomerID: req.CustomerID,
		SiteID:     req.SiteID,
		Status:     Status(req.Status),
		Subtotal:   sub,
		TaxRate:    taxRate,
		Total:      total,
		Currency:   strings.ToUpper(strings.TrimSpace(req.Currency)),
		Notes:      req.Notes,
	}
	if req.ValidUntil != nil {
		v, _ := parseDate(strings.TrimSpace(*req.ValidUntil))
		q.ValidUntil = &v
	}
	created, err := s.repo.Create(ctx, q, req.Items)
	if err != nil {
		return Quote{}, mapRepoError(err)
	}
	return created, nil
}

func (s *Service) Update(ctx context.Context, id int64, req UpdateQuoteRequest, approve bool) (Quote, error) {
	if err := req.Validate(); err != nil {
		return Quote{}, err
	}
	if req.IsEmpty() && !approve {
		return Quote{}, response.NewAPIError(400, "at least one field must be provided")
	}
	current, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Quote{}, mapRepoError(err)
	}
	// An accepted quote is the agreed price: it can be linked to the booking
	// or contract it became, but not re-priced.
	if current.Status == StatusAccepted && (req.Items != nil || req.TaxRate != nil || req.Currency != nil) {
		return Quote{}, response.NewAPIError(422, "an accepted quote cannot be changed — create a new quote")
	}
	if req.Notes != nil && utf8.RuneCountInString(*req.Notes) > 2000 {
		return Quote{}, response.NewAPIError(400, "notes are too long (max 2000 characters)")
	}
	var patch Patch
	patch.SiteID = req.SiteID
	if req.ClearSiteID != nil {
		patch.ClearSiteID = *req.ClearSiteID
	}
	if req.Status != nil {
		v := Status(strings.TrimSpace(*req.Status))
		// Terminal states are sticky: an accepted quote cannot go back to draft.
		if current.Status == StatusAccepted && v != StatusAccepted {
			return Quote{}, response.NewAPIError(422, "accepted quotes cannot be reopened")
		}
		patch.Status = &v
	}
	if approve {
		v := StatusAccepted
		if current.Status == StatusAccepted {
			return current, nil
		}
		patch.Status = &v
	}
	if req.ValidUntil != nil {
		v, _ := parseDate(strings.TrimSpace(*req.ValidUntil))
		patch.ValidUntil = &v
	}
	if req.ClearValidUntil != nil {
		patch.ClearValidUntil = *req.ClearValidUntil
	}
	patch.TaxRate = req.TaxRate
	if req.Currency != nil {
		v := strings.ToUpper(strings.TrimSpace(*req.Currency))
		patch.Currency = &v
	}
	if req.Notes != nil {
		v := strings.TrimSpace(*req.Notes)
		patch.Notes = &v
	}
	if req.Items != nil {
		patch.Items = req.Items
		patch.HasItems = true
	}
	patch.ConvertedBookingID = req.ConvertedBookingID
	patch.ConvertedContractID = req.ConvertedContractID
	updated, err := s.repo.Update(ctx, id, patch)
	if err != nil {
		return Quote{}, mapRepoError(err)
	}
	return updated, nil
}

func mapRepoError(err error) error {
	return dberror.Map(err, ErrNotFound, "quote not found")
}
