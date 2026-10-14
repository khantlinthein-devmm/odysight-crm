package checklists

import (
	"context"
	"errors"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/odysight/crm/internal/auth"
	"github.com/odysight/crm/pkg/dberror"
	"github.com/odysight/crm/pkg/pagination"
	"github.com/odysight/crm/pkg/response"
)

// OfficeNotifier tells the office about a submitted checklist.
type OfficeNotifier interface {
	NotifyOffice(ctx context.Context, eventType, subject, bodyHTML string)
}

type Service struct {
	repo     *Repository
	notifier OfficeNotifier
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) WithNotifier(n OfficeNotifier) *Service {
	s.notifier = n
	return s
}

// Caller is who is acting. Cleaners only reach checklists of jobs they work,
// cannot pick templates and cannot review.
type Caller struct {
	UserID  int64
	Cleaner bool
}

func CallerFrom(ctx context.Context) Caller {
	id, _ := auth.IdentityFromContext(ctx)
	return Caller{UserID: id.UserID, Cleaner: id.Role == auth.RoleCleaner}
}

var (
	errNotYourJob   = response.NewAPIError(403, "you can only open checklists of jobs assigned to you")
	errStaffOnly    = response.NewAPIError(403, "only office staff can do this")
	errLocked       = response.NewAPIError(409, "this checklist was submitted; the office must send it back before it can change")
	errNotSubmitted = response.NewAPIError(409, "this checklist has not been submitted yet")
)

func (s *Service) checkBooking(ctx context.Context, c Caller, bookingID int64) error {
	if !c.Cleaner {
		return nil
	}
	ok, err := s.repo.CleanerAssigned(ctx, c.UserID, bookingID)
	if err != nil {
		return err
	}
	if !ok {
		return errNotYourJob
	}
	return nil
}

// Templates

func (s *Service) ListTemplates(ctx context.Context, params pagination.Params) ([]Template, int, error) {
	return s.repo.ListTemplates(ctx, params)
}

func (s *Service) CreateTemplate(ctx context.Context, c Caller, req CreateTemplateRequest) (Template, error) {
	if c.Cleaner {
		return Template{}, errStaffOnly
	}
	if err := req.Validate(); err != nil {
		return Template{}, err
	}
	return s.repo.CreateTemplate(ctx, req.Name, req.ServiceType, req.items)
}

func (s *Service) UpdateTemplate(ctx context.Context, c Caller, id int64, req UpdateTemplateRequest) (Template, error) {
	if c.Cleaner {
		return Template{}, errStaffOnly
	}
	if err := req.Validate(); err != nil {
		return Template{}, err
	}
	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}
	t, err := s.repo.UpdateTemplate(ctx, id, req.Name, req.ServiceType, active, req.items)
	if err != nil {
		return Template{}, mapRepoError(err)
	}
	return t, nil
}

// Checklists

// GetByBooking returns the booking's checklist. When a cleaner opens a job
// that has none yet, it is created from the booking's template: that is how
// the checklist reaches the crew without the office creating each one.
func (s *Service) GetByBooking(ctx context.Context, c Caller, bookingID int64) (Checklist, error) {
	if err := s.checkBooking(ctx, c, bookingID); err != nil {
		return Checklist{}, err
	}
	cl, err := s.repo.GetByBooking(ctx, bookingID)
	if errors.Is(err, ErrNotFound) && c.Cleaner {
		cl, err = s.createFromTemplate(ctx, bookingID, nil)
		if errors.Is(err, ErrAlreadyExists) {
			cl, err = s.repo.GetByBooking(ctx, bookingID)
		}
		if errors.Is(err, ErrNoTemplate) {
			err = ErrNotFound
		}
	}
	if err != nil {
		return Checklist{}, mapRepoError(err)
	}
	return cl, nil
}

func (s *Service) createFromTemplate(ctx context.Context, bookingID int64, templateID *int64) (Checklist, error) {
	if templateID == nil {
		id, err := s.repo.TemplateForBooking(ctx, bookingID)
		if err != nil {
			return Checklist{}, err
		}
		templateID = &id
	}
	return s.repo.Create(ctx, bookingID, templateID, nil)
}

func (s *Service) Create(ctx context.Context, c Caller, req CreateChecklistRequest) (Checklist, error) {
	if err := req.Validate(); err != nil {
		return Checklist{}, err
	}
	if err := s.checkBooking(ctx, c, req.BookingID); err != nil {
		return Checklist{}, err
	}
	// Cleaners always get the booking's own template.
	if c.Cleaner {
		req.TemplateID, req.items = nil, nil
	}
	var (
		cl  Checklist
		err error
	)
	if len(req.items) > 0 {
		cl, err = s.repo.Create(ctx, req.BookingID, req.TemplateID, req.items)
	} else {
		cl, err = s.createFromTemplate(ctx, req.BookingID, req.TemplateID)
	}
	if err != nil {
		return Checklist{}, mapRepoError(err)
	}
	return cl, nil
}

