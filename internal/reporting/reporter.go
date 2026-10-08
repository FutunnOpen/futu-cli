package reporting

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

const (
	headerAuthorization = "Authorization"
	headerContentType   = "Content-Type"
	contentTypeJSON     = "application/json"
	bearerPrefix        = "Bearer "
)

type Reporter struct {
	enabled  bool
	endpoint string
	token    string
	client   *http.Client
	queue    chan Event
	flush    chan chan struct{}
	done     chan struct{}
	wg       sync.WaitGroup
	once     sync.Once
}

type Options struct {
	Enabled        bool
	Host           string
	Endpoint       string
	Token          string
	QueueSize      int
	RequestTimeout time.Duration
	HTTPClient     *http.Client
}

type reportToolCallRequest struct {
	Event Event `json:"event,omitempty"`
}

func NewReporter(opts Options) *Reporter {
	queueSize := opts.QueueSize
	if queueSize <= 0 {
		queueSize = DefaultQueueSize
	}
	timeout := opts.RequestTimeout
	if timeout <= 0 {
		timeout = DefaultRequestLimit
	}
	httpClient := opts.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: timeout}
	}
	endpoint := opts.Endpoint
	if endpoint == "" {
		endpoint = ReportURL(opts.Host)
	}
	enabled := opts.Enabled && endpoint != ""

	r := &Reporter{
		enabled:  enabled,
		endpoint: endpoint,
		token:    opts.Token,
		client:   httpClient,
		queue:    make(chan Event, queueSize),
		flush:    make(chan chan struct{}),
		done:     make(chan struct{}),
	}
	if r.enabled {
		r.wg.Add(1)
		go r.run()
	}
	return r
}

func (r *Reporter) Report(event Event) {
	if r == nil || !r.enabled {
		return
	}
	select {
	case r.queue <- event:
	default:
	}
}

func (r *Reporter) Flush(ctx context.Context) error {
	if r == nil || !r.enabled {
		return nil
	}
	ack := make(chan struct{})
	select {
	case r.flush <- ack:
	case <-ctx.Done():
		return ctx.Err()
	}
	select {
	case <-ack:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (r *Reporter) Close(ctx context.Context) error {
	if r == nil || !r.enabled {
		return nil
	}
	_ = r.Flush(ctx)
	r.once.Do(func() {
		close(r.done)
	})
	done := make(chan struct{})
	go func() {
		r.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (r *Reporter) run() {
	defer r.wg.Done()
	for {
		select {
		case event := <-r.queue:
			r.send(event)
		case ack := <-r.flush:
			r.drain()
			close(ack)
		case <-r.done:
			r.drain()
			return
		}
	}
}

func (r *Reporter) drain() {
	for {
		select {
		case event := <-r.queue:
			r.send(event)
		default:
			return
		}
	}
}

func (r *Reporter) send(event Event) {
	payload, err := json.Marshal(reportToolCallRequest{Event: event})
	if err != nil {
		return
	}
	req, err := http.NewRequest(http.MethodPost, r.endpoint, bytes.NewReader(payload))
	if err != nil {
		return
	}
	req.Header.Set(headerContentType, contentTypeJSON)
	if r.token != "" {
		req.Header.Set(headerAuthorization, bearerPrefix+r.token)
	}
	resp, err := r.client.Do(req)
	if err != nil {
		return
	}
	_ = resp.Body.Close()
}
