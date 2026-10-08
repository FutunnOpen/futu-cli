package cmd

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/spf13/cobra"
)

func TestInitSharedStateSkipsConfigForReadOnlyCommands(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	tests := []struct {
		name string
		cmd  *cobra.Command
	}{
		{name: "version", cmd: versionCmd},
		{name: "completion", cmd: completionCmd},
		{name: "update", cmd: updateCmd},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := initSharedState(test.cmd); err != nil {
				t.Fatalf("initSharedState() error = %v", err)
			}
			if _, err := os.Stat(filepath.Join(home, ".futu")); !os.IsNotExist(err) {
				t.Fatalf("config directory was created: %v", err)
			}
		})
	}
}

func TestFormatVersionUsesFutuBinaryName(t *testing.T) {
	if got := formatVersion(); len(got) < len("futu version") || got[:len("futu version")] != "futu version" {
		t.Fatalf("formatVersion() = %q, want futu binary name", got)
	}
}

func TestReportSourceHeaderValueForFutu(t *testing.T) {
	previous := reportSourceMode
	t.Cleanup(func() { reportSourceMode = previous })

	reportSourceMode = reportSourceModeCLI
	got, err := reportSourceHeaderValue()
	if err != nil {
		t.Fatalf("reportSourceHeaderValue() error = %v", err)
	}
	if got != reportSourceFutuCLI {
		t.Fatalf("CLI report source = %q, want %q", got, reportSourceFutuCLI)
	}

	reportSourceMode = reportSourceModeSkill
	got, err = reportSourceHeaderValue()
	if err != nil {
		t.Fatalf("reportSourceHeaderValue() error = %v", err)
	}
	if got != reportSourceFutuSkill {
		t.Fatalf("skill report source = %q, want %q", got, reportSourceFutuSkill)
	}
}

func TestReportSourceHeaderValueRejectsInvalidMode(t *testing.T) {
	previous := reportSourceMode
	t.Cleanup(func() { reportSourceMode = previous })

	reportSourceMode = "unknown"
	if _, err := reportSourceHeaderValue(); err == nil {
		t.Fatal("expected invalid report source error")
	}
}

func TestReportSourceFlagIsHiddenAndRenamed(t *testing.T) {
	if rootCmd.PersistentFlags().Lookup("source") != nil {
		t.Fatal("source must be handled as an internal argument, not a cobra flag")
	}
	if rootCmd.PersistentFlags().Lookup("agent") != nil {
		t.Fatal("agent must be handled as an internal argument, not a cobra flag")
	}
	if rootCmd.PersistentFlags().Lookup("report-source") != nil {
		t.Fatal("legacy report-source flag must not be registered")
	}
}

func TestEffectiveAgentName(t *testing.T) {
	previousSource := reportSourceMode
	previousAgent := agentName
	clearAgentEnv(t)
	t.Cleanup(func() {
		reportSourceMode = previousSource
		agentName = previousAgent
	})

	reportSourceMode = ""
	agentName = ""
	if got := effectiveAgentName(); got != agentNameCLI {
		t.Fatalf("default agent = %q, want %q", got, agentNameCLI)
	}

	reportSourceMode = reportSourceModeSkill
	agentName = ""
	if got := effectiveAgentName(); got != agentNameUnknown {
		t.Fatalf("skill agent = %q, want %q", got, agentNameUnknown)
	}

	t.Setenv(envCodexSessionID, "codex-session")
	if got := effectiveAgentName(); got != agentNameCodex {
		t.Fatalf("codex env agent = %q, want %q", got, agentNameCodex)
	}

	t.Setenv(envCodexSessionID, "")
	t.Setenv(envClaudeCode, "1")
	if got := effectiveAgentName(); got != agentNameClaudeCode {
		t.Fatalf("claude code env agent = %q, want %q", got, agentNameClaudeCode)
	}

	agentName = agentNameCodex
	if got := effectiveAgentName(); got != agentNameCodex {
		t.Fatalf("explicit agent = %q, want %q", got, agentNameCodex)
	}
}

