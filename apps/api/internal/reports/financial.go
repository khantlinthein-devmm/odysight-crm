package reports

import "time"

// RevenueBucket is one period's billed vs collected amounts.
type RevenueBucket struct {
	Period      string  `json:"period"`
	Billed      float64 `json:"billed"`
	Collected   float64 `json:"collected"`
	Outstanding float64 `json:"outstanding"`
}

// TaxSummary aggregates invoiced, tax-collected, and tax-outstanding amounts.
type TaxSummary struct {
	TaxRate        float64 `json:"taxRate"`
	TaxBilled      float64 `json:"taxBilled"`
	TaxCollected   float64 `json:"taxCollected"`
	TaxOutstanding float64 `json:"taxOutstanding"`
	BilledTotal    float64 `json:"billedTotal"`
	CollectedTotal float64 `json:"collectedTotal"`
}

// ARAgingBucket groups open invoices by days past due.
type ARAgingBucket struct {
	Label  string  `json:"label"`
	Amount float64 `json:"amount"`
	Count  int64   `json:"count"`
}

// FinancialReport is the output of the financial & tax report endpoint.
type FinancialReport struct {
	From     time.Time       `json:"from"`
	To       time.Time       `json:"to"`
	Revenue  []RevenueBucket `json:"revenue"`
	Tax      TaxSummary      `json:"taxSummary"`
	ARAging  []ARAgingBucket `json:"arAging"`
	Currency string          `json:"currency"`
}