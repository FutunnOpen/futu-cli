package cmd

import (
	"encoding/json"
	"testing"
)

func TestDecodeJSONValueKeepsLargeInt(t *testing.T) {
	raw := json.RawMessage(`{"account_id":9007199254740993}`)
	value, ok := decodeJSONValue(raw).(map[string]any)
	if !ok {
		t.Fatal("expected object")
	}
	if got := jsonString(value, "account_id"); got != "9007199254740993" {
		t.Fatalf("account_id=%s", got)
	}
}
