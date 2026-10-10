package bookings

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/odysight/crm/internal/notifications"
)

func TestThaiDateTime(t *testing.T) {
	loc := time.Local
	got := thaiDateTime(time.Date(2026, 10, 5, 9, 30, 0, 0, loc))
	if got != "วันจันทร์ที่ 5 ต.ค. 2569 เวลา 09:30 น." {
		t.Fatalf("thaiDateTime = %s", got)
	}
}

func TestReminderWindow(t *testing.T) {
	loc := time.Local
	if _, _, ok := reminderWindow(time.Date(2026, 10, 4, 7, 0, 0, 0, loc)); ok {
		t.Error("no reminders before 09:00")
	}
	if _, _, ok := reminderWindow(time.Date(2026, 10, 4, 21, 0, 0, 0, loc)); ok {
		t.Error("no reminders after 20:00")
	}
	from, to, ok := reminderWindow(time.Date(2026, 10, 4, 10, 0, 0, 0, loc))
	if !ok || !from.Equal(time.Date(2026, 10, 5, 0, 0, 0, 0, loc)) || !to.Equal(time.Date(2026, 10, 6, 0, 0, 0, 0, loc)) {
		t.Fatalf("window = %v..%v ok=%v", from, to, ok)
	}
}

func TestLineTexts(t *testing.T) {
	b := Booking{BookingNumber: "BK-2026-0001", ServiceType: "deep", Address: "1 Silom",
		ScheduledFor: time.Date(2026, 10, 5, 9, 0, 0, 0, time.Local)}
	for name, text := range map[string]string{
		"confirm": lineConfirmationText(b), "reminder": lineReminderText(b), "done": lineCompletedText(b),
	} {
		if !strings.Contains(text, "BK-2026-0001") {
			t.Errorf("%s text lacks booking number: %s", name, text)
		}
	}
}

type fakePusher struct {
	mu   sync.Mutex
	sent []string
}

func (f *fakePusher) Push(_ context.Context, to, text string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, to+"|"+text)
	return nil
}

func (f *fakePusher) count(to string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, s := range f.sent {
		if strings.HasPrefix(s, to+"|") {
			n++
		}
	}
	return n
}

// Reminders and job-done notices must reach the customer's LINE exactly once.
func TestDBLineRemindersAndCompletionFireOnce(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	repo := NewRepository(pool)
	pusher := &fakePusher{}
	notifier := notifications.NewService(notifications.NewRepository(pool), nil, func() notifications.Emailer { return nil }).WithLINE(pusher)
	svc := NewService(repo, notifier)

	lineID := "U" + strings.Repeat("a", 31) + time.Now().Format("150405.000")
	var customerID int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO customers (first_name, phone, address, property_type, area, status, line_user_id)
		 VALUES ('Line Test', '0800000000', '1 Road', 'office', 'Silom', 'active', $1) RETURNING id`, lineID).Scan(&customerID); err != nil {
		t.Fatalf("seed customer: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM bookings WHERE customer_id = $1`, customerID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM customers WHERE id = $1`, customerID)
	})

	now := time.Date(2026, 10, 4, 10, 0, 0, 0, time.Local)
	b, err := repo.Create(ctx, Booking{CustomerName: "Line Test", CustomerID: &customerID, ServiceType: "deep",
		ScheduledFor: now.Add(24 * time.Hour), DurationMinutes: 60, Address: "1 Road", Status: StatusConfirmed})
	if err != nil {
		t.Fatalf("seed booking: %v", err)
	}

	runner := NewReminderRunner(svc, time.Hour)
	runner.now = func() time.Time { return now }
	runner.fire(ctx)
	runner.fire(ctx)
	if n := pusher.count(lineID); n != 1 {
		t.Fatalf("reminder pushes = %d, want 1", n)
	}

	done := string(StatusCompleted)
	for i := 0; i < 2; i++ {
		if _, err := svc.Update(ctx, b.ID, UpdateBookingRequest{Status: &done}); err != nil {
			t.Fatalf("complete: %v", err)
		}
	}
	deadline := time.Now().Add(3 * time.Second)
	for pusher.count(lineID) < 2 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	time.Sleep(100 * time.Millisecond)
	if n := pusher.count(lineID); n != 2 {
		t.Fatalf("total pushes after completion = %d, want 2 (reminder + one job-done)", n)
	}
}
