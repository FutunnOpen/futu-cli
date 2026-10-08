package output

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestPrintJSONOmitsEmptyObjectFields(t *testing.T) {
	input := map[string]any{
		"empty_string": "",
		"nil_value":    nil,
		"zero_integer": 0,
		"zero_float":   0.0,
		"empty_list":   []string{},
		"empty_object": map[string]any{"empty": ""},
		"false_value":  false,
		"name":         "Futu",
		"nested":       map[string]any{"empty": "", "count": 2},
	}
	var output bytes.Buffer
	if err := PrintJSON(&output, input); err != nil {
		t.Fatalf("PrintJSON() error = %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(output.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	for _, key := range []string{
		"empty_string", "nil_value", "zero_integer", "zero_float", "empty_list", "empty_object",
	} {
		if _, exists := result[key]; exists {
			t.Errorf("empty field %q was not omitted", key)
		}
	}
	if result["false_value"] != false || result["name"] != "Futu" {
		t.Fatalf("non-empty fields = %#v", result)
	}
	nested, ok := result["nested"].(map[string]any)
	if !ok || len(nested) != 1 || nested["count"] != float64(2) {
		t.Fatalf("nested = %#v", result["nested"])
	}
}

func TestPrintJSONPreservesArrayPositionsAndTopLevelEmptyList(t *testing.T) {
	var output bytes.Buffer
	input := []any{0, "", nil, map[string]any{"empty": ""}}
	if err := PrintJSONCompact(&output, input); err != nil {
		t.Fatalf("PrintJSONCompact() error = %v", err)
	}
	if got, want := output.String(), `[0,"",null,{}]`+"\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}

	output.Reset()
	if err := PrintJSON(&output, []string{}); err != nil {
		t.Fatalf("PrintJSON() error = %v", err)
	}
	if got, want := output.String(), "[]\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestIsZeroJSONNumber(t *testing.T) {
	zeroValues := []string{"0", "-0", "0.0", "0e10", "-0.000E-2"}
	for _, value := range zeroValues {
		if !isZeroJSONNumber(value) {
			t.Errorf("isZeroJSONNumber(%q) = false", value)
		}
	}
	for _, value := range []string{"1", "-1", "0.01", "1e-10"} {
		if isZeroJSONNumber(value) {
			t.Errorf("isZeroJSONNumber(%q) = true", value)
		}
	}
}
