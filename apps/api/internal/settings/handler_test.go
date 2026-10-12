package settings

import (
	"encoding/json"
	"testing"
)

func TestWithoutPricesKeepsNamesAndDurations(t *testing.T) {
	all := map[string]json.RawMessage{
		KeyServices: json.RawMessage(`[{"id":"deep","name":"Deep Cleaning","durationMinutes":240,"basePrice":2500,"pricePerSqm":35,"active":true}]`),
		KeyCompany:  json.RawMessage(`{"name":"Smile Clean"}`),
	}
	out := withoutPrices(all)

	var items []ServiceItem
	if err := json.Unmarshal(out[KeyServices], &items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Name != "Deep Cleaning" || items[0].DurationMinutes != 240 {
		t.Fatalf("service details lost: %+v", items)
	}
	if items[0].BasePrice != 0 || items[0].PricePerSqm != 0 {
		t.Fatalf("prices not hidden: %+v", items[0])
	}
	if string(out[KeyCompany]) != `{"name":"Smile Clean"}` {
		t.Fatal("other settings must be untouched")
	}
}

func TestWithoutPricesDropsUnreadableCatalog(t *testing.T) {
	out := withoutPrices(map[string]json.RawMessage{KeyServices: json.RawMessage(`not json`)})
	if _, ok := out[KeyServices]; ok {
		t.Fatal("an unreadable catalog must not be passed through")
	}
}
