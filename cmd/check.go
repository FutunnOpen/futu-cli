package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/FutunnOpen/futu-cli/internal/auth"
	"github.com/FutunnOpen/futu-cli/internal/output"
)

const (
	connectivityTimeout             = 5 * time.Second
	tokenValid                      = "valid"
	tokenExpired                    = "expired"
	tokenMissing                    = "missing"
	tokenInvalid                    = "invalid"
	tokenUnknown                    = "unknown"
	checkLabelToken                 = "Token"
	checkLabelDetail                = "Detail"
	checkLabelAPI                   = "CN API"
	tokenDetailPermissionAvailable  = "local quote or trade permission available"
	tokenDetailNoPermission         = "no local quote or trade permission"
	tokenDetailRefreshFailurePrefix = "refresh failed"
)

type endpointCheck struct {
	URL string `json:"url"`
	OK  bool   `json:"ok"`
	MS  int64  `json:"ms"`
}

type sessionCheck struct {
	Token  string `json:"token"`
	Detail string `json:"detail,omitempty"`
}

type connectivityCheck struct {
	API endpointCheck `json:"api"`
}

type checkResult struct {
	Session      sessionCheck      `json:"session"`
	Connectivity connectivityCheck `json:"connectivity"`
}

var checkCmd = &cobra.Command{
	Use: "check", Short: "验证 Token 有效性与 API 连通性", RunE: runCheck,
}

func init() {
	rootCmd.AddCommand(checkCmd)
}

func runCheck(_ *cobra.Command, _ []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), connectivityTimeout)
	defer cancel()
	client := diagnosticHTTPClient()
	apiBase := strings.TrimRight(cfg.APIBase, "/")
	connectivityCh := probeEndpointAsync(ctx, client, apiBase)
	result := checkResult{
		Session:      checkToken(ctx, client),
		Connectivity: connectivityCheck{API: <-connectivityCh},
	}
	printResult(result, func() { renderCheckTable(result) })
	return nil
}

func diagnosticHTTPClient() *http.Client {
	return &http.Client{
		Timeout: connectivityTimeout,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func probeEndpointAsync(
	ctx context.Context, client *http.Client, endpoint string,
) <-chan endpointCheck {
	result := make(chan endpointCheck, 1)
	go func() { result <- probeEndpoint(ctx, client, endpoint) }()
	return result
}

func probeEndpoint(ctx context.Context, client *http.Client, endpoint string) endpointCheck {
	startedAt := time.Now()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return endpointCheck{URL: endpoint}
	}
	response, err := client.Do(request)
	elapsed := time.Since(startedAt).Milliseconds()
	if err != nil {
		return endpointCheck{URL: endpoint, MS: elapsed}
	}
	response.Body.Close()
	return endpointCheck{URL: endpoint, OK: true, MS: elapsed}
}

func checkToken(ctx context.Context, client *http.Client) sessionCheck {
	store, err := auth.LoadToken(effectiveTokenFile())
	if err != nil {
		return sessionCheck{Token: tokenMissing, Detail: err.Error()}
	}
	if store.IsExpired() {
		store, err = refreshTokenIfExpired(effectiveTokenFile(), store)
		if err != nil {
			return sessionCheck{Token: tokenExpired, Detail: fmt.Sprintf("%s: %v", tokenDetailRefreshFailurePrefix, err)}
		}
	}
	store.EnrichMetadata()
	return tokenStatusFromPermissions(store.Permissions)
}

func tokenStatusFromPermissions(permissions auth.PermissionInfo) sessionCheck {
	if permissions.Quote || permissions.Trade {
		return sessionCheck{Token: tokenValid, Detail: tokenDetailPermissionAvailable}
	}
	return sessionCheck{Token: tokenUnknown, Detail: tokenDetailNoPermission}
}

func renderCheckTable(result checkResult) {
	tw := output.NewTableWriter(os.Stdout, statusHeaderKey, statusHeaderValue)
	tw.AddRow(checkLabelToken, result.Session.Token)
	addStatusRow(tw, checkLabelDetail, result.Session.Detail)
	tw.AddRow(checkLabelAPI, formatEndpointCheck(result.Connectivity.API))
	tw.Render()
}

func formatEndpointCheck(result endpointCheck) string {
	if !result.OK {
		return fmt.Sprintf("unreachable (%d ms) %s", result.MS, result.URL)
	}
	return fmt.Sprintf("ok (%d ms) %s", result.MS, result.URL)
}
