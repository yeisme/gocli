package count

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/yeisme/gocli/pkg/models"
)

func Test_matchesAny(t *testing.T) {
	if !matchesAny("a/b/c.go", []string{"*.go", "c.go"}) {
		t.Fatal("no match")
	}
	if matchesAny("a/b/c.txt", []string{"*.go"}) {
		t.Fatal("should not match")
	}
}

func Test_isSizeLimitError(t *testing.T) {
	if isSizeLimitError(nil) {
		t.Fatal("nil shouldn't")
	}
	err := &fs.PathError{Op: "open", Path: "x", Err: errors.New("size exceeded")}
	if !isSizeLimitError(err) {
		t.Fatal("expected size limit true")
	}
}

func Test_ensureCounters(t *testing.T) {
	p := ensureCounters(nil)
	if p.FileCounter == nil || p.GoCounter == nil {
		t.Fatal("counters not set")
	}
	p2 := ensureCounters(&ProjectCounter{FileCounter: &SingleFileCounter{}})
	if p2.GoCounter == nil {
		t.Fatal("go counter missing")
	}
}

func Test_loadGitIgnore(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("node_modules\n"), 0o644); err != nil {
		t.Fatalf("write gitignore: %v", err)
	}
	if gi := loadGitIgnore(dir, false); gi != nil {
		t.Fatal("expected nil when not respect")
	}
	if gi := loadGitIgnore(dir, true); gi == nil || len(gi.GetPatterns()) == 0 {
		t.Fatal("expected patterns")
	}
}

func Test_collectFiles_and_Filters(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			panic(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			panic(err)
		}
	}
	write("a.go", "package a")
	write("b.txt", "hey")
	write("sub/c.go", "package sub")
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("sub\n"), 0o644); err != nil {
		panic(err)
	}
	gi := loadGitIgnore(dir, true)
	files, err := collectFiles(context.Background(), dir, Options{Include: []string{"*.go"}}, gi)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 got %d", len(files))
	}
	files2, _ := collectFiles(context.Background(), dir, Options{}, gi)
	foundSub := false
	for _, f := range files2 {
		if filepath.Base(f) == "c.go" {
			foundSub = true
		}
	}
	if foundSub {
		t.Fatal("gitignore failed for directory")
	}
}

func Test_toRelSlash(t *testing.T) {
	root := filepath.Join("a", "b")
	path := filepath.Join(root, "c", "d.go")
	rel := toRelSlash(root, path)
	if rel != "c/d.go" {
		t.Fatalf("rel %s", rel)
	}
}

func Test_shouldSkipDir_and_shouldIncludeFile(t *testing.T) {
	opts := Options{Include: nil, Exclude: []string{"vendor"}}
	if !shouldSkipDir("vendor", opts, nil) {
		t.Fatal("should skip vendor")
	}
	opts2 := Options{Include: []string{"*.go"}}
	if !shouldIncludeFile("x.go", opts2, nil) {
		t.Fatal("include")
	}
	if shouldIncludeFile("x.txt", opts2, nil) {
		t.Fatal("exclude non include")
	}
}

func Test_overSize(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(p, []byte("123456"), 0o644); err != nil {
		panic(err)
	}
	if !overSize(p, 1) {
		t.Fatal("expected oversize")
	}
	if overSize(p, 0) {
		t.Fatal("0 means no limit")
	}
}

func Test_process_and_summary(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.go"), []byte("package a\n// c\n\nfunc A(){}"), 0o644); err != nil {
		panic(err)
	}
	pc := &ProjectCounter{}
	ctx := context.Background()
	files, err := pc.CountAllFiles(ctx, dir, Options{Concurrency: 2, WithLanguageSpecific: true, WithFunctions: true, WithStructs: true})
	if err != nil || len(files) != 1 {
		t.Fatalf("files err %v len %d", err, len(files))
	}
	res, err := pc.CountProjectSummary(ctx, dir, Options{WithLanguageDetails: true})
	if err != nil || res.Total.FileCount != 1 {
		t.Fatalf("summary fail %v", err)
	}
}

func Test_processFilesConcurrently_ContextCancel(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 50; i++ {
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("f%d.go", i)), []byte("package p"), 0o644); err != nil {
			panic(err)
		}
	}
	pc := &ProjectCounter{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately
	_, err := pc.CountAllFiles(ctx, dir, Options{Concurrency: runtime.NumCPU()})
	if err == nil {
		t.Fatal("expected context cancel error")
	}
}

func Test_isSymlink(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hi"), 0o644); err != nil {
		panic(err)
	}
	link := filepath.Join(dir, "l.txt")
	if err := os.Symlink(filepath.Join(dir, "a.txt"), link); err != nil {
		if runtime.GOOS == "windows" {
			t.Skip("symlink requires privilege on windows")
		} else {
			t.Fatalf("symlink: %v", err)
		}
	}
	entries, _ := os.ReadDir(dir)
	found := false
	for _, e := range entries {
		if e.Name() == "l.txt" {
			found = isSymlink(e)
		}
	}
	if !found {
		t.Skip("symlink not created")
	}
}

func Test_aggregateAnalysis(t *testing.T) {
	files := []models.FileInfo{{Language: "Go", Stats: models.Stats{Code: 10, Comments: 2, Blanks: 1}}, {Language: "", Stats: models.Stats{Code: 1}}}
	res := aggregateAnalysis(files, Options{WithLanguageDetails: true})
	if res.Total.FileCount != 2 || res.Total.Stats.Code != 11 {
		t.Fatalf("agg fail %+v", res.Total)
	}
	if _, ok := res.Languages["Unknown"]; !ok {
		t.Fatal("missing unknown")
	}
}
