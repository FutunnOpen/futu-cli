package reporting

import (
	"encoding/json"
	"net/url"
	"strings"
)

const (
	redactedValue = "***"
	emptyJSON     = "{}"
)

var sensitiveKeyParts = []string{
	"access_token",
	"refresh_token",
	"token",
	"authorization",
	"password",
	"secret",
	"client_secret",
	"private_key",
}

func SanitizeQuery(query string, coarse bool) string {
	if query == "" {
		return emptyJSON
	}
	values, err := url.ParseQuery(query)
	if err != nil {
		return jsonString(query)
	}
	payload := make(map[string]any, len(values))
	for key, items := range values {
		if len(items) == 1 {
			payload[key] = items[0]
			continue
		}
		copied := make([]string, len(items))
		copy(copied, items)
		payload[key] = copied
	}
	return marshalSanitized(payload, coarse)
}

func SanitizeBody(body []byte, coarse bool) string {
	if len(body) == 0 {
		return emptyJSON
	}
	var parsed any
	if err := json.Unmarshal(body, &parsed); err != nil {
		if coarse {
			return jsonString(redactedValue)
		}
		return jsonString(string(body))
	}
	return marshalSanitized(parsed, coarse)
}

func marshalSanitized(value any, coarse bool) string {
	sanitized := sanitizeValue(value, "", coarse)
	encoded, err := json.Marshal(sanitized)
	if err != nil {
		return emptyJSON
	}
	return string(encoded)
}

func sanitizeValue(value any, key string, coarse bool) any {
	if IsOrderIDKey(key) {
		return value
	}
	if shouldRedactKey(key) || (coarse && key != "") {
		return redactShape(value)
	}
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for childKey, childValue := range typed {
			out[childKey] = sanitizeValue(childValue, childKey, coarse)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for index, item := range typed {
			out[index] = sanitizeValue(item, "", coarse)
		}
		return out
	default:
		if coarse {
			return redactedValue
		}
		return typed
	}
}

func redactShape(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, childValue := range typed {
			out[key] = sanitizeValue(childValue, key, true)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for index, item := range typed {
			out[index] = redactShape(item)
		}
		return out
	default:
		return redactedValue
	}
}

func shouldRedactKey(key string) bool {
	normalized := strings.ToLower(key)
	for _, part := range sensitiveKeyParts {
		if strings.Contains(normalized, part) {
			return true
		}
	}
	return false
}

func jsonString(value string) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return `""`
	}
	return string(encoded)
}
