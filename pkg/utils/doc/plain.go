package doc

import (
	"fmt"
	gdoc "go/doc"
	"go/printer"
	"go/token"
	"sort"
	"strings"
)

// renderPlain renders documentation in plain text. It is composed of smaller helpers
// so we can later add other renderers (markdown/html/json) easily.
func renderPlainDoc(opts Options, dpkg *gdoc.Package, fset *token.FileSet) (string, error) {
	var buf strings.Builder

	renderHeader(&buf, dpkg)
	renderFilesAndImports(&buf, dpkg)
	renderNotes(&buf, dpkg)
	renderDecls(&buf, dpkg, fset, opts)
	renderExamples(&buf, dpkg, fset)

	return buf.String(), nil
}

func renderHeader(buf *strings.Builder, dpkg *gdoc.Package) {
	if strings.TrimSpace(dpkg.Doc) != "" {
		fmt.Fprintf(buf, "%s\n\n", strings.TrimSpace(dpkg.Doc))
	}
}

func renderFilesAndImports(buf *strings.Builder, dpkg *gdoc.Package) {
	if len(dpkg.Filenames) > 0 {
		fmt.Fprintf(buf, "Files:\n")
		sort.Strings(dpkg.Filenames)
		for _, fn := range dpkg.Filenames {
			fmt.Fprintf(buf, "    %s\n", fn)
		}
		fmt.Fprintln(buf)
	}

	if len(dpkg.Imports) > 0 {
		fmt.Fprintf(buf, "Imports:\n")
		imports := make([]string, len(dpkg.Imports))
		copy(imports, dpkg.Imports)
		sort.Strings(imports)
		for _, im := range imports {
			fmt.Fprintf(buf, "    %s\n", im)
		}
		fmt.Fprintln(buf)
	}
}

func renderNotes(buf *strings.Builder, dpkg *gdoc.Package) {
	if len(dpkg.Notes) == 0 {
		return
	}
	keys := make([]string, 0, len(dpkg.Notes))
	for k := range dpkg.Notes {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(buf, "Notes (%s):\n", k)
		for _, n := range dpkg.Notes[k] {
			fmt.Fprintf(buf, "    %s\n", strings.TrimSpace(n.Body))
		}
		fmt.Fprintln(buf)
	}
}

func renderDecls(buf *strings.Builder, dpkg *gdoc.Package, fset *token.FileSet, opts Options) {
	if !opts.Detailed {
		renderDeclsSimple(buf, dpkg, fset)
		return
	}
	renderDeclsDetailed(buf, dpkg, fset)
}

// renderDeclsSimple simple (summary) renderer
func renderDeclsSimple(buf *strings.Builder, dpkg *gdoc.Package, fset *token.FileSet) {
	joinNames := func(names []string) string { return strings.Join(names, ", ") }

	if len(dpkg.Consts) > 0 {
		fmt.Fprintf(buf, "Constants:\n")
		for _, v := range dpkg.Consts {
			fmt.Fprintf(buf, "    %s", joinNames(v.Names))
			if v.Doc != "" {
				first := strings.SplitN(strings.TrimSpace(v.Doc), "\n", 2)[0]
				fmt.Fprintf(buf, " —> %s", first)
			}
			fmt.Fprintln(buf)
		}
		fmt.Fprintln(buf)
	}

	if len(dpkg.Vars) > 0 {
		fmt.Fprintf(buf, "Variables:\n")
		for _, v := range dpkg.Vars {
			fmt.Fprintf(buf, "    %s", joinNames(v.Names))
			if v.Doc != "" {
				first := strings.SplitN(strings.TrimSpace(v.Doc), "\n", 2)[0]
				fmt.Fprintf(buf, " —> %s", first)
			}
			fmt.Fprintln(buf)
		}
		fmt.Fprintln(buf)
	}

	if len(dpkg.Funcs) > 0 {
		fmt.Fprintf(buf, "Functions:\n")
		for _, f := range dpkg.Funcs {
			printFuncSignatureSimple(buf, f, fset)
			if f.Doc != "" {
				first := strings.SplitN(strings.TrimSpace(f.Doc), "\n", 2)[0]
				fmt.Fprintf(buf, " —> %s", first)
			}
			fmt.Fprintln(buf)
		}
		fmt.Fprintln(buf)
	}

	if len(dpkg.Types) > 0 {
		fmt.Fprintf(buf, "Types:\n")
		for _, t := range dpkg.Types {
			fmt.Fprintf(buf, "    %s", t.Name)
			if t.Doc != "" {
				first := strings.SplitN(strings.TrimSpace(t.Doc), "\n", 2)[0]
				fmt.Fprintf(buf, " —> %s", first)
			}
			fmt.Fprintln(buf)
			renderTypeSummarySimple(buf, t, fset)
		}
		fmt.Fprintln(buf)
	}
}

func printFuncSignatureSimple(buf *strings.Builder, f *gdoc.Func, fset *token.FileSet) {
	if f.Decl != nil {
		fd := *f.Decl
		fd.Body = nil
		var sb strings.Builder
		_ = printer.Fprint(&sb, fset, &fd)
		fmt.Fprintf(buf, "    %s", strings.TrimSpace(sb.String()))
	} else {
		fmt.Fprintf(buf, "    %s", f.Name)
	}
}

