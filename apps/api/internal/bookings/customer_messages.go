package bookings

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/odysight/crm/internal/notifications"
)

// CustomerContact is how the office reaches a booking's customer.
type CustomerContact struct {
	Name       string
	Phone      string
	LineUserID string
}

// CustomerContact loads the phone and LINE id for a customer. A missing
// customer yields a zero contact, not an error.
func (r *Repository) CustomerContact(ctx context.Context, id int64) (CustomerContact, error) {
	var c CustomerContact
	err := r.pool.QueryRow(ctx,
		`SELECT TRIM(first_name || ' ' || last_name), COALESCE(phone, ''), line_user_id
		   FROM customers WHERE id = $1`, id).Scan(&c.Name, &c.Phone, &c.LineUserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return CustomerContact{}, nil
	}
	if err != nil {
		return CustomerContact{}, fmt.Errorf("get customer %d contact: %w", id, err)
	}
	return c, nil
}

// ClaimCompletionNotice marks the booking's "job done" message as sent and
// reports whether this call won the claim, so it goes out at most once even
// if a job is completed, reopened and completed again.
func (r *Repository) ClaimCompletionNotice(ctx context.Context, id int64) (bool, error) {
	tag, err := r.pool.Exec(ctx,
		`UPDATE bookings SET completion_notified_at = now()
		  WHERE id = $1 AND completion_notified_at IS NULL AND status = 'completed'`, id)
	if err != nil {
		return false, fmt.Errorf("claim completion notice %d: %w", id, err)
	}
	return tag.RowsAffected() == 1, nil
}

