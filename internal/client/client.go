package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"

	apierr "github.com/FutunnOpen/futu-cli/internal/errors"
	"github.com/FutunnOpen/futu-cli/internal/reporting"
)

// ── Constants ────────────────────────────────────────────────────────────────

const (
	defaultRequestTimeout = 30 * time.Second
	rateLimit             = 10 // requests per second
	rateBurst             = 10

	maxRetries      = 3
	baseBackoff     = 500 * time.Millisecond
	backoffMultiply = 2

	headerUserAgent    = "User-Agent"
	headerClientName   = "X-Client-Name"
	headerCliCmd       = "X-Cli-Cmd"
	headerReportSource = "X-Fhl-Report-Source"
	headerAuth         = "Authorization"
	headerContentType  = "Content-Type"

	contentTypeJSON = "application/json"
)

// contextKey is an unexported type used for context value keys to avoid collisions.
type contextKey int

const (
	// CtxKeyCommand stores the current CLI command name in the context.
	CtxKeyCommand contextKey = iota
	ctxKeyReportSourceRequired
	ctxKeyPathTemplate
)

// ── Client ───────────────────────────────────────────────────────────────────

// Client is the single HTTP client for all Futu API calls.
type Client struct {
	base         string
	http         *http.Client
	token        string
	clientName   string
	reportSource string
	platform     string
	agentName    string
	reporter     requestReporter
	limiter      *rate.Limiter
	version      string
	mu           sync.RWMutex // guards token
}

type requestReporter interface {
	Report(reporting.Event)
}

// CallOpts carries optional parameters for the Call method.
type CallOpts struct {
	Params     url.Values
	Body       any
	PathParams map[string]string // e.g. {id} substitution
}

// ── Options ──────────────────────────────────────────────────────────────────

// ClientOption is a functional option for constructing a Client.
type ClientOption func(*Client)

// WithToken sets the bearer token used for authorization.
func WithToken(token string) ClientOption {
	return func(c *Client) { c.token = token }
}

// WithClientName sets the client identifier sent in the X-Client-Name header.
func WithClientName(name string) ClientOption {
	return func(c *Client) { c.clientName = name }
}

// WithVersion sets the CLI version reported in the User-Agent header.
func WithVersion(ver string) ClientOption {
	return func(c *Client) { c.version = ver }
}

// WithReportSource sets the value sent in X-Fhl-Report-Source for trade mutations.
func WithReportSource(source string) ClientOption {
	return func(c *Client) { c.reportSource = source }
}

func WithReportMetadata(platform, agentName string) ClientOption {
	return func(c *Client) {
		c.platform = platform
		c.agentName = agentName
	}
}

func WithRequestReporter(reporter requestReporter) ClientOption {
	return func(c *Client) { c.reporter = reporter }
}

// ── Constructor ──────────────────────────────────────────────────────────────

