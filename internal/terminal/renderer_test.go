package terminal

import (
	"bytes"
	"strings"
	"testing"

	"github.com/fatih/color"
)

func renderLines(lines ...string) string {
	var buf bytes.Buffer
	r := NewRenderer(&buf)
	r.Write(strings.Join(lines, "\n"))
	r.Flush()
	return buf.String()
}

func withNoColor(t *testing.T, fn func()) {
	t.Helper()
	prev := color.NoColor
	color.NoColor = true
	t.Cleanup(func() {
		color.NoColor = prev
	})
	fn()
}

func TestRenderer_TableAndRule(t *testing.T) {
	withNoColor(t, func() {
		output := renderLines(
			"| A | B |",
			"|---|---|",
			"| 1 | 2 |",
			"---",
		)

		if !strings.Contains(output, "+---+---+") {
			t.Fatalf("expected table border, got:\n%s", output)
		}
		if !strings.Contains(output, "| A | B |") {
			t.Fatalf("expected header row, got:\n%s", output)
		}
		if !strings.Contains(output, "────────") {
			t.Fatalf("expected horizontal rule, got:\n%s", output)
		}
	})
}

func TestRenderer_ListsAndCheckboxes(t *testing.T) {
	withNoColor(t, func() {
		output := renderLines(
			"- [ ] todo item",
			"- [x] done item",
			"1. [ ] numbered todo",
		)

		if !strings.Contains(output, "• [ ] todo item") {
			t.Fatalf("expected unchecked checkbox list item, got:\n%s", output)
		}
		if !strings.Contains(output, "• [x] done item") {
			t.Fatalf("expected checked checkbox list item, got:\n%s", output)
		}
		if !strings.Contains(output, "1. [ ] numbered todo") {
			t.Fatalf("expected numbered checkbox list item, got:\n%s", output)
		}
	})
}

func TestRenderer_BlockQuoteAndInline(t *testing.T) {
	withNoColor(t, func() {
		output := renderLines(
			"> quoted line",
			"Use **bold** and `code` here.",
			"• **합계**: 2,800Gi",
		)

		if !strings.Contains(output, "| quoted line") {
			t.Fatalf("expected block quote, got:\n%s", output)
		}
		if strings.Contains(output, "**") || strings.Contains(output, "`") {
			t.Fatalf("expected inline markers to be stripped, got:\n%s", output)
		}
		if !strings.Contains(output, "bold") || !strings.Contains(output, "code") {
			t.Fatalf("expected inline content, got:\n%s", output)
		}
		if !strings.Contains(output, "합계") {
			t.Fatalf("expected unicode inline content, got:\n%s", output)
		}
	})
}

func TestRenderer_LinksAndImages(t *testing.T) {
	withNoColor(t, func() {
		output := renderLines(
			"See [docs](http://example.com) and ![logo](http://img).",
		)

		if !strings.Contains(output, "docs (http://example.com)") {
			t.Fatalf("expected link rendering, got:\n%s", output)
		}
		if !strings.Contains(output, "img: logo (http://img)") {
			t.Fatalf("expected image rendering, got:\n%s", output)
		}
	})
}

func TestRenderer_CodeBlockLanguage(t *testing.T) {
	withNoColor(t, func() {
		output := renderLines(
			"```go",
			`fmt.Println("hi")`,
			"```",
		)

		if !strings.Contains(output, "```go") {
			t.Fatalf("expected code fence with language, got:\n%s", output)
		}
		if !strings.Contains(output, `fmt.Println("hi")`) {
			t.Fatalf("expected code line, got:\n%s", output)
		}
	})
}
