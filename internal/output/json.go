package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// JSON indentation settings.
const (
	jsonPrefix = ""
	jsonIndent = "  "
)

// PrintJSON writes v as pretty-printed JSON to w.
func PrintJSON(w io.Writer, v any) error {
	value, err := cleanJSON(v)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, jsonPrefix, jsonIndent)
	if err != nil {
		return fmt.Errorf("marshaling JSON: %w", err)
	}
	_, err = fmt.Fprintln(w, string(data))
	if err != nil {
		return fmt.Errorf("writing JSON: %w", err)
	}
	return nil
}

// PrintJSONCompact writes v as compact JSON to w.
func PrintJSONCompact(w io.Writer, v any) error {
	value, err := cleanJSON(v)
	if err != nil {
		return err
	}
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshaling JSON: %w", err)
	}
	_, err = fmt.Fprintln(w, string(data))
	if err != nil {
		return fmt.Errorf("writing JSON: %w", err)
	}
	return nil
}

func cleanJSON(value any) (any, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshaling JSON: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var decoded any
	if err := decoder.Decode(&decoded); err != nil {
		return nil, fmt.Errorf("decoding JSON for cleanup: %w", err)
	}
	return cleanJSONValue(decoded), nil
}

func cleanJSONValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return cleanJSONObject(typed)
	case []any:
		for index := range typed {
			typed[index] = cleanJSONValue(typed[index])
		}
		return typed
	default:
		return value
	}
}

func cleanJSONObject(object map[string]any) map[string]any {
	for key, value := range object {
		cleaned := cleanJSONValue(value)
		if isEmptyJSONValue(cleaned) {
			delete(object, key)
			continue
		}
		object[key] = cleaned
	}
	return object
}

func isEmptyJSONValue(value any) bool {
	switch typed := value.(type) {
	case nil:
		return true
	case string:
		return len(typed) == 0
	case json.Number:
		return isZeroJSONNumber(typed.String())
	case []any:
		return len(typed) == 0
	case map[string]any:
		return len(typed) == 0
	default:
		return false
	}
}

func isZeroJSONNumber(value string) bool {
	unsigned := strings.TrimPrefix(value, "-")
	mantissa := strings.FieldsFunc(unsigned, func(character rune) bool {
		return character == 'e' || character == 'E'
	})[0]
	digits := strings.ReplaceAll(mantissa, ".", "")
	return len(strings.Trim(digits, "0")) == 0
}
