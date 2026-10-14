package checklists

import (
	"encoding/json"
	"testing"
	"time"
)

func intp(v int) *int       { return &v }
func strp(v string) *string { return &v }

func TestTemplateItemsAcceptStringsAndObjects(t *testing.T) {
	var req CreateTemplateRequest
	body := `{"name":"Condo","items":["Floors mopped",{"label":"Kitchen","itemType":"score","requirePhoto":true}]}`
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if err := req.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if req.items[0].ItemType != ItemCheck || req.items[0].Label != "Floors mopped" {
		t.Fatalf("plain label should become a tick item: %+v", req.items[0])
	}
	if req.items[1].ItemType != ItemScore || !req.items[1].RequirePhoto {
		t.Fatalf("object item lost its settings: %+v", req.items[1])
	}
	bad := CreateTemplateRequest{Name: "x", Items: []TemplateItemInput{{Label: "a", ItemType: "stars"}}}
	if err := bad.Validate(); err == nil {
		t.Fatal("unknown item type must be rejected")
	}
}

func TestResolveScoreItem(t *testing.T) {
	now := time.Now()
	cur := ChecklistItem{ID: 1, ItemType: ItemScore}
	u, err := resolveItem(cur, CompleteItemRequest{IsCompleted: true, Score: intp(4)}, 7, now)
	if err != nil || !u.Completed || *u.Score != 4 || *u.ByUserID != 7 || u.CompletedAt == nil {
		t.Fatalf("score 4 should complete the item: %+v %v", u, err)
	}
	if _, err := resolveItem(cur, CompleteItemRequest{Score: intp(6)}, 7, now); err == nil {
		t.Fatal("score 6 must be rejected")
	}
	// Notes-only update keeps the existing score.
	cur.Score, cur.IsCompleted = intp(3), true
	u, _ = resolveItem(cur, CompleteItemRequest{IsCompleted: true, Notes: strp("dusty shelf")}, 7, now)
	if !u.Completed || *u.Score != 3 || *u.Notes != "dusty shelf" {
		t.Fatalf("notes update lost the score: %+v", u)
	}
	u, _ = resolveItem(cur, CompleteItemRequest{Score: intp(0)}, 7, now)
	if u.Completed || u.Score != nil || u.ByUserID != nil {
		t.Fatalf("score 0 should clear the item: %+v", u)
	}
}

func TestResolvePassFailAndTick(t *testing.T) {
	now := time.Now()
	u, err := resolveItem(ChecklistItem{ItemType: ItemPassFail}, CompleteItemRequest{Result: strp("fail")}, 2, now)
	if err != nil || !u.Completed || *u.Result != ResultFail {
		t.Fatalf("fail answer should complete the item: %+v %v", u, err)
	}
	if _, err := resolveItem(ChecklistItem{ItemType: ItemPassFail}, CompleteItemRequest{Result: strp("maybe")}, 2, now); err == nil {
		t.Fatal("unknown result must be rejected")
	}
	u, _ = resolveItem(ChecklistItem{ItemType: ItemCheck}, CompleteItemRequest{IsCompleted: true}, 2, now)
	if !u.Completed || u.Score != nil || u.Result != nil {
		t.Fatalf("tick item: %+v", u)
	}
}

func TestSubmitProblems(t *testing.T) {
	photo := "/api/v1/checklists/photos/item1_after_1.jpg"
	ok := Checklist{Items: []ChecklistItem{
		{Label: "Floor", ItemType: ItemCheck, IsCompleted: true},
		{Label: "Kitchen", ItemType: ItemScore, IsCompleted: true, Score: intp(5), RequirePhoto: true, AfterPhotoURL: &photo},
		{Label: "Bins", ItemType: ItemPassFail, IsCompleted: true, Result: strp(ResultFail), Notes: "no bags left"},
	}}
	if p := SubmitProblems(ok); len(p) != 0 {
		t.Fatalf("complete checklist flagged: %v", p)
	}
	bad := Checklist{Items: []ChecklistItem{
		{Label: "Floor", IsCompleted: false},
		{Label: "Kitchen", IsCompleted: true, RequirePhoto: true},
		{Label: "Bins", IsCompleted: true, Result: strp(ResultFail)},
	}}
	if p := SubmitProblems(bad); len(p) != 3 {
		t.Fatalf("want 3 problems, got %v", p)
	}
	if avg := averageScore(ok.Items); avg == nil || *avg != 5 {
		t.Fatalf("average score: %v", avg)
	}
}

func TestReviewValidation(t *testing.T) {
	if err := (&ReviewRequest{Decision: "approved"}).Validate(); err != nil {
		t.Fatalf("approve rejected: %v", err)
	}
	if err := (&ReviewRequest{Decision: "rework"}).Validate(); err == nil {
		t.Fatal("rework without a note must be rejected")
	}
	if err := (&ReviewRequest{Decision: "maybe"}).Validate(); err == nil {
		t.Fatal("unknown decision must be rejected")
	}
}

func TestPhotoItemID(t *testing.T) {
	if id, ok := PhotoItemID("item42_after_1700000000.jpg"); !ok || id != 42 {
		t.Fatalf("got %d %v", id, ok)
	}
	for _, n := range []string{"chat_1.jpg", "item_after.jpg", "itemx_after.jpg", "../item3_a.jpg/x"} {
		if _, ok := PhotoItemID(n); ok {
			t.Fatalf("%q should not parse", n)
		}
	}
}
