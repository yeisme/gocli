package count

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func Test_GoDetailsCounter(t *testing.T) {
	dir := t.TempDir()
	code := `package demo
import "fmt"
import alias "strings"
// comment
/* block */

type A struct{ X int }

type B struct{}

func f(){}
func (A) M(){}
`
	p := filepath.Join(dir, "file.go")
	if err := os.WriteFile(p, []byte(code), 0o644); err != nil {
		t.Fatal(err)
	}
	g := &GoDetailsCounter{}
	ctx := context.Background()
	d, err := g.CountGoDetails(ctx, p)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if d.PackageName != "demo" {
		t.Fatalf("pkg %s", d.PackageName)
	}
	if d.Structs != 2 {
		t.Fatalf("structs %d", d.Structs)
	}
	if d.Functions != 2 {
		t.Fatalf("funcs %d", d.Functions)
	}
	fns, _ := g.CountFunctions(ctx, p)
	if fns != 2 {
		t.Fatalf("functions %d", fns)
	}
	ss, _ := g.CountStructs(ctx, p)
	if ss != 2 {
		t.Fatalf("structs %d", ss)
	}
}

func Test_parseGoFile_helpers(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "x.go")
	_ = os.WriteFile(p, []byte("package x"), 0o644)
	f, err := parseGoFile(p)
	if err != nil || f.Name.Name != "x" {
		t.Fatalf("parse fail: %v", err)
	}
	if countGoFunctions(f) != 0 {
		t.Fatal("unexpected funcs")
	}
	if countGoStructs(f) != 0 {
		t.Fatal("unexpected structs")
	}
}
