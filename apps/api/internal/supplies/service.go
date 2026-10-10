package supplies

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/odysight/crm/pkg/response"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func mapErr(err error) error {
	var short *InsufficientStock
	switch {
	case errors.Is(err, ErrNotFound):
		return response.NewAPIError(404, "supply not found")
	case errors.Is(err, ErrDuplicate):
		return response.NewAPIError(409, err.Error())
	case errors.As(err, &short):
		return response.NewAPIError(409, short.Error())
	}
	return err
}

var units = map[string]bool{"piece": true, "bottle": true, "litre": true, "kg": true, "pack": true, "roll": true, "box": true, "gallon": true}

type CreateRequest struct {
	Name         string  `json:"name"`
	Unit         string  `json:"unit"`
	UnitCost     float64 `json:"unitCost"`
	StockQty     float64 `json:"stockQty"`
	ReorderLevel float64 `json:"reorderLevel"`
}

func (s *Service) List(ctx context.Context, all bool) ([]Supply, error) {
	return s.repo.List(ctx, all)
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (Supply, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.Unit = strings.TrimSpace(req.Unit)
	if req.Name == "" {
		return Supply{}, response.NewAPIError(400, "name is required")
	}
	if req.Unit == "" {
		req.Unit = "piece"
	}
	if !units[req.Unit] {
		return Supply{}, response.NewAPIError(400, "unit must be piece, bottle, litre, kg, pack, roll, box or gallon")
	}
	if req.UnitCost < 0 || req.StockQty < 0 || req.ReorderLevel < 0 {
		return Supply{}, response.NewAPIError(400, "cost, stock and reorder level must be zero or more")
	}
	out, err := s.repo.Create(ctx, Supply{Name: req.Name, Unit: req.Unit, UnitCost: req.UnitCost,
		StockQty: req.StockQty, ReorderLevel: req.ReorderLevel})
	return out, mapErr(err)
}

type UpdateRequest struct {
	Name         *string  `json:"name"`
	Unit         *string  `json:"unit"`
	UnitCost     *float64 `json:"unitCost"`
	ReorderLevel *float64 `json:"reorderLevel"`
	Active       *bool    `json:"active"`
}

func (s *Service) Update(ctx context.Context, id int64, req UpdateRequest) (Supply, error) {
	if req.Name != nil {
		n := strings.TrimSpace(*req.Name)
		if n == "" {
			return Supply{}, response.NewAPIError(400, "name cannot be empty")
		}
		req.Name = &n
	}
	if req.Unit != nil && !units[strings.TrimSpace(*req.Unit)] {
		return Supply{}, response.NewAPIError(400, "invalid unit")
	}
	if (req.UnitCost != nil && *req.UnitCost < 0) || (req.ReorderLevel != nil && *req.ReorderLevel < 0) {
		return Supply{}, response.NewAPIError(400, "cost and reorder level must be zero or more")
	}
	out, err := s.repo.Update(ctx, id, Patch{Name: req.Name, Unit: req.Unit, UnitCost: req.UnitCost,
		ReorderLevel: req.ReorderLevel, Active: req.Active})
	return out, mapErr(err)
}

// MovementRequest records stock in or out. Quantity is entered positive for
// purchases and usage; adjustments take a signed correction.
type MovementRequest struct {
	Kind      string  `json:"kind"`
	Quantity  float64 `json:"quantity"`
	UnitCost  float64 `json:"unitCost"`
	SiteID    *int64  `json:"siteId"`
	BookingID *int64  `json:"bookingId"`
	Note      string  `json:"note"`
}

func (s *Service) Record(ctx context.Context, supplyID, userID int64, req MovementRequest) (Movement, Supply, error) {
	m := Movement{SupplyID: supplyID, Kind: strings.TrimSpace(req.Kind), UnitCost: req.UnitCost,
		SiteID: req.SiteID, BookingID: req.BookingID, Note: strings.TrimSpace(req.Note)}
	if req.UnitCost < 0 {
		return Movement{}, Supply{}, response.NewAPIError(400, "unitCost must be zero or more")
	}
	switch m.Kind {
	case "purchase", "usage":
		if req.Quantity <= 0 {
			return Movement{}, Supply{}, response.NewAPIError(400, "quantity must be more than zero")
		}
		m.Quantity = req.Quantity
		if m.Kind == "usage" {
			m.Quantity = -req.Quantity
		}
	case "adjustment":
		if req.Quantity == 0 {
			return Movement{}, Supply{}, response.NewAPIError(400, "adjustment quantity cannot be zero")
		}
		if m.Note == "" {
			return Movement{}, Supply{}, response.NewAPIError(400, "say why stock is being adjusted (note)")
		}
		m.Quantity = req.Quantity
	default:
		return Movement{}, Supply{}, response.NewAPIError(400, "kind must be purchase, usage or adjustment")
	}
	if m.Kind == "usage" {
		if m.BookingID != nil {
			site, err := s.repo.BookingSite(ctx, *m.BookingID)
			if err != nil {
				if errors.Is(err, ErrNotFound) {
					return Movement{}, Supply{}, response.NewAPIError(400, "booking not found")
				}
				return Movement{}, Supply{}, err
			}
			if m.SiteID == nil {
				m.SiteID = site
			}
		}
		if m.SiteID == nil {
			return Movement{}, Supply{}, response.NewAPIError(400, "usage needs a site or a job with a site, so it counts toward that site's cost")
		}
	}
	mv, sup, err := s.repo.Record(ctx, m, userID)
	if err != nil && strings.Contains(err.Error(), "supply_movements_site_id_fkey") {
		return Movement{}, Supply{}, response.NewAPIError(400, "site not found")
	}
	return mv, sup, mapErr(err)
}

// Range parses YYYY-MM-DD dates into [from, to+1day), defaulting to the
// current month.
func Range(fromRaw, toRaw string, now time.Time) (time.Time, time.Time, error) {
	now = now.Local()
	from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
	to := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	var err error
	if v := strings.TrimSpace(fromRaw); v != "" {
		if from, err = time.ParseInLocation("2006-01-02", v, time.Local); err != nil {
			return time.Time{}, time.Time{}, response.NewAPIError(400, "from must be YYYY-MM-DD")
		}
	}
	if v := strings.TrimSpace(toRaw); v != "" {
		if to, err = time.ParseInLocation("2006-01-02", v, time.Local); err != nil {
			return time.Time{}, time.Time{}, response.NewAPIError(400, "to must be YYYY-MM-DD")
		}
	}
	if to.Before(from) {
		return time.Time{}, time.Time{}, response.NewAPIError(400, "to must not be before from")
	}
	return from, to.AddDate(0, 0, 1), nil
}

func (s *Service) Movements(ctx context.Context, f MovementFilters) ([]Movement, error) {
	return s.repo.Movements(ctx, f)
}

func (s *Service) SiteCosts(ctx context.Context, from, to time.Time) ([]SiteCost, error) {
	return s.repo.SiteCosts(ctx, from, to)
}
