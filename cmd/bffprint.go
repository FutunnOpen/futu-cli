package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/futunn/futu/internal/output"
	"github.com/futunn/futu/internal/service"
)

func printBFFResult(result *service.BFFResult, renderTable func()) {
	if isJSONOutput() {
		printBFFJSON(result)
		return
	}
	renderTable()
}

func printBFFJSON(result *service.BFFResult) {
	data := decodeJSONValue(result.Data)
	if len(result.Pagination) == 0 && len(result.Skipped) == 0 {
		_ = output.PrintJSON(os.Stdout, data)
		return
	}
	payload := map[string]any{"data": data}
	if value := decodeJSONValue(result.Pagination); value != nil {
		payload["pagination"] = value
	}
	if value := decodeJSONValue(result.Skipped); value != nil {
		payload["skipped"] = value
	}
	_ = output.PrintJSON(os.Stdout, payload)
}

func decodeJSONValue(raw json.RawMessage) any {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var value any
	if err := dec.Decode(&value); err != nil {
		return nil
	}
	return value
}

func jsonObjectList(data any, key string) []map[string]any {
	root, _ := data.(map[string]any)
	if root == nil {
		if list, ok := data.([]any); ok {
			return anyListToObjects(list)
		}
		return nil
	}
	list, _ := root[key].([]any)
	return anyListToObjects(list)
}

func anyListToObjects(list []any) []map[string]any {
	out := make([]map[string]any, 0, len(list))
	for _, item := range list {
		if object, ok := item.(map[string]any); ok {
			out = append(out, object)
		}
	}
	return out
}

func jsonString(object map[string]any, key string) string {
	if object == nil {
		return ""
	}
	switch typed := object[key].(type) {
	case string:
		return typed
	case json.Number:
		return typed.String()
	case float64:
		if typed == float64(int64(typed)) {
			return strconv.FormatInt(int64(typed), 10)
		}
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	case bool:
		return strconv.FormatBool(typed)
	default:
		if typed == nil {
			return ""
		}
		return fmt.Sprint(typed)
	}
}

func jsonFloat(object map[string]any, key string) float64 {
	if object == nil {
		return 0
	}
	switch typed := object[key].(type) {
	case float64:
		return typed
	case json.Number:
		value, _ := typed.Float64()
		return value
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	case string:
		value, _ := strconv.ParseFloat(typed, 64)
		return value
	default:
		return 0
	}
}

func jsonInt(object map[string]any, key string) int64 {
	if object == nil {
		return 0
	}
	switch typed := object[key].(type) {
	case float64:
		return int64(typed)
	case json.Number:
		value, _ := typed.Int64()
		return value
	case int:
		return int64(typed)
	case int64:
		return typed
	case string:
		value, _ := strconv.ParseInt(typed, 10, 64)
		return value
	default:
		return 0
	}
}

func formatPrice(v float64) string {
	return fmt.Sprintf("%.3f", v)
}
