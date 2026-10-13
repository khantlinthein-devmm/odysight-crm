package reports

import "testing"

func f(v float64) *float64 { return &v }

func TestLaborCostByPayType(t *testing.T) {
	cases := []struct {
		rate  PayRate
		hours float64
		want  float64
	}{
		{PayRate{"hourly", 150, 8}, 4, 600},
		{PayRate{"daily", 800, 8}, 4, 400},
		{PayRate{"per_job", 500, 8}, 4, 500},
		{PayRate{"monthly", 15600, 8}, 8, 600},
		{PayRate{"daily", 800, 0}, 8, 800}, // missing standard hours falls back to 8
		{PayRate{"unknown", 100, 8}, 4, 0},
	}
	for _, c := range cases {
		if got := round2(laborCost(c.rate, c.hours)); got != c.want {
			t.Errorf("%+v %vh = %v, want %v", c.rate, c.hours, got, c.want)
		}
	}
}

func TestComputeProfit(t *testing.T) {
	team := []TeamMember{{1, "Arti", "primary"}, {2, "Zaw", "crew"}, {3, "Mya", "crew"}}
	rates := map[int64]PayRate{1: {"hourly", 150, 8}, 2: {"hourly", 100, 8}}
	p := computeProfit(9, f(3000), 240, team, rates, 180)
	if p.Labor != 1000 || p.Supplies != 180 || p.Cost != 1180 {
		t.Fatalf("labor/supplies/cost = %v/%v/%v", p.Labor, p.Supplies, p.Cost)
	}
	if p.Profit == nil || *p.Profit != 1820 || p.Margin == nil || *p.Margin != 60.67 {
		t.Fatalf("profit/margin = %v/%v", p.Profit, p.Margin)
	}
	if p.MissingRates != 1 || p.Team[2].HasRate || p.Team[0].Cost != 600 || p.Team[0].Hours != 4 {
		t.Fatalf("team lines wrong: %+v", p.Team)
	}
}

func TestComputeProfitWithoutPriceOrTeam(t *testing.T) {
	p := computeProfit(1, nil, 120, nil, nil, 50)
	if p.Profit != nil || p.Margin != nil || p.Cost != 50 || len(p.Team) != 0 {
		t.Fatalf("unexpected: %+v", p)
	}
	loss := computeProfit(2, f(500), 120, []TeamMember{{1, "A", "primary"}}, map[int64]PayRate{1: {"hourly", 300, 8}}, 0)
	if *loss.Profit != -100 || *loss.Margin != -20 {
		t.Fatalf("loss = %v / %v", *loss.Profit, *loss.Margin)
	}
}

func TestHideRatesKeepsTotals(t *testing.T) {
	p := computeProfit(1, f(1000), 60, []TeamMember{{1, "A", "primary"}}, map[int64]PayRate{1: {"hourly", 200, 8}}, 0)
	h := hideRates(p)
	if h.Team[0].Rate != 0 || h.Team[0].Cost != 0 || h.Team[0].PayType != "" {
		t.Fatalf("rates leaked: %+v", h.Team[0])
	}
	if h.Labor != 200 || *h.Profit != 800 || p.Team[0].Rate != 200 {
		t.Fatal("totals must stay and the original must not be mutated")
	}
}
