package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestTableWriterAlignsWideCharacters(t *testing.T) {
	var output bytes.Buffer
	table := NewTableWriter(&output, "Side", "Type", "Status")
	table.AddRow("BUY(Buy)", "LIMIT(Limit)", "FILLED_ALL(Filled)")
	table.AddRow("SELL_SHORT(Sell Short)", "MARKET(Market)", "FAILED(Failed)")

	table.Render()

	lines := strings.Split(strings.TrimRight(output.String(), "\n"), "\n")
	if len(lines) != 4 {
		t.Fatalf("line count = %d, want 4\n%s", len(lines), output.String())
	}
	separatorStart := displayWidth(lines[1][:strings.Index(lines[1], "  ")])
	firstRowStart := displayWidth(lines[2][:strings.Index(lines[2], "LIMIT")])
	secondRowStart := displayWidth(lines[3][:strings.Index(lines[3], "MARKET")])
	if separatorStart <= 0 || firstRowStart != separatorStart+columnPadding || secondRowStart != separatorStart+columnPadding {
		t.Fatalf("first column not aligned:\n%s", output.String())
	}
}

func TestDisplayWidthCountsWideCharacters(t *testing.T) {
	if got, want := displayWidth("BUY(Buy)"), 8; got != want {
		t.Fatalf("displayWidth() = %d, want %d", got, want)
	}
}
