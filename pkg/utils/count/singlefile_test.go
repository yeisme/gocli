package count

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// helper to write file
func writeFile(t *testing.T, dir, name, content string) string {
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	return path
}

func Test_detectLanguage(t *testing.T) {
	cases := map[string]string{"a.go": "Go", "b.py": "Python", "c.unknown": "Unknown"}
	for f, want := range cases {
		if got := detectLanguage(f); got != want {
			t.Fatalf("%s => %s want %s", f, got, want)
		}
	}
}

func Test_commentStyleHelpers(t *testing.T) {
	st := getCommentStyle(".go")
	if st.isNone() {
		t.Fatal("go style none")
	}
	if !hasSingleLineCommentPrefix("//abc", st.single) {
		t.Fatal("single line prefix")
	}
	if hasSingleLineCommentPrefix("abc//", st.single) {
		t.Fatal("should require prefix at start")
	}
	if style := getCommentStyle(".nope"); !style.isNone() {
		t.Fatal("unexpected style")
	}
}

func Test_isBlankAndSpace(t *testing.T) {
	if !isBlank("  \t") || isBlank("a") {
		t.Fatal("blank logic")
	}
	if !isSpace(' ') || isSpace('x') {
		t.Fatal("space logic")
	}
}

func Test_SingleFileCounter_Counts(t *testing.T) {
	dir := t.TempDir()
	content := `// package doc
	// package main
	/* block
	comment */
	// line 1

	func main() {}
	`
	p := writeFile(t, dir, "main.go", content)
	s := &SingleFileCounter{}
	ctx := context.Background()
	fi, err := s.CountSingleFile(ctx, p, Options{WithLanguageSpecific: true})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if fi.Language != "Go" {
		t.Fatalf("lang %s", fi.Language)
	}
	lines, _ := s.CountLines(ctx, p)
	if lines == 0 {
		t.Fatal("lines 0")
	}
	blanks, _ := s.CountBlanks(ctx, p)
	if blanks != 2 {
		t.Fatalf("blanks %d", blanks)
	}
	comments, _ := s.CountComments(ctx, p)
	if comments < 3 {
		t.Fatalf("comments %d", comments)
	}
}

func Test_SingleFileCounter_SizeLimit(t *testing.T) {
	dir := t.TempDir()
	var b strings.Builder
	for range 100 {
		b.WriteString("line\n")
	}
	p := writeFile(t, dir, "big.txt", b.String())
	st, _ := os.Stat(p)
	s := &SingleFileCounter{}
	_, err := s.CountSingleFile(context.Background(), p, Options{MaxFileSizeBytes: st.Size() - 1})
	if err == nil {
		t.Fatal("expected size error")
	}
}

func Test_prepareConcurrency(t *testing.T) {
	if got := prepareConcurrency(5); got != 5 {
		t.Fatalf("got %d", got)
	}
	if got := prepareConcurrency(0); got < 1 || got > runtime.NumCPU() {
		t.Fatalf("default conc %d", got)
	}
}