// Statuses gives job cards their checklist badge.
func (s *Service) Statuses(ctx context.Context, c Caller, bookingIDs []int64) ([]StatusRow, error) {
	if len(bookingIDs) == 0 {
		return []StatusRow{}, nil
	}
	var scope *int64
	if c.Cleaner {
		scope = &c.UserID
	}
	return s.repo.Statuses(ctx, bookingIDs, scope)
}

func (s *Service) List(ctx context.Context, c Caller, f ListFilter) ([]Summary, int, error) {
	if c.Cleaner {
		return nil, 0, errStaffOnly
	}
	return s.repo.List(ctx, f)
}

// itemFor loads an item and checks the caller may change it.
func (s *Service) itemFor(ctx context.Context, c Caller, itemID int64) (ItemContext, error) {
	ic, err := s.repo.ItemContext(ctx, itemID)
	if err != nil {
		return ItemContext{}, mapRepoError(err)
	}
	if err := s.checkBooking(ctx, c, ic.BookingID); err != nil {
		return ItemContext{}, err
	}
	if ic.Locked {
		return ItemContext{}, errLocked
	}
	return ic, nil
}

func (s *Service) CompleteItem(ctx context.Context, c Caller, itemID int64, req CompleteItemRequest) (Checklist, error) {
	ic, err := s.itemFor(ctx, c, itemID)
	if err != nil {
		return Checklist{}, err
	}
	u, err := resolveItem(ic.Item, req, c.UserID, time.Now())
	if err != nil {
		return Checklist{}, err
	}
	cl, err := s.repo.UpdateItem(ctx, ic, u)
	if err != nil {
		return Checklist{}, mapRepoError(err)
	}
	return cl, nil
}

// resolveItem turns a request into the item's new state. Score and
// pass/fail items count as done once answered; a tick item follows
// isCompleted. Who answered and when are recorded from the login.
func resolveItem(cur ChecklistItem, req CompleteItemRequest, userID int64, now time.Time) (ItemUpdate, error) {
	u := ItemUpdate{Score: cur.Score, Result: cur.Result}
	switch cur.ItemType {
	case ItemScore:
		switch {
		case req.Score != nil && *req.Score == 0:
			u.Score = nil
		case req.Score != nil:
			if *req.Score < 1 || *req.Score > 5 {
				return ItemUpdate{}, response.NewAPIError(400, "score must be 1 to 5")
			}
			v := *req.Score
			u.Score = &v
		case !req.IsCompleted:
			u.Score = nil
		}
		u.Completed = u.Score != nil
	case ItemPassFail:
		switch {
		case req.Result != nil && strings.TrimSpace(*req.Result) == "":
			u.Result = nil
		case req.Result != nil:
			v := strings.TrimSpace(*req.Result)
			if v != ResultPass && v != ResultFail && v != ResultNA {
				return ItemUpdate{}, response.NewAPIError(400, "result must be pass, fail or na")
			}
			u.Result = &v
		case !req.IsCompleted:
			u.Result = nil
		}
		u.Completed = u.Result != nil
	default:
		u.Completed = req.IsCompleted
	}
	if req.Notes != nil {
		v := strings.TrimSpace(*req.Notes)
		u.Notes = &v
	}
	if req.CompletedBy != nil {
		v := strings.TrimSpace(*req.CompletedBy)
		u.ByName = &v
	}
	if u.Completed {
		u.CompletedAt = cur.CompletedAt
		answerChanged := !sameInt(cur.Score, u.Score) || !sameStr(cur.Result, u.Result)
		if !cur.IsCompleted || answerChanged || u.CompletedAt == nil {
			u.CompletedAt = &now
		}
		if userID > 0 {
			u.ByUserID = &userID
		}
	}
	return u, nil
}

func sameInt(a, b *int) bool {
	return (a == nil && b == nil) || (a != nil && b != nil && *a == *b)
}

func sameStr(a, b *string) bool {
	return (a == nil && b == nil) || (a != nil && b != nil && *a == *b)
}

func (s *Service) Confirm(ctx context.Context, c Caller, bookingID int64, req ConfirmChecklistRequest) (Checklist, error) {
	if err := req.Validate(); err != nil {
		return Checklist{}, err
	}
	if err := s.checkBooking(ctx, c, bookingID); err != nil {
		return Checklist{}, err
	}
	cl, err := s.repo.Confirm(ctx, bookingID, req.ClientSignature)
	if err != nil {
		return Checklist{}, mapRepoError(err)
	}
	return cl, nil
}

// SubmitProblems lists what stops a checklist from being submitted.
func SubmitProblems(cl Checklist) []string {
	var out []string
	for _, it := range cl.Items {
		switch {
		case !it.IsCompleted:
			out = append(out, fmt.Sprintf("%q is not done", it.Label))
		case it.RequirePhoto && it.AfterPhotoURL == nil:
			out = append(out, fmt.Sprintf("%q needs an after photo", it.Label))
		case it.Result != nil && *it.Result == ResultFail && strings.TrimSpace(it.Notes) == "":
			out = append(out, fmt.Sprintf("%q failed: add a note saying why", it.Label))
		}
	}
	return out
}

