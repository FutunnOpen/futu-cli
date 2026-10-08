package reporting

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/FutunnOpen/futu-cli/internal/auth"
	"github.com/FutunnOpen/futu-cli/internal/config"
)

const (
	envReportingIntegration = "FUTU_REPORTING_INTEGRATION"
	integrationTimeout      = 10 * time.Second
)

type reportToolCallResponse struct {
	Code    int32  `json:"code"`
	Message string `json:"message"`
}

func TestIntegrationReportToolCall(t *testing.T) {
	if os.Getenv(envReportingIntegration) != "1" {
		t.Skipf("set %s=1 to run real reporting endpoint integration test", envReportingIntegration)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	token, err := auth.EnsureValid(cfg.ClientID, cfg.TokenFile)
	if err != nil {
		t.Fatalf("load OAuth token: %v", err)
	}

	event := BuildEvent(BuildInput{
		Source:          "1",
		AgentName:       "cli",
		Platform:        "futu",
		Command:         "integration reporting test",
		Method:          http.MethodGet,
		APIPath:         "/integration/reporting-test",
		APIPathTemplate: "/integration/reporting-test",
		StatusCode:      http.StatusOK,
		DurationMs:      1,
		Version:         "integration-test",
	})
	payload, err := json.Marshal(reportToolCallRequest{Event: event})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	client := &http.Client{Timeout: integrationTimeout}
	req, err := http.NewRequest(http.MethodPost, ReportURL(DefaultHost), bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set(headerContentType, contentTypeJSON)
	req.Header.Set(headerAuthorization, bearerPrefix+token)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("post reporting event: %v", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read reporting response: %v", err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		t.Fatalf("reporting endpoint status = %d, body = %s, want < 400", resp.StatusCode, string(respBody))
	}
	var reportResp reportToolCallResponse
	if len(respBody) > 0 {
		if err := json.Unmarshal(respBody, &reportResp); err != nil {
			t.Fatalf("decode reporting response body %q: %v", string(respBody), err)
		}
	}
	if reportResp.Code != 0 {
		t.Fatalf("reporting response code = %d, message = %q, want code 0", reportResp.Code, reportResp.Message)
	}
	if strings.Contains(strings.ToLower(reportResp.Message), "auth failed") {
		t.Fatalf("reporting response message = %q, want authenticated success", reportResp.Message)
	}
	t.Logf("reporting response code = %d, message = %q", reportResp.Code, reportResp.Message)
}