// New creates a Client pointed at the given base URL.
func New(base string, opts ...ClientOption) *Client {
	c := &Client{
		base:    strings.TrimRight(base, "/"),
		http:    &http.Client{Timeout: defaultRequestTimeout},
		limiter: rate.NewLimiter(rate.Limit(rateLimit), rateBurst),
		version: "dev",
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// ── Token management ─────────────────────────────────────────────────────────

// SetToken allows callers (e.g. the auth module) to inject or update the
// bearer token after the client has been created.
func (c *Client) SetToken(token string) {
	c.mu.Lock()
	c.token = token
	c.mu.Unlock()
}

// readToken returns the current token in a concurrency-safe way.
func (c *Client) readToken() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.token
}

// ── Convenience verbs ────────────────────────────────────────────────────────

// Get performs an HTTP GET request.
func (c *Client) Get(ctx context.Context, path string, params url.Values, result any) error {
	return c.Do(ctx, http.MethodGet, path, params, nil, result)
}

// Post performs an HTTP POST request with a JSON body.
func (c *Client) Post(ctx context.Context, path string, body any, result any) error {
	return c.Do(ctx, http.MethodPost, path, nil, body, result)
}

// ── Endpoint-based call ──────────────────────────────────────────────────────

// Call looks up an endpoint by name in the registry, resolves path parameters,
// and dispatches the request through Do.
func (c *Client) Call(ctx context.Context, endpoint string, opts CallOpts, result any) error {
	ep, ok := Endpoints[endpoint]
	if !ok {
		return fmt.Errorf("unknown endpoint: %s", endpoint)
	}
	path := resolvePathParams(ep.Path, opts.PathParams)
	if ep.ReportSource {
		ctx = context.WithValue(ctx, ctxKeyReportSourceRequired, true)
	}
	ctx = context.WithValue(ctx, ctxKeyPathTemplate, ep.Path)
	return c.Do(ctx, ep.Method, path, opts.Params, opts.Body, result)
}

// resolvePathParams replaces {key} placeholders in a path template with
// values from the provided map.
func resolvePathParams(path string, params map[string]string) string {
	for key, val := range params {
		path = strings.ReplaceAll(path, "{"+key+"}", url.PathEscape(val))
	}
	return path
}

// ── Core request pipeline ────────────────────────────────────────────────────

// Do is the single exit point for ALL requests. It constructs the request,
// signs it, injects headers, applies rate limiting, sends with retries on
// 5xx / timeout, parses the response, and maps errors.
func (c *Client) Do(ctx context.Context, method, path string, params url.Values, body any, result any) error {
	bodyBytes, err := marshalBody(body)
	if err != nil {
		return fmt.Errorf("marshal request body: %w", err)
	}

	queryString := ""
	if len(params) > 0 {
		queryString = params.Encode()
	}

	fullURL := c.buildURL(path, queryString)

	if err := c.limiter.Wait(ctx); err != nil {
		return fmt.Errorf("rate limiter: %w", err)
	}

	pathTemplate := pathTemplateFromCtx(ctx)
	return c.doWithRetry(ctx, method, fullURL, path, pathTemplate, queryString, bodyBytes, result)
}

// buildURL joins the base, path, and optional query string.
func (c *Client) buildURL(path, queryString string) string {
	u := c.base + path
	if len(queryString) > 0 {
		u += "?" + queryString
	}
	return u
}

// marshalBody JSON-encodes the body when non-nil.
func marshalBody(body any) ([]byte, error) {
	if body == nil {
		return nil, nil
	}
	return json.Marshal(body)
}

// ── Retry loop ───────────────────────────────────────────────────────────────

// doWithRetry executes the request up to maxRetries times for retryable errors.
func (c *Client) doWithRetry(ctx context.Context, method, fullURL, path, pathTemplate, query string, bodyBytes []byte, result any) error {
	var lastErr error
	backoff := baseBackoff

	for attempt := range maxRetries {
		lastErr = c.sendAndParse(ctx, method, fullURL, path, pathTemplate, query, bodyBytes, result)
		if lastErr == nil {
			return nil
		}
		if !isRetryable(lastErr) || attempt == maxRetries-1 {
			break
		}
		if err := sleep(ctx, backoff); err != nil {
			return lastErr
		}
		backoff *= backoffMultiply
	}
	return lastErr
}

// sleep pauses for the given duration or returns early if the context is cancelled.
func sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// isRetryable reports whether the error warrants a retry (5xx or timeout).
func isRetryable(err error) bool {
	cliErr, ok := err.(*apierr.CLIError)
	if !ok {
		return false
	}
	switch cliErr.Code {
	case apierr.CodeServerError,
		apierr.CodeGatewayError,
		apierr.CodeServiceUnavail,
		apierr.CodeTimeout:
		return true
	default:
		return false
	}
}

// ── Single-attempt send & parse ──────────────────────────────────────────────

// sendAndParse builds, sends, and parses one request attempt.
func (c *Client) sendAndParse(ctx context.Context, method, fullURL, path, pathTemplate, query string, bodyBytes []byte, result any) error {
	req, err := c.newRequest(ctx, method, fullURL, path, query, bodyBytes)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}

	start := time.Now()
	resp, err := c.http.Do(req)
	duration := time.Since(start).Milliseconds()
	if err != nil {
		wrapped := c.wrapTransportError(err)
		c.reportRequest(ctx, method, path, pathTemplate, query, bodyBytes, nil, 0, duration, wrapped)
		return wrapped
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		readErr := fmt.Errorf("read response body: %w", err)
		c.reportRequest(ctx, method, path, pathTemplate, query, bodyBytes, nil, resp.StatusCode, duration, readErr)
		return readErr
	}

	err = parseResponseBody(resp.StatusCode, respBody, result)
	c.reportRequest(ctx, method, path, pathTemplate, query, bodyBytes, respBody, resp.StatusCode, duration, err)
	return err
}

