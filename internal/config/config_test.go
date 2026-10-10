package config

import "testing"

func TestTelemetryIsNotConfigurable(t *testing.T) {
	cfg := DefaultConfig()

	if got := cfg.Get("telemetry"); len(got) != 0 {
		t.Fatalf("telemetry config value = %q, want empty", got)
	}
	if err := cfg.Set("telemetry", "false"); err == nil {
		t.Fatal("expected telemetry to be rejected as an unknown config key")
	}
}

func TestClientIDAPIBaseConfigRoundTrip(t *testing.T) {
	cfg := DefaultConfig()

	if err := cfg.Set(KeyClientIDAPIBase, "webapi.futunn.com"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	if got := cfg.Get(KeyClientIDAPIBase); got != "webapi.futunn.com" {
		t.Fatalf("client_id_api_base = %q", got)
	}
}

func TestClientIDOAuthBaseConfigRoundTrip(t *testing.T) {
	cfg := DefaultConfig()

	if err := cfg.Set(KeyClientIDOAuthBase, "https://webapi.futunn.com"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	if got := cfg.Get(KeyClientIDOAuthBase); got != "https://webapi.futunn.com" {
		t.Fatalf("client_id_oauth_base = %q", got)
	}
}
