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
