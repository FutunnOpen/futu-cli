package analytics

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"
)

const (
	contentTypeJSON = "application/json"
	flushTimeout    = 2 * time.Second
)

// Event represents a single telemetry event to report.
type Event struct {
	Command    string    `json:"command"`
	Outcome    string    `json:"outcome"`
	DurationMs int64     `json:"duration_ms"`
	Version    string    `json:"version"`
	OS         string    `json:"os"`
	Arch       string    `json:"arch"`
	AgentName  string    `json:"agent_name"`
	Timestamp  time.Time `json:"timestamp"`
}

// Reporter collects telemetry events and flushes them to a remote endpoint.
type Reporter struct {
	endpoint string
	enabled  bool
	events   []Event
	client   *http.Client
}

type flushRequest struct {
	Events []Event `json:"events"`
}

// NewReporter creates a Reporter that sends events to the given endpoint.
// If enabled is false, Report calls are silently dropped.
func NewReporter(endpoint string, enabled bool) *Reporter {
	return &Reporter{
		endpoint: endpoint,
		enabled:  enabled,
		client:   &http.Client{Timeout: flushTimeout},
	}
}

// Report records a telemetry event. Events are silently dropped when the
// reporter is disabled.
func (r *Reporter) Report(e Event) {
	if !r.enabled {
		return
	}
	e.Timestamp = time.Now()
	r.events = append(r.events, e)
}

// Flush sends all buffered events to the remote endpoint and clears the buffer.
// Transport errors are intentionally ignored so telemetry never breaks CLI use.
func (r *Reporter) Flush() {
	if !r.enabled || len(r.events) == 0 || r.endpoint == "" {
		r.events = nil
		return
	}
	payload, err := json.Marshal(flushRequest{Events: r.events})
	r.events = nil
	if err != nil {
		return
	}
	req, err := http.NewRequest(http.MethodPost, r.endpoint, bytes.NewReader(payload))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", contentTypeJSON)
	resp, err := r.client.Do(req)
	if err != nil {
		return
	}
	_ = resp.Body.Close()
}
