package errors

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// CLI error code constants.
const (
	CodeAuthInvalid    = "AUTH_INVALID"
	CodeAuthExpired    = "AUTH_EXPIRED"
	CodePermission     = "PERMISSION_DENIED"
	CodeNotFound       = "NOT_FOUND"
	CodeRateLimit      = "RATE_LIMIT"
	CodeBadRequest     = "BAD_REQUEST"
	CodeConflict       = "CONFLICT"
	CodeServerError    = "SERVER_ERROR"
	CodeGatewayError   = "GATEWAY_ERROR"
	CodeServiceUnavail = "SERVICE_UNAVAILABLE"
	CodeTimeout        = "TIMEOUT"
	CodeUnknown        = "UNKNOWN"
)

// CLIError represents a structured CLI error with a machine-readable code,
// a human-readable message, and an optional hint for resolution.
type CLIError struct {
	Code    string
	Message string
	Hint    string
}

// Error implements the error interface.
func (e *CLIError) Error() string {
	base := fmt.Sprintf("[%s] %s", e.Code, e.Message)
	if e.Hint != "" {
		base += "\nHint: " + e.Hint
	}
	return base
}

// apiErrorResponse represents the JSON error body returned by the Futu API.
type apiErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// FromAPI maps a server HTTP response to a CLIError.
func FromAPI(statusCode int, body []byte) error {
	apiErr := parseAPIBody(body)
	code := mapHTTPStatus(statusCode)
	if apiErr.Code != 0 {
		code = mapAPICode(apiErr.Code, code)
	}
	msg := buildMessage(statusCode, apiErr)
	return &CLIError{
		Code:    code,
		Message: msg,
		Hint:    hintFor(code),
	}
}

// parseAPIBody attempts to extract a structured error from the response body.
func parseAPIBody(body []byte) apiErrorResponse {
	var resp apiErrorResponse
	if len(body) > 0 {
		_ = json.Unmarshal(body, &resp)
	}
	return resp
}

// buildMessage constructs a human-readable error message.
func buildMessage(statusCode int, apiErr apiErrorResponse) string {
	if apiErr.Message != "" {
		return apiErr.Message
	}
	return fmt.Sprintf("API returned HTTP %d", statusCode)
}

// mapHTTPStatus maps an HTTP status code to a CLI error code.
func mapHTTPStatus(statusCode int) string {
	switch statusCode {
	case http.StatusUnauthorized:
		return CodeAuthInvalid
	case http.StatusForbidden:
		return CodePermission
	case http.StatusNotFound:
		return CodeNotFound
	case http.StatusTooManyRequests:
		return CodeRateLimit
	case http.StatusBadRequest:
		return CodeBadRequest
	case http.StatusConflict:
		return CodeConflict
	case http.StatusRequestTimeout, http.StatusGatewayTimeout:
		return CodeTimeout
	case http.StatusBadGateway:
		return CodeGatewayError
	case http.StatusServiceUnavailable:
		return CodeServiceUnavail
	case http.StatusInternalServerError:
		return CodeServerError
	default:
		return CodeUnknown
	}
}

// API error code range constants.
const (
	apiCodeAuthStart       = 1000
	apiCodeAuthEnd         = 1099
	apiCodePermissionStart = 1100
	apiCodePermissionEnd   = 1199
	apiCodeNotFoundStart   = 2000
	apiCodeNotFoundEnd     = 2099
	apiCodeRateLimitStart  = 3000
	apiCodeRateLimitEnd    = 3099
	apiCodeTokenExpired    = 1001
)

// mapAPICode refines the CLI error code using the API-specific error code.
func mapAPICode(apiCode int, fallback string) string {
	switch {
	case apiCode == apiCodeTokenExpired:
		return CodeAuthExpired
	case apiCode >= apiCodeAuthStart && apiCode <= apiCodeAuthEnd:
		return CodeAuthInvalid
	case apiCode >= apiCodePermissionStart && apiCode <= apiCodePermissionEnd:
		return CodePermission
	case apiCode >= apiCodeNotFoundStart && apiCode <= apiCodeNotFoundEnd:
		return CodeNotFound
	case apiCode >= apiCodeRateLimitStart && apiCode <= apiCodeRateLimitEnd:
		return CodeRateLimit
	default:
		return fallback
	}
}

// hintFor provides user-friendly hints for common error codes.
func hintFor(code string) string {
	hints := map[string]string{
		CodeAuthInvalid:    "Run 'futu auth login' to authenticate.",
		CodeAuthExpired:    "Your session has expired. Run 'futu auth login' to re-authenticate.",
		CodePermission:     "Check that your account has the required permissions for this operation.",
		CodeNotFound:       "Verify the resource identifier and try again.",
		CodeRateLimit:      "Too many requests. Wait a moment and retry.",
		CodeBadRequest:     "Check your input parameters and try again.",
		CodeConflict:       "The resource was modified concurrently. Refresh and retry.",
		CodeServerError:    "Server error. Try again later or contact support.",
		CodeGatewayError:   "Gateway error. The service may be temporarily unavailable.",
		CodeServiceUnavail: "Service is temporarily unavailable. Try again later.",
		CodeTimeout:        "Request timed out. Check your network and try again.",
	}
	return hints[code]
}
