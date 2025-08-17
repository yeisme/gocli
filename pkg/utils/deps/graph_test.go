package deps

import (
	"strings"
	"testing"
)

func TestParseGoModGraph(t *testing.T) {
	input := `github.com/yeisme/gocli github.com/aymanbagabas/go-osc52/v2@v2.0.1
github.com/yeisme/gocli github.com/charmbracelet/colorprofile@v0.3.2
github.com/yeisme/gocli github.com/charmbracelet/lipgloss@v1.1.0
github.com/yeisme/gocli github.com/charmbracelet/x/ansi@v0.10.1
github.com/charmbracelet/colorprofile@v0.3.2 github.com/charmbracelet/x/ansi@v0.10.1
`
	g, err := ParseGoModGraph(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !g.Has("github.com/yeisme/gocli") {
		t.Fatalf("root not found")
	}

	for _, m := range g.Modules() {
		mID := m.ID()
		t.Logf("Module ID: %s", mID)
	}

	rootChildren := g.Children("github.com/yeisme/gocli")
	rootParent := g.Parents("github.com/yeisme/gocli")
	if len(rootChildren) != 4 {
		t.Fatalf("expected 4 children, got %d", len(rootChildren))
	}
	if len(rootParent) != 0 {
		t.Fatalf("expected no parent, got %q", rootParent)
	}

	secondChildren := g.Children("github.com/charmbracelet/colorprofile@v0.3.2")
	secondParent := g.Parents("github.com/charmbracelet/colorprofile@v0.3.2")
	if len(secondChildren) != 1 {
		t.Fatalf("expected 1 child, got %d", len(secondChildren))
	}
	if len(secondParent) != 1 {
		t.Fatalf("expected 1 parent, got %q", secondParent)
	}
}

func TestParseModuleToken(t *testing.T) {
	cases := []struct {
		in   string
		path string
		ver  string
	}{
		{"github.com/yeisme/gocli", "github.com/yeisme/gocli", ""},
		{"github.com/charmbracelet/lipgloss@v1.1.0", "github.com/charmbracelet/lipgloss", "v1.1.0"},
		{"example.com/a@b@v1", "example.com/a@b", "v1"},
	}
	for _, c := range cases {
		m := parseModuleToken(c.in)
		if m.Path != c.path || m.Version != c.ver {
			t.Fatalf("parse failed for %q: got %q %q", c.in, m.Path, m.Version)
		}
	}

	// roundtrip ID
	m := parseModuleToken("github.com/x/y@v2.3.4")
	if got := m.ID(); got != "github.com/x/y@v2.3.4" {
		t.Fatalf("id mismatch: %s", got)
	}

	if !strings.Contains(m.DocURL(), "pkg.go.dev/github.com/x/y@v2.3.4") {
		t.Fatalf("doc url unexpected: %s", m.DocURL())
	}
}
