package bookings

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// TEST_DATABASE_URL=postgres://... go test ./internal/bookings/ -run ChecklistMissing
func TestChecklistMissingDB(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	repo := NewRepository(pool)
	tag := fmt.Sprint(time.Now().UnixNano())
	svc := "gate" + tag

	var noTpl, withTpl int64
	if err := pool.QueryRow(ctx, `INSERT INTO bookings (booking_number, customer_name, service_type, scheduled_for)
		VALUES ($1, 'Gate', $2, now()) RETURNING id`, "G1"+tag, "other"+tag).Scan(&noTpl); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO bookings (booking_number, customer_name, service_type, scheduled_for)
		VALUES ($1, 'Gate', $2, now()) RETURNING id`, "G2"+tag, svc).Scan(&withTpl); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM bookings WHERE id = ANY($1)`, []int64{noTpl, withTpl})
		_, _ = pool.Exec(ctx, `DELETE FROM checklist_templates WHERE service_type = $1`, svc)
	})

	// A service with no template (and no general template in this test DB
	// would make every job need one, so only assert the specific cases).
	var tplID int64
	if err := pool.QueryRow(ctx, `INSERT INTO checklist_templates (name, service_type) VALUES ($1, $2) RETURNING id`, "gate"+tag, svc).Scan(&tplID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO checklist_template_items (template_id, label) VALUES ($1, 'Floor')`, tplID); err != nil {
		t.Fatal(err)
	}

	if missing, err := repo.ChecklistMissing(ctx, withTpl); err != nil || !missing {
		t.Fatalf("template applies, no checklist yet: missing=%v err=%v", missing, err)
	}
	var clID int64
	if err := pool.QueryRow(ctx, `INSERT INTO booking_checklists (booking_id, template_id) VALUES ($1, $2) RETURNING id`, withTpl, tplID).Scan(&clID); err != nil {
		t.Fatal(err)
	}
	if missing, _ := repo.ChecklistMissing(ctx, withTpl); !missing {
		t.Fatal("unsubmitted checklist must still block completion")
	}
	if _, err := pool.Exec(ctx, `UPDATE booking_checklists SET submitted_at = now() WHERE id = $1`, clID); err != nil {
		t.Fatal(err)
	}
	if missing, _ := repo.ChecklistMissing(ctx, withTpl); missing {
		t.Fatal("submitted checklist must allow completion")
	}

	var general int
	_ = pool.QueryRow(ctx, `SELECT COUNT(*) FROM checklist_templates t WHERE t.is_active AND t.service_type = ''
		AND EXISTS (SELECT 1 FROM checklist_template_items i WHERE i.template_id = t.id)`).Scan(&general)
	missing, err := repo.ChecklistMissing(ctx, noTpl)
	if err != nil || missing != (general > 0) {
		t.Fatalf("no specific template: missing=%v (general templates: %d) err=%v", missing, general, err)
	}
}