// ClaimDueReminders atomically marks and returns bookings scheduled within
// [from, to) that still need a reminder. Claiming before sending means a
// crash can drop a reminder but never send it twice.
func (r *Repository) ClaimDueReminders(ctx context.Context, from, to time.Time) ([]Booking, error) {
	rows, err := r.pool.Query(ctx,
		`UPDATE bookings SET reminder_sent_at = now()
		  WHERE reminder_sent_at IS NULL
		    AND status IN ('pending', 'confirmed')
		    AND customer_id IS NOT NULL
		    AND scheduled_for >= $1 AND scheduled_for < $2
		 RETURNING `+bookingColumns, from, to)
	if err != nil {
		return nil, fmt.Errorf("claim due reminders: %w", err)
	}
	defer rows.Close()
	var out []Booking
	for rows.Next() {
		b, err := scanBooking(rows)
		if err != nil {
			return nil, fmt.Errorf("scan reminder booking: %w", err)
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

var thaiWeekdays = [...]string{"อาทิตย์", "จันทร์", "อังคาร", "พุธ", "พฤหัสบดี", "ศุกร์", "เสาร์"}
var thaiMonths = [...]string{"ม.ค.", "ก.พ.", "มี.ค.", "เม.ย.", "พ.ค.", "มิ.ย.", "ก.ค.", "ส.ค.", "ก.ย.", "ต.ค.", "พ.ย.", "ธ.ค."}

// thaiDateTime formats t in local time the way Thai customers read dates:
// "วันจันทร์ที่ 5 ต.ค. 2569 เวลา 09:00 น." (Buddhist era year).
func thaiDateTime(t time.Time) string {
	t = t.Local()
	return fmt.Sprintf("วัน%sที่ %d %s %d เวลา %s น.",
		thaiWeekdays[t.Weekday()], t.Day(), thaiMonths[t.Month()-1], t.Year()+543, t.Format("15:04"))
}

func lineConfirmationText(b Booking) string {
	var s strings.Builder
	s.WriteString("✅ ยืนยันการจองบริการ Smile Clean\n")
	fmt.Fprintf(&s, "เลขที่การจอง: %s\n", b.BookingNumber)
	fmt.Fprintf(&s, "บริการ: %s\n", b.ServiceType)
	fmt.Fprintf(&s, "%s\n", thaiDateTime(b.ScheduledFor))
	if a := strings.TrimSpace(b.Address); a != "" {
		fmt.Fprintf(&s, "สถานที่: %s\n", a)
	}
	s.WriteString("\nหากต้องการเลื่อนหรือยกเลิก ตอบกลับข้อความนี้ได้เลยค่ะ 🙏")
	return s.String()
}

func lineReminderText(b Booking) string {
	var s strings.Builder
	s.WriteString("⏰ แจ้งเตือนนัดหมายพรุ่งนี้\n")
	fmt.Fprintf(&s, "ทีมงาน Smile Clean จะเข้าให้บริการ %s\n", b.ServiceType)
	fmt.Fprintf(&s, "%s\n", thaiDateTime(b.ScheduledFor))
	if a := strings.TrimSpace(b.Address); a != "" {
		fmt.Fprintf(&s, "สถานที่: %s\n", a)
	}
	fmt.Fprintf(&s, "เลขที่การจอง: %s\n", b.BookingNumber)
	s.WriteString("\nกรุณาเตรียมพื้นที่ให้พร้อม หากมีการเปลี่ยนแปลงแจ้งได้ที่แชทนี้ค่ะ")
	return s.String()
}

func lineCompletedText(b Booking) string {
	return fmt.Sprintf("✨ งานทำความสะอาดเสร็จเรียบร้อยแล้วค่ะ\n"+
		"เลขที่การจอง: %s\nบริการ: %s\n\n"+
		"ขอบคุณที่ไว้วางใจ Smile Clean 💙\n"+
		"พอใจกับบริการไหมคะ? ให้คะแนน 1–5 ดาวตอบกลับข้อความนี้ได้เลย",
		b.BookingNumber, b.ServiceType)
}

// notifyLINE pushes text to the booking customer's LINE chat when they have
// one. Runs in the caller's goroutine.
func (s *Service) notifyLINE(ctx context.Context, b Booking, eventType, text string) {
	if s.notifier == nil || b.CustomerID == nil {
		return
	}
	c, err := s.repo.CustomerContact(ctx, *b.CustomerID)
	if err != nil {
		slog.Warn("line notify: load contact failed", "booking", b.BookingNumber, "error", err)
		return
	}
	s.notifier.EmitLINE(ctx, eventType, c.LineUserID, c.Name, text)
}

// notifyCompletedOnce sends the job-done message the first time a booking
// reaches completed.
func (s *Service) notifyCompletedOnce(b Booking) {
	if s.notifier == nil || b.Status != StatusCompleted || b.CustomerID == nil {
		return
	}
	go func() {
		ctx := context.Background()
		won, err := s.repo.ClaimCompletionNotice(ctx, b.ID)
		if err != nil || !won {
			return
		}
		s.notifyLINE(ctx, b, notifications.EventBookingCompleted, lineCompletedText(b))
	}()
}

// ReminderRunner sends a LINE reminder the day before each booking.
type ReminderRunner struct {
	service  *Service
	interval time.Duration
	now      func() time.Time
}

func NewReminderRunner(service *Service, interval time.Duration) *ReminderRunner {
	return &ReminderRunner{service: service, interval: interval, now: time.Now}
}

// Run blocks until ctx is cancelled, firing once immediately then every interval.
func (r *ReminderRunner) Run(ctx context.Context) {
	r.fire(ctx)
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.fire(ctx)
		}
	}
}

// reminderWindow is "tomorrow" in local time, but only from 09:00 today so
// customers are not pinged overnight.
func reminderWindow(now time.Time) (from, to time.Time, ok bool) {
	now = now.Local()
	if now.Hour() < 9 || now.Hour() >= 20 {
		return time.Time{}, time.Time{}, false
	}
	y, m, d := now.Date()
	from = time.Date(y, m, d+1, 0, 0, 0, 0, time.Local)
	return from, from.AddDate(0, 0, 1), true
}

func (r *ReminderRunner) fire(ctx context.Context) {
	from, to, ok := reminderWindow(r.now())
	if !ok || r.service.notifier == nil {
		return
	}
	due, err := r.service.repo.ClaimDueReminders(ctx, from, to)
	if err != nil {
		slog.Warn("booking reminder run failed", "error", err)
		return
	}
	for _, b := range due {
		r.service.notifyLINE(ctx, b, notifications.EventBookingReminder, lineReminderText(b))
	}
	if len(due) > 0 {
		slog.Info("booking reminders sent", "count", len(due))
	}
}
