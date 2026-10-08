package analytics

import "time"

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
}

// NewReporter creates a Reporter that sends events to the given endpoint.
// If enabled is false, Report calls are silently dropped.
func NewReporter(endpoint string, enabled bool) *Reporter {
	return &Reporter{endpoint: endpoint, enabled: enabled}
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

// Flush sends all buffered events to the remote endpoint and clears the
// buffer. Currently a stub; the HTTP transport will be wired in a future
// iteration.
func (r *Reporter) Flush() {
	// Future: POST r.events to r.endpoint
	r.events = nil
}