func renderTypeSummarySimple(buf *strings.Builder, t *gdoc.Type, fset *token.FileSet) {
	if len(t.Methods) > 0 {
		for _, m := range t.Methods {
			if m.Decl != nil {
				md := *m.Decl
				md.Body = nil
				var sb strings.Builder
				_ = printer.Fprint(&sb, fset, &md)
				fmt.Fprintf(buf, "        %s\n", strings.TrimSpace(sb.String()))
			} else {
				fmt.Fprintf(buf, "        %s\n", m.Name)
			}
		}
		fmt.Fprintf(buf, "        (methods: %d)\n", len(t.Methods))
	}
	if len(t.Funcs) > 0 {
		for _, af := range t.Funcs {
			if af.Decl != nil {
				fd := *af.Decl
				fd.Body = nil
				var sb strings.Builder
				_ = printer.Fprint(&sb, fset, &fd)
				fmt.Fprintf(buf, "        %s\n", strings.TrimSpace(sb.String()))
			} else {
				fmt.Fprintf(buf, "        %s\n", af.Name)
			}
		}
	}
}

// detailed renderer (beautified)
func renderDeclsDetailed(buf *strings.Builder, dpkg *gdoc.Package, fset *token.FileSet) {
	indent := func(s string, pref string) string {
		return indentLines(s, pref)
	}

	if len(dpkg.Consts) > 0 {
		fmt.Fprintf(buf, "=== Constants ===\n\n")
		for _, v := range dpkg.Consts {
			if v.Doc != "" {
				fmt.Fprintf(buf, "%s\n", indent(strings.TrimSpace(v.Doc), "    "))
			}
			if pos := declPosition(v.Decl, fset); pos != "" {
				fmt.Fprintf(buf, "    // defined at %s\n", pos)
			}
			fmt.Fprintf(buf, "%s\n", indentCapture(func() string { var b strings.Builder; _ = printer.Fprint(&b, fset, v.Decl); return b.String() }, "    "))
			fmt.Fprintln(buf)
		}
	}

	if len(dpkg.Vars) > 0 {
		fmt.Fprintf(buf, "=== Variables ===\n\n")
		for _, v := range dpkg.Vars {
			if v.Doc != "" {
				fmt.Fprintf(buf, "%s\n", indent(strings.TrimSpace(v.Doc), "    "))
			}
			if pos := declPosition(v.Decl, fset); pos != "" {
				fmt.Fprintf(buf, "    // defined at %s\n", pos)
			}
			fmt.Fprintf(buf, "%s\n", indentCapture(func() string { var b strings.Builder; _ = printer.Fprint(&b, fset, v.Decl); return b.String() }, "    "))
			fmt.Fprintln(buf)
		}
	}

	if len(dpkg.Funcs) > 0 {
		fmt.Fprintf(buf, "=== Functions ===\n\n")
		for _, f := range dpkg.Funcs {
			if f.Doc != "" {
				fmt.Fprintf(buf, "%s\n", indent(strings.TrimSpace(f.Doc), "    "))
			}
			if pos := declPosition(f.Decl, fset); pos != "" {
				fmt.Fprintf(buf, "    // defined at %s\n", pos)
			}
			// print signature only (body omitted)
			if f.Decl != nil {
				fd := *f.Decl
				fd.Body = nil
				var b strings.Builder
				_ = printer.Fprint(&b, fset, &fd)
				fmt.Fprintf(buf, "%s\n", indent(strings.TrimSpace(b.String()), "    "))
			}
			fmt.Fprintln(buf)
		}
	}

	if len(dpkg.Types) > 0 {
		fmt.Fprintf(buf, "=== Types ===\n\n")
		for _, t := range dpkg.Types {
			if t.Doc != "" {
				fmt.Fprintf(buf, "%s\n", indent(strings.TrimSpace(t.Doc), "    "))
			}
			if pos := declPosition(t.Decl, fset); pos != "" {
				fmt.Fprintf(buf, "    // defined at %s\n", pos)
			}
			// print type decl
			fmt.Fprintf(buf, "%s\n", indentCapture(func() string { var b strings.Builder; _ = printer.Fprint(&b, fset, t.Decl); return b.String() }, "    "))

			if len(t.Consts) > 0 {
				fmt.Fprintf(buf, "    -- associated constants --\n")
				for _, v := range t.Consts {
					fmt.Fprintf(buf, "%s\n", indentCapture(func() string { var b strings.Builder; _ = printer.Fprint(&b, fset, v.Decl); return b.String() }, "        "))
				}
			}
			if len(t.Vars) > 0 {
				fmt.Fprintf(buf, "    -- associated variables --\n")
				for _, v := range t.Vars {
					fmt.Fprintf(buf, "%s\n", indentCapture(func() string { var b strings.Builder; _ = printer.Fprint(&b, fset, v.Decl); return b.String() }, "        "))
				}
			}
			if len(t.Funcs) > 0 {
				fmt.Fprintf(buf, "    -- associated functions --\n")
				for _, af := range t.Funcs {
					if af.Decl != nil {
						fd := *af.Decl
						fd.Body = nil
						var b strings.Builder
						_ = printer.Fprint(&b, fset, &fd)
						fmt.Fprintf(buf, "%s\n", indent(strings.TrimSpace(b.String()), "        "))
					} else {
						fmt.Fprintf(buf, "        %s\n", af.Name)
					}
				}
			}
			if len(t.Methods) > 0 {
				fmt.Fprintf(buf, "    -- methods --\n")
				for _, m := range t.Methods {
					if m.Doc != "" {
						fmt.Fprintf(buf, "%s\n", indent(strings.TrimSpace(m.Doc), "        "))
					}
					if pos := declPosition(m.Decl, fset); pos != "" {
						fmt.Fprintf(buf, "        // defined at %s\n", pos)
					}
					if m.Decl != nil {
						md := *m.Decl
						md.Body = nil
						var b strings.Builder
						_ = printer.Fprint(&b, fset, &md)
						fmt.Fprintf(buf, "%s\n", indent(strings.TrimSpace(b.String()), "        "))
					}
				}
			}
			fmt.Fprintln(buf)
		}
	}
}
