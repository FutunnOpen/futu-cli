package service

import (
	"encoding/json"
	"testing"
)

func TestScreenRankItemScale(t *testing.T) {
	raw := []byte(`{
		"code": "HK.01638",
		"sc_name": "佳兆业集团",
		"results": [
			{"simple_property_result": {"property": {"name": 2220}, "value": "21000000"}},
			{"cumulative_property_result": {"property": {"name": 3102, "days": 1}, "value": "10526"}},
			{"cumulative_property_result": {"property": {"name": 3104, "days": 1}, "value": "474118400"}},
			{"cumulative_property_result": {"property": {"name": 3105, "days": 1}, "value": "9711544000"}}
		]
	}`)
	var item screenRankItem
	if err := json.Unmarshal(raw, &item); err != nil {
		t.Fatal(err)
	}
	got := item.toRankItem()
	if got.Symbol != "HK.01638" || got.Name != "佳兆业集团" {
		t.Fatalf("identity: %+v", got)
	}
	if got.LastPrice != 0.021 {
		t.Fatalf("price=%v", got.LastPrice)
	}
	if got.ChangePct != 10.526 {
		t.Fatalf("change=%v", got.ChangePct)
	}
	if got.Volume != 474118400 {
		t.Fatalf("volume=%d", got.Volume)
	}
	if got.Turnover != 9711544 {
		t.Fatalf("turnover=%v", got.Turnover)
	}
}

func TestRankSort(t *testing.T) {
	name, direction, err := rankSort("losers")
	if err != nil || name != rankPropChange || direction != rankSortAsc {
		t.Fatalf("losers: name=%d direction=%d err=%v", name, direction, err)
	}
	if _, _, err := rankSort("nope"); err == nil {
		t.Fatal("expected invalid type")
	}
}
