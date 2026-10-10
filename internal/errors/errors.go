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
	CodeNoData         = "NO_DATA"
	CodeUnsupported    = "UNSUPPORTED"
	CodeInvalidSymbol  = "INVALID_SYMBOL"
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

// BFFError is the optional error object in a BFF envelope.
type BFFError struct {
	Code    string
	Message string
}

// BFF ret_code values from bff_gateway_svr envelope.
const (
	bffCodeOK               = 0
	bffCodeInvalidRequest   = -3
	bffCodeInvalidSymbol    = -7
	bffCodeUnsupported      = -8
	bffCodePermissionDenied = -9
	bffCodeNoData           = -10
	bffCodeRateLimited      = -11
)

// FromBFF maps a BFF JSON envelope (HTTP 200 + ret_code) to a CLIError.
func FromBFF(retCode int, retMsg string, bffErr *BFFError) error {
	if retCode == bffCodeOK {
		return nil
	}
	code := mapBFFRetCode(retCode)
	msg := retMsg
	if bffErr != nil {
		if len(bffErr.Message) > 0 {
			msg = bffErr.Message
		}
		if mapped := mapBFFErrCode(bffErr.Code); mapped != "" {
			code = mapped
		}
	}
	if len(msg) == 0 {
		msg = fmt.Sprintf("BFF error (ret_code %d)", retCode)
	}
	return &CLIError{Code: code, Message: msg, Hint: hintFor(code)}
}

func mapBFFRetCode(retCode int) string {
	switch retCode {
	case bffCodeInvalidRequest:
		return CodeBadRequest
	case bffCodeInvalidSymbol:
		return CodeInvalidSymbol
	case bffCodeUnsupported:
		return CodeUnsupported
	case bffCodePermissionDenied:
		return CodePermission
	case bffCodeNoData:
		return CodeNoData
	case bffCodeRateLimited:
		return CodeRateLimit
	default:
		if retCode < 0 {
			return CodeServerError
		}
		return CodeUnknown
	}
}

func mapBFFErrCode(errCode string) string {
	switch errCode {
	case "invalid_parameter":
		return CodeBadRequest
	case "invalid_symbol":
		return CodeInvalidSymbol
	case "unsupported":
		return CodeUnsupported
	case "permission_denied":
		return CodePermission
	case "no_data":
		return CodeNoData
	case "rate_limited":
		return CodeRateLimit
	default:
		return ""
	}
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
		CodeNoData:         "The symbol is valid but the API returned no data for this query.",
		CodeUnsupported:    "This market or instrument type is not supported by the API.",
		CodeInvalidSymbol:  "Verify the symbol format (MARKET.CODE, e.g. HK.00700) and try again.",
	}
	return hints[code]
}
