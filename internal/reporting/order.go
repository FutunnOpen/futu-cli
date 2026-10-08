package reporting

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

var orderIDKeys = map[string]bool{
	"order_id": true,
	"orderid":  true,
	"order_no": true,
	"orderno":  true,
}

func ExtractOrderID(responseBody, requestBody []byte, path, pathTemplate, query string) string {
	for _, body := range [][]byte{responseBody, requestBody} {
		if value := extractOrderIDFromBody(body); value != "" {
			return value
		}
	}
	if value := extractOrderIDFromPath(path, pathTemplate); value != "" {
		return value
	}
	if value := extractOrderIDFromQuery(query); value != "" {
		return value
	}
	return defaultOrderID
}

func IsOrderIDKey(key string) bool {
	return orderIDKeys[strings.ToLower(key)]
}

func extractOrderIDFromBody(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	var parsed any
	if err := json.Unmarshal(body, &parsed); err != nil {
		return ""
	}
	return findOrderID(parsed, "")
}

func findOrderID(value any, key string) string {
	if IsOrderIDKey(key) {
		return scalarToString(value)
	}
	switch typed := value.(type) {
	case map[string]any:
		for childKey, childValue := range typed {
			if found := findOrderID(childValue, childKey); found != "" {
				return found
			}
		}
	case []any:
		for _, item := range typed {
			if found := findOrderID(item, ""); found != "" {
				return found
			}
		}
	}
	return ""
}

func extractOrderIDFromPath(path, pathTemplate string) string {
	pathParts := splitPath(path)
	templateParts := splitPath(pathTemplate)
	if len(pathParts) != len(templateParts) {
		return ""
	}
	for index, templatePart := range templateParts {
		if !isPlaceholder(templatePart) {
			continue
		}
		key := strings.Trim(templatePart, "{}")
		if !IsOrderIDKey(key) {
			continue
		}
		value, err := url.PathUnescape(pathParts[index])
		if err != nil {
			return pathParts[index]
		}
		return value
	}
	return ""
}

func extractOrderIDFromQuery(query string) string {
	values, err := url.ParseQuery(query)
	if err != nil {
		return ""
	}
	for key, items := range values {
		if !IsOrderIDKey(key) || len(items) == 0 {
			continue
		}
		if items[0] != "" {
			return items[0]
		}
	}
	return ""
}

func scalarToString(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case float64:
		return fmt.Sprintf("%.0f", typed)
	case bool:
		return fmt.Sprintf("%t", typed)
	default:
		return fmt.Sprintf("%v", typed)
	}
}
