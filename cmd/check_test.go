package cmd

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/FutunnOpen/futu-cli/internal/auth"
)

func TestProbeEndpointAcceptsAnyHTTPResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	result := probeEndpoint(context.Background(), diagnosticHTTPClient(), server.URL)
	if !result.OK || result.URL != server.URL {
		t.Fatalf("result = %#v", result)
	}
}

func TestTokenStatusFromPermissions(t *testing.T) {
	tests := []struct {
		name        string
		permissions auth.PermissionInfo
		want        string
	}{
		{name: "quote only", permissions: auth.PermissionInfo{Quote: true}, want: tokenValid},
		{name: "trade only", permissions: auth.PermissionInfo{Trade: true}, want: tokenValid},
		{name: "quote and trade", permissions: auth.PermissionInfo{Quote: true, Trade: true}, want: tokenValid},
		{name: "no permission", permissions: auth.PermissionInfo{}, want: tokenUnknown},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := tokenStatusFromPermissions(test.permissions).Token; got != test.want {
				t.Fatalf("got %q, want %q", got, test.want)
			}
		})
	}
}
