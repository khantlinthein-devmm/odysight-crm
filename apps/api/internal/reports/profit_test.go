package reports

import "testing"

func f(v float64) *float64 { return &v }

func TestComputeProfit(t *testing.T) {
	team := []TeamMember{{1, "Arti", "primary"}, {2, "Zaw", "crew"}, {3, "Mya", "crew"}}
	costs := BookingCosts{CleanerFees: map[int64]float64{1: 600, 2: 400, 9: 999}, Transport: 150, Other: 70, OtherNote: "parking"}
	p := computeProfit(9, f(3000), team, costs, 180)
	// Fee 999 belongs to cleaner 9, who is not on the team: ignored.
	if p.Labor != 1000 || p.Transport != 150 || p.Other != 70 || p.Supplies != 180 || p.Cost != 1400 {
		t.Fatalf("labor/transport/other/supplies/cost = %v/%v/%v/%v/%v", p.Labor, p.Transport, p.Other, p.Supplies, p.Cost)
	}
	if p.Profit == nil || *p.Profit != 1600 || p.Margin == nil || *p.Margin != 53.33 {
		t.Fatalf("profit/margin = %v/%v", p.Profit, p.Margin)
	}
	if p.MissingFees != 1 || p.Team[2].HasFee || p.Team[0].Fee != 600 || p.OtherNote != "parking" {
		t.Fatalf("team lines wrong: %+v", p.Team)
	}
}

func TestComputeProfitWithoutPriceOrCosts(t *testing.T) {
	p := computeProfit(1, nil, nil, BookingCosts{}, 50)
	if p.Profit != nil || p.Margin != nil || p.Cost != 50 || len(p.Team) != 0 {
		t.Fatalf("unexpected: %+v", p)
	}
	loss := computeProfit(2, f(500), []TeamMember{{1, "A", "primary"}}, BookingCosts{CleanerFees: map[int64]float64{1: 600}}, 0)
	if *loss.Profit != -100 || *loss.Margin != -20 {
		t.Fatalf("loss = %v / %v", *loss.Profit, *loss.Margin)
	}
}

func TestSaveCostsRequestParse(t *testing.T) {
	c, err := SaveCostsRequest{CleanerFees: map[string]float64{"5": 300}, Transport: 100, OtherNote: "  tolls  "}.parse()
	if err != nil || c.CleanerFees[5] != 300 || c.OtherNote != "tolls" {
		t.Fatalf("parse: %+v %v", c, err)
	}
	bad := []SaveCostsRequest{
		{Transport: -1},
		{Other: maxCost + 1},
		{CleanerFees: map[string]float64{"abc": 1}},
		{CleanerFees: map[string]float64{"0": 1}},
		{CleanerFees: map[string]float64{"3": -5}},
	}
	for _, b := range bad {
		if _, err := b.parse(); err == nil {
			t.Errorf("%+v must be rejected", b)
		}
	}
}

func TestDecodeFees(t *testing.T) {
	m, err := decodeFees([]byte(`{"1": 250.5, "x": 9}`))
	if err != nil || m[1] != 250.5 || len(m) != 1 {
		t.Fatalf("decode: %v %v", m, err)
	}
	if m, err := decodeFees(nil); err != nil || len(m) != 0 {
		t.Fatal("empty fees must decode to an empty map")
	}
}