func clearAgentEnv(t *testing.T) {
	t.Helper()
	for _, key := range codexAgentEnvKeys {
		t.Setenv(key, "")
	}
	for _, key := range claudeCodeAgentEnvKeys {
		t.Setenv(key, "")
	}
}

func TestExtractInternalInvocationArgs(t *testing.T) {
	commandArgs := []string{"order", "detail", "O1"}
	tests := []struct {
		name       string
		args       []string
		wantSource string
		wantAgent  string
	}{
		{name: "source split beginning", args: []string{"--source", "skill", "order", "detail", "O1"}, wantSource: reportSourceModeSkill},
		{name: "source split middle", args: []string{"order", "--source", "skill", "detail", "O1"}, wantSource: reportSourceModeSkill},
		{name: "source split end", args: []string{"order", "detail", "O1", "--source", "skill"}, wantSource: reportSourceModeSkill},
		{name: "source equals beginning", args: []string{"--source=skill", "order", "detail", "O1"}, wantSource: reportSourceModeSkill},
		{name: "source equals middle", args: []string{"order", "--source=skill", "detail", "O1"}, wantSource: reportSourceModeSkill},
		{name: "source equals end", args: []string{"order", "detail", "O1", "--source=skill"}, wantSource: reportSourceModeSkill},
		{name: "source and agent split end", args: []string{"order", "detail", "O1", "--source", "skill", "--agent", "codex"}, wantSource: reportSourceModeSkill, wantAgent: agentNameCodex},
		{name: "agent equals normalizes hyphen", args: []string{"order", "detail", "O1", "--agent=claude-code"}, wantAgent: agentNameClaudeCode},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			args, internal, err := extractInternalInvocationArgs(test.args)
			if err != nil {
				t.Fatalf("extractInternalInvocationArgs() error = %v", err)
			}
			if internal.source != test.wantSource {
				t.Fatalf("source = %q, want %q", internal.source, test.wantSource)
			}
			if internal.agent != test.wantAgent {
				t.Fatalf("agent = %q, want %q", internal.agent, test.wantAgent)
			}
			if !reflect.DeepEqual(args, commandArgs) {
				t.Fatalf("args = %#v, want %#v", args, commandArgs)
			}
		})
	}
}

func TestExtractInternalSourceArgWithoutMarker(t *testing.T) {
	input := []string{"order", "detail", "O1"}
	args, internal, err := extractInternalInvocationArgs(input)
	if err != nil {
		t.Fatalf("extractInternalInvocationArgs() error = %v", err)
	}
	if len(internal.source) != 0 {
		t.Fatalf("source = %q, want empty", internal.source)
	}
	if len(internal.agent) != 0 {
		t.Fatalf("agent = %q, want empty", internal.agent)
	}
	if !reflect.DeepEqual(args, input) {
		t.Fatalf("args = %#v, want %#v", args, input)
	}
}

func TestExtractInternalInvocationArgsRejectsMalformedMarker(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "missing value", args: []string{"version", "--source"}},
		{name: "empty equals value", args: []string{"version", "--source="}},
		{name: "missing value before flag", args: []string{"--source", "--format", "json", "version"}},
		{name: "invalid split value", args: []string{"--source", "unknown", "version"}},
		{name: "invalid equals value", args: []string{"version", "--source=unknown"}},
		{name: "duplicate split", args: []string{"--source", "skill", "version", "--source", "cli"}},
		{name: "duplicate equals", args: []string{"--source=skill", "version", "--source=cli"}},
		{name: "duplicate mixed", args: []string{"--source", "skill", "version", "--source=cli"}},
		{name: "missing agent value", args: []string{"version", "--agent"}},
		{name: "empty agent equals value", args: []string{"version", "--agent="}},
		{name: "missing agent value before flag", args: []string{"--agent", "--format", "json", "version"}},
		{name: "invalid agent", args: []string{"version", "--agent", "cursor"}},
		{name: "duplicate agent", args: []string{"--agent", "codex", "version", "--agent=claude_code"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, _, err := extractInternalInvocationArgs(test.args); err == nil {
				t.Fatal("expected internal marker error")
			}
		})
	}
}