// wrapTransportError converts a Go transport error into a CLIError.
func (c *Client) wrapTransportError(err error) error {
	if isTimeoutErr(err) {
		return &apierr.CLIError{
			Code:    apierr.CodeTimeout,
			Message: fmt.Sprintf("request timed out: %v", err),
		}
	}
	return fmt.Errorf("http transport: %w", err)
}

// isTimeoutErr checks whether the error represents a timeout.
func isTimeoutErr(err error) bool {
	type timeouter interface{ Timeout() bool }
	if te, ok := err.(timeouter); ok {
		return te.Timeout()
	}
	return false
}

// ── Request construction ─────────────────────────────────────────────────────

// newRequest constructs an *http.Request with all required headers and signing.
func (c *Client) newRequest(ctx context.Context, method, fullURL, path, query string, bodyBytes []byte) (*http.Request, error) {
	var bodyReader io.Reader
	if len(bodyBytes) > 0 {
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return nil, err
	}

	c.setHeaders(req, ctx, bodyBytes)
	return req, nil
}

// setHeaders injects the OAuth2 bearer token and request metadata.
func (c *Client) setHeaders(req *http.Request, ctx context.Context, bodyBytes []byte) {
	req.Header.Set(headerUserAgent, "futu/"+c.version)
	req.Header.Set(headerClientName, c.clientName)
	req.Header.Set(headerCliCmd, commandFromCtx(ctx))
	c.setReportSourceHeader(req, ctx)

	if len(bodyBytes) > 0 {
		req.Header.Set(headerContentType, contentTypeJSON)
	}

	c.setOAuthHeaders(req)
}

func (c *Client) setReportSourceHeader(req *http.Request, ctx context.Context) {
	required, _ := ctx.Value(ctxKeyReportSourceRequired).(bool)
	if required && len(c.reportSource) > 0 {
		req.Header.Set(headerReportSource, c.reportSource)
	}
}

// setOAuthHeaders sets the Authorization header for OAuth mode.
func (c *Client) setOAuthHeaders(req *http.Request) {
	if token := c.readToken(); len(token) > 0 {
		req.Header.Set(headerAuth, "Bearer "+token)
	}
}

// commandFromCtx extracts the current CLI command name from the context, if set.
func commandFromCtx(ctx context.Context) string {
	cmd, _ := ctx.Value(CtxKeyCommand).(string)
	return cmd
}

func pathTemplateFromCtx(ctx context.Context) string {
	pathTemplate, _ := ctx.Value(ctxKeyPathTemplate).(string)
	return pathTemplate
}

func (c *Client) reportRequest(ctx context.Context, method, path, pathTemplate, query string, requestBody, responseBody []byte, statusCode int, durationMs int64, requestErr error) {
	if c.reporter == nil || reporting.IsReportEndpoint(path) {
		return
	}
	errorText := ""
	if requestErr != nil {
		errorText = requestErr.Error()
	}
	c.reporter.Report(reporting.BuildEvent(reporting.BuildInput{
		Source:          c.reportSource,
		AgentName:       c.agentName,
		Platform:        c.platform,
		Command:         commandFromCtx(ctx),
		Method:          method,
		APIPath:         path,
		APIPathTemplate: pathTemplate,
		Query:           query,
		RequestBody:     requestBody,
		ResponseBody:    responseBody,
		StatusCode:      statusCode,
		DurationMs:      durationMs,
		Error:           errorText,
		Version:         c.version,
	}))
}

// ── Response parsing ─────────────────────────────────────────────────────────

// parseResponseBody decodes the response body or converts it to a CLIError.
func parseResponseBody(statusCode int, respBody []byte, result any) error {
	if statusCode >= http.StatusBadRequest {
		return apierr.FromAPI(statusCode, respBody)
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}
