package output

import (
	"fmt"
	"io"
	"strings"
	"unicode"
)

// Column padding between columns.
const columnPadding = 2

// TableWriter formats tabular data with aligned columns.
type TableWriter struct {
	headers []string
	rows    [][]string
	w       io.Writer
}

// NewTableWriter creates a new TableWriter that writes to w.
func NewTableWriter(w io.Writer, headers ...string) *TableWriter {
	return &TableWriter{
		headers: headers,
		rows:    make([][]string, 0),
		w:       w,
	}
}

// AddRow appends a row of values to the table.
func (t *TableWriter) AddRow(values ...string) {
	t.rows = append(t.rows, values)
}

// Render writes the formatted table to the writer.
func (t *TableWriter) Render() {
	widths := t.computeColumnWidths()
	t.renderHeader(widths)
	t.renderSeparator(widths)
	t.renderRows(widths)
}

// computeColumnWidths calculates the maximum width for each column.
func (t *TableWriter) computeColumnWidths() []int {
	colCount := len(t.headers)
	widths := make([]int, colCount)

	for i, h := range t.headers {
		widths[i] = displayWidth(h)
	}
	for _, row := range t.rows {
		for i := 0; i < colCount && i < len(row); i++ {
			width := displayWidth(row[i])
			if width > widths[i] {
				widths[i] = width
			}
		}
	}
	return widths
}

// renderHeader writes the header row.
func (t *TableWriter) renderHeader(widths []int) {
	t.renderLine(t.headers, widths)
}

// renderSeparator writes a dashed separator line beneath the header.
func (t *TableWriter) renderSeparator(widths []int) {
	parts := make([]string, len(widths))
	for i, w := range widths {
		parts[i] = strings.Repeat("-", w)
	}
	separator := strings.Join(parts, strings.Repeat(" ", columnPadding))
	fmt.Fprintln(t.w, separator)
}

// renderRows writes all data rows.
func (t *TableWriter) renderRows(widths []int) {
	for _, row := range t.rows {
		t.renderLine(row, widths)
	}
}

// renderLine writes a single line with columns padded to the given widths.
func (t *TableWriter) renderLine(values []string, widths []int) {
	parts := make([]string, len(widths))
	for i, w := range widths {
		val := ""
		if i < len(values) {
			val = values[i]
		}
		parts[i] = padRight(val, w)
	}
	line := strings.Join(parts, strings.Repeat(" ", columnPadding))
	fmt.Fprintln(t.w, strings.TrimRight(line, " "))
}

// padRight pads s with spaces on the right to reach the given width.
func padRight(s string, width int) string {
	displayWidth := displayWidth(s)
	if displayWidth >= width {
		return s
	}
	return s + strings.Repeat(" ", width-displayWidth)
}

func displayWidth(s string) int {
	width := 0
	for _, r := range s {
		width += runeDisplayWidth(r)
	}
	return width
}

func runeDisplayWidth(r rune) int {
	if r == 0 {
		return 0
	}
	if r < 32 || (r >= 0x7f && r < 0xa0) {
		return 0
	}
	if isWideRune(r) {
		return 2
	}
	return 1
}

func isWideRune(r rune) bool {
	return unicode.Is(unicode.Han, r) ||
		unicode.Is(unicode.Hangul, r) ||
		unicode.Is(unicode.Hiragana, r) ||
		unicode.Is(unicode.Katakana, r) ||
		(r >= 0x1100 && r <= 0x115f) ||
		(r >= 0x2329 && r <= 0x232a) ||
		(r >= 0x2e80 && r <= 0xa4cf) ||
		(r >= 0xac00 && r <= 0xd7a3) ||
		(r >= 0xf900 && r <= 0xfaff) ||
		(r >= 0xfe10 && r <= 0xfe19) ||
		(r >= 0xfe30 && r <= 0xfe6f) ||
		(r >= 0xff00 && r <= 0xff60) ||
		(r >= 0xffe0 && r <= 0xffe6)
}
