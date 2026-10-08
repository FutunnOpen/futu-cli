package reporting

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultHost = "report.futunn.com"

	DefaultQueueSize    = 256
	DefaultRequestLimit = 2 * time.Second
	DefaultFlushTimeout = 500 * time.Millisecond

	reportPath     = "/agent_report/call"
	defaultOrderID = ""
	unknownToolID  = "unknown"

	uuidByteLen      = 16
	uuidVersionByte  = 6
	uuidVariantByte  = 8
	uuidVersionMask  = 0x40
	uuidVariantMask  = 0x80
	uuidClearHigh    = 0x0f
	uuidClearVariant = 0x3f
)

// Event mirrors agent_report_svr.ReportEvent.
type Event struct {
	Source          int32  `json:"source,omitempty"`
	AgentName       string `json:"agent_name,omitempty"`
	Platform        string `json:"platform,omitempty"`
	Command         string `json:"command,omitempty"`
	Method          string `json:"method,omitempty"`
	APIPath         string `json:"api_path,omitempty"`
	APIPathTemplate string `json:"api_path_template,omitempty"`
	ToolID          string `json:"tool_id,omitempty"`
	RequestQuery    string `json:"request_query,omitempty"`
	RequestBody     string `json:"request_body,omitempty"`
	ResponseBody    string `json:"response_body,omitempty"`
	StatusCode      int32  `json:"status_code,omitempty"`
	DurationMs      int32  `json:"duration_ms,omitempty"`
	OrderID         string `json:"order_id,omitempty"`
	TraceID         string `json:"trace_id,omitempty"`
	Error           string `json:"error,omitempty"`
	Version         string `json:"version,omitempty"`
	OS              string `json:"os,omitempty"`
	Arch            string `json:"arch,omitempty"`
	CreatedAt       int64  `json:"created_at,omitempty"`
}

type BuildInput struct {
	Source          string
	AgentName       string
	Platform        string
	Command         string
	Method          string
	APIPath         string
	APIPathTemplate string
	Query           string
	RequestBody     []byte
	ResponseBody    []byte
	StatusCode      int
	DurationMs      int64
	Error           string
	Version         string
}

func BuildEvent(input BuildInput) Event {
	method := strings.ToUpper(input.Method)
	route := MatchToolRoute(method, firstNonEmpty(input.APIPathTemplate, input.APIPath))
	if route.ID == "" {
		route = MatchToolRoute(method, input.APIPath)
	}

	toolID := unknownToolID
	pathTemplate := input.APIPathTemplate
	if route.ID != "" {
		toolID = route.ID
		pathTemplate = route.Path
	}

	orderID := ExtractOrderID(input.ResponseBody, input.RequestBody, input.APIPath, pathTemplate, input.Query)
	coarseRedact := ShouldCoarseRedact(toolID, firstNonEmpty(pathTemplate, input.APIPath))

	return Event{
		Source:          parseSource(input.Source),
		AgentName:       input.AgentName,
		Platform:        input.Platform,
		Command:         input.Command,
		Method:          method,
		APIPath:         input.APIPath,
		APIPathTemplate: pathTemplate,
		ToolID:          toolID,
		RequestQuery:    SanitizeQuery(input.Query, coarseRedact),
		RequestBody:     SanitizeBody(input.RequestBody, coarseRedact),
		ResponseBody:    SanitizeBody(input.ResponseBody, coarseRedact),
		StatusCode:      int32(input.StatusCode),
		DurationMs:      clampInt32(input.DurationMs),
		OrderID:         orderID,
		TraceID:         NewTraceID(),
		Error:           input.Error,
		Version:         input.Version,
		OS:              runtime.GOOS,
		Arch:            runtime.GOARCH,
		CreatedAt:       time.Now().UnixMicro(),
	}
}

func IsReportEndpoint(path string) bool {
	parsed, err := url.Parse(path)
	if err == nil {
		path = parsed.Path
	}
	return path == reportPath
}

func ReportURL(host string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		host = DefaultHost
	}
	host = strings.TrimRight(host, "/")
	if strings.HasPrefix(host, "http://") || strings.HasPrefix(host, "https://") {
		return host + reportPath
	}
	return "https://" + host + reportPath
}

func NewTraceID() string {
	var b [uuidByteLen]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	b[uuidVersionByte] = (b[uuidVersionByte] & uuidClearHigh) | uuidVersionMask
	b[uuidVariantByte] = (b[uuidVariantByte] & uuidClearVariant) | uuidVariantMask
	encoded := hex.EncodeToString(b[:])
	return encoded[0:8] + "-" + encoded[8:12] + "-" + encoded[12:16] + "-" + encoded[16:20] + "-" + encoded[20:32]
}

func parseSource(source string) int32 {
	n, err := strconv.ParseInt(strings.TrimSpace(source), 10, 32)
	if err != nil {
		return 0
	}
	return int32(n)
}

func clampInt32(value int64) int32 {
	const maxInt32 = int64(1<<31 - 1)
	if value > maxInt32 {
		return int32(maxInt32)
	}
	if value < 0 {
		return 0
	}
	return int32(value)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
