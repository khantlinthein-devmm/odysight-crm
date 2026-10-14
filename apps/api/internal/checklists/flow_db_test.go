package checklists

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/odysight/crm/pkg/response"
)

// The send → score → submit → review flow against a real database, because
// the cleaner scoping and locking live in SQL. Skips unless
// TEST_DATABASE_URL is set:
//
//	TEST_DATABASE_URL=postgres://... go test ./internal/checklists/ -run DB
func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(context.Background()); err != nil {
		t.Fatalf("ping: %v", err)
	}
	return pool
}

func apiStatus(err error) int {
	var e *response.APIError
	if errors.As(err, &e) {
		return e.Status
	}
	return 0
}

func TestChecklistFlowDB(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	tag := fmt.Sprint(time.Now().UnixNano())
	q := func(sql string, args ...any) int64 {
		t.Helper()
		var id int64
		if err := pool.QueryRow(ctx, sql, args...).Scan(&id); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
		return id
	}
	userA := q(`INSERT INTO users (name, email, password_hash, role) VALUES ('Aye Aye', $1, 'x', 'CLEANER') RETURNING id`, "a"+tag+"@t")
	userB := q(`INSERT INTO users (name, email, password_hash, role) VALUES ('Bo Bo', $1, 'x', 'CLEANER') RETURNING id`, "b"+tag+"@t")
	admin := q(`INSERT INTO users (name, email, password_hash, role) VALUES ('Office', $1, 'x', 'ADMIN') RETURNING id`, "o"+tag+"@t")
	cleanerA := q(`INSERT INTO cleaners (first_name, last_name, phone, email, user_id) VALUES ('Aye', 'Aye', '1', $1, $2) RETURNING id`, "a"+tag+"@t", userA)
	// Bo Bo's profile is linked by email only.
	q(`INSERT INTO cleaners (first_name, last_name, phone, email) VALUES ('Bo', 'Bo', '2', $1) RETURNING id`, "b"+tag+"@t")
	booking := q(`INSERT INTO bookings (booking_number, customer_name, service_type, scheduled_for) VALUES ($1, 'Test Condo', $2, now()) RETURNING id`, "T"+tag, "svc"+tag)
	q(`INSERT INTO booking_cleaners (booking_id, cleaner_id, role) VALUES ($1, $2, 'primary') RETURNING id`, booking, cleanerA)
	legacy := q(`INSERT INTO bookings (booking_number, customer_name, service_type, scheduled_for, assigned_cleaner) VALUES ($1, 'Legacy', $2, now(), 'Bo Bo') RETURNING id`, "L"+tag, "svc"+tag)
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM bookings WHERE id = ANY($1)`, []int64{booking, legacy})
		_, _ = pool.Exec(ctx, `DELETE FROM checklist_templates WHERE name = $1`, "tpl"+tag)
		_, _ = pool.Exec(ctx, `DELETE FROM cleaners WHERE email LIKE $1`, "%"+tag+"@t")
		_, _ = pool.Exec(ctx, `DELETE FROM users WHERE email LIKE $1`, "%"+tag+"@t")
	})

	svc := NewService(NewRepository(pool))
	a := Caller{UserID: userA, Cleaner: true}
	b := Caller{UserID: userB, Cleaner: true}
	office := Caller{UserID: admin}

	// Office template with all three item types; condo bookings pick it.
	tpl, err := svc.CreateTemplate(ctx, office, CreateTemplateRequest{Name: "tpl" + tag, ServiceType: "svc" + tag, Items: []TemplateItemInput{
		{Label: "Floors"}, {Label: "Kitchen", ItemType: "score"}, {Label: "Bins", ItemType: "pass_fail", RequirePhoto: true},
	}})
	if err != nil {
		t.Fatalf("create template: %v", err)
	}
	if _, err := svc.CreateTemplate(ctx, a, CreateTemplateRequest{Name: "x", Items: []TemplateItemInput{{Label: "y"}}}); apiStatus(err) != 403 {
		t.Fatalf("cleaner created a template: %v", err)
	}

	// Office does not auto-create; the assigned cleaner does on open.
	if _, err := svc.GetByBooking(ctx, office, booking); apiStatus(err) != 404 {
		t.Fatalf("office open should 404 before the cleaner opens: %v", err)
	}
	if _, err := svc.GetByBooking(ctx, b, booking); apiStatus(err) != 403 {
		t.Fatalf("other cleaner opened the job: %v", err)
	}
	cl, err := svc.GetByBooking(ctx, a, booking)
	if err != nil || len(cl.Items) != 3 || *cl.TemplateID != tpl.ID {
		t.Fatalf("cleaner open should create from template: %+v %v", cl, err)
	}
	floors, kitchen, bins := cl.Items[0], cl.Items[1], cl.Items[2]
	if kitchen.ItemType != ItemScore || !bins.RequirePhoto {
		t.Fatalf("item settings not copied: %+v %+v", kitchen, bins)
	}
	// Legacy name-assigned job, email-linked profile.
	if _, err := svc.GetByBooking(ctx, b, legacy); err != nil {
		t.Fatalf("legacy assigned cleaner denied: %v", err)
	}

	if _, err := svc.CompleteItem(ctx, b, floors.ID, CompleteItemRequest{IsCompleted: true}); apiStatus(err) != 403 {
		t.Fatalf("other cleaner ticked an item: %v", err)
	}
	if _, err := svc.CompleteItem(ctx, a, floors.ID, CompleteItemRequest{IsCompleted: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CompleteItem(ctx, a, kitchen.ID, CompleteItemRequest{IsCompleted: true, Score: intp(4)}); err != nil {
		t.Fatal(err)
	}
	cl, err = svc.CompleteItem(ctx, a, bins.ID, CompleteItemRequest{IsCompleted: true, Result: strp("fail")})
	if err != nil || cl.Status != StatusCompleted {
		t.Fatalf("all answered should be completed: %v %v", cl.Status, err)
	}
	if cl.Items[1].CompletedBy == nil || *cl.Items[1].CompletedBy != "Aye Aye" || *cl.Items[1].Score != 4 {
		t.Fatalf("who scored not recorded: %+v", cl.Items[1])
	}

	// Fail without note and missing required photo block submit.
	if _, err := svc.Submit(ctx, a, booking); apiStatus(err) != 422 {
		t.Fatalf("incomplete submit allowed: %v", err)
	}
	if _, err := svc.CompleteItem(ctx, a, bins.ID, CompleteItemRequest{IsCompleted: true, Notes: strp("bin bags ran out")}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.repo.SetItemPhoto(ctx, ItemContext{Item: ChecklistItem{ID: bins.ID}, BookingID: booking}, "after", "/api/v1/checklists/photos/item1_after_1.jpg"); err != nil {
		t.Fatal(err)
	}
	cl, err = svc.Submit(ctx, a, booking)
	if err != nil || !cl.Locked() || cl.SubmittedBy == nil || *cl.SubmittedBy != "Aye Aye" {
		t.Fatalf("submit: %+v %v", cl, err)
	}
	if _, err := svc.CompleteItem(ctx, a, floors.ID, CompleteItemRequest{IsCompleted: false}); apiStatus(err) != 409 {
		t.Fatalf("submitted checklist still editable: %v", err)
	}

	// Office list and job-card statuses.
	rows, total, err := svc.List(ctx, office, ListFilter{State: "submitted", CleanerID: cleanerA, Search: tag, Limit: 10})
	if err != nil || total != 1 || rows[0].ItemsDone != 3 || rows[0].FailCount != 1 || rows[0].PhotoCount != 1 ||
		rows[0].AvgScore == nil || *rows[0].AvgScore != 4 || rows[0].Cleaners != "Aye Aye" {
		t.Fatalf("list: %+v total=%d err=%v", rows, total, err)
	}
	if _, _, err := svc.List(ctx, a, ListFilter{Limit: 10}); apiStatus(err) != 403 {
		t.Fatalf("cleaner listed all checklists: %v", err)
	}
	st, err := svc.Statuses(ctx, b, []int64{booking, legacy})
	if err != nil || len(st) != 1 || st[0].BookingID != legacy {
		t.Fatalf("cleaner statuses not scoped: %+v %v", st, err)
	}

	// Rework unlocks with a note; resubmit; approve.
	if _, err := svc.Review(ctx, a, booking, ReviewRequest{Decision: "approved"}); apiStatus(err) != 403 {
		t.Fatalf("cleaner reviewed: %v", err)
	}
	cl, err = svc.Review(ctx, office, booking, ReviewRequest{Decision: "rework", Note: "Kitchen sink still dirty"})
	if err != nil || cl.Locked() || cl.ReviewStatus != ReviewRework {
		t.Fatalf("rework: %+v %v", cl, err)
	}
	if rows, _, _ := svc.List(ctx, office, ListFilter{State: "rework", Search: tag, Limit: 10}); len(rows) != 1 {
		t.Fatalf("rework filter: %+v", rows)
	}
	if _, err := svc.CompleteItem(ctx, a, kitchen.ID, CompleteItemRequest{IsCompleted: true, Score: intp(5)}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Submit(ctx, a, booking); err != nil {
		t.Fatal(err)
	}
	cl, err = svc.Review(ctx, office, booking, ReviewRequest{Decision: "approved"})
	if err != nil || cl.ReviewStatus != ReviewApproved || cl.ReviewedBy == nil || *cl.ReviewedBy != "Office" {
		t.Fatalf("approve: %+v %v", cl, err)
	}

	// Client signs once.
	if _, err := svc.Confirm(ctx, a, booking, ConfirmChecklistRequest{ClientSignature: "Khun Somchai"}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Confirm(ctx, a, booking, ConfirmChecklistRequest{ClientSignature: "Someone else"}); apiStatus(err) != 409 {
		t.Fatalf("signature overwritten: %v", err)
	}

	// Template edit with deactivate.
	off := false
	tpl2, err := svc.UpdateTemplate(ctx, office, tpl.ID, UpdateTemplateRequest{Name: "tpl" + tag, ServiceType: "svc" + tag, IsActive: &off,
		Items: []TemplateItemInput{{Label: "Only item", ItemType: "score"}}})
	if err != nil || tpl2.IsActive || len(tpl2.Items) != 1 {
		t.Fatalf("update template: %+v %v", tpl2, err)
	}
}