func (s *Service) Submit(ctx context.Context, c Caller, bookingID int64) (Checklist, error) {
	if err := s.checkBooking(ctx, c, bookingID); err != nil {
		return Checklist{}, err
	}
	cl, err := s.repo.GetByBooking(ctx, bookingID)
	if err != nil {
		return Checklist{}, mapRepoError(err)
	}
	if cl.Locked() {
		return Checklist{}, errLocked
	}
	if p := SubmitProblems(cl); len(p) > 0 {
		return Checklist{}, response.NewAPIError(422, "cannot submit yet: "+strings.Join(p, "; "))
	}
	cl, err = s.repo.Submit(ctx, bookingID, c.UserID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return Checklist{}, errLocked
		}
		return Checklist{}, mapRepoError(err)
	}
	s.notifySubmitted(ctx, cl)
	return cl, nil
}

func (s *Service) notifySubmitted(ctx context.Context, cl Checklist) {
	if s.notifier == nil {
		return
	}
	by := "a cleaner"
	if cl.SubmittedBy != nil {
		by = *cl.SubmittedBy
	}
	fails := 0
	for _, it := range cl.Items {
		if it.Result != nil && *it.Result == ResultFail {
			fails++
		}
	}
	body := fmt.Sprintf("<p>%s submitted the checklist for booking #%d (%d items).</p>",
		html.EscapeString(by), cl.BookingID, len(cl.Items))
	if avg := averageScore(cl.Items); avg != nil {
		body += fmt.Sprintf("<p>Average score: %.1f / 5</p>", *avg)
	}
	if fails > 0 {
		body += fmt.Sprintf("<p><strong>%d item(s) failed.</strong></p>", fails)
	}
	body += "<p>Open Checklists → Review to approve it or send it back.</p>"
	s.notifier.NotifyOffice(ctx, "checklist.submitted", fmt.Sprintf("Checklist submitted for booking #%d", cl.BookingID), body)
}

func (s *Service) Review(ctx context.Context, c Caller, bookingID int64, req ReviewRequest) (Checklist, error) {
	if c.Cleaner {
		return Checklist{}, errStaffOnly
	}
	if err := req.Validate(); err != nil {
		return Checklist{}, err
	}
	cl, err := s.repo.Review(ctx, bookingID, c.UserID, req.Decision, req.Note)
	if errors.Is(err, ErrNotFound) {
		if _, gerr := s.repo.GetByBooking(ctx, bookingID); gerr == nil {
			return Checklist{}, errNotSubmitted
		}
	}
	if err != nil {
		return Checklist{}, mapRepoError(err)
	}
	return cl, nil
}

// AttachPhoto records an uploaded photo URL after the file is stored.
func (s *Service) AttachPhoto(ctx context.Context, c Caller, itemID int64, kind, url string) (Checklist, error) {
	k, err := normalizeKind(kind)
	if err != nil {
		return Checklist{}, err
	}
	ic, err := s.itemFor(ctx, c, itemID)
	if err != nil {
		return Checklist{}, err
	}
	cl, err := s.repo.SetItemPhoto(ctx, ic, k, url)
	if err != nil {
		return Checklist{}, mapRepoError(err)
	}
	return cl, nil
}

// CheckItem is the access check for a photo upload, run before the file is
// written so rejected uploads leave nothing on disk.
func (s *Service) CheckItem(ctx context.Context, c Caller, itemID int64) error {
	_, err := s.itemFor(ctx, c, itemID)
	return err
}

// CheckPhoto lets a cleaner download only photos of their own jobs.
func (s *Service) CheckPhoto(ctx context.Context, c Caller, itemID int64) error {
	if !c.Cleaner {
		return nil
	}
	ic, err := s.repo.ItemContext(ctx, itemID)
	if err != nil {
		return mapRepoError(err)
	}
	return s.checkBooking(ctx, c, ic.BookingID)
}

func mapRepoError(err error) error {
	var apiErr *response.APIError
	if errors.As(err, &apiErr) {
		return err
	}
	switch {
	case errors.Is(err, ErrTemplateNotFound):
		return response.NewAPIError(422, "that checklist template has no items or does not exist")
	case errors.Is(err, ErrNoTemplate):
		return response.NewAPIError(422, "no checklist template yet: add one under Checklists → Templates")
	case errors.Is(err, ErrAlreadyExists):
		return response.NewAPIError(409, "this booking already has a checklist")
	case errors.Is(err, ErrAlreadySigned):
		return response.NewAPIError(409, "the client has already confirmed this checklist")
	}
	if errors.Is(err, ErrItemNotFound) {
		return dberror.Map(err, ErrItemNotFound, "checklist item not found")
	}
	return dberror.Map(err, ErrNotFound, "checklist not found")
}
