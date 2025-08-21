package doc

import (
	"fmt"
	"go/ast"
	gdoc "go/doc"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// GetGoDoc 解析 root/rel 下的 Go 包文档，返回按 opts.Style 渲染后的字符串
// 行为简述：
//   - 解析目录（或文件）构建 *go/doc.Package
//   - 根据 opts.IncludeTests 过滤 *_test.go 文件
//   - 根据 opts.IncludePrivate 设置 doc.Mode 以及再次通过 Filter 过滤导出符号
//   - 渲染输出：plain/markdown/html/json/yaml
func GetGoDoc(opts Options, root, rel string) (string, error) {
	// 解析实际路径
	dir := rel
	if dir == "" || dir == "." {
		dir = root
	} else if !filepath.IsAbs(dir) {
		dir = filepath.Join(root, rel)
	}

	fi, err := os.Stat(dir)
	if err != nil {
		return "", fmt.Errorf("stat path failed: %w", err)
	}
	if !fi.IsDir() { // 如果传入的是文件，则取其目录
		dir = filepath.Dir(dir)
	}

	log.Debug().Str("dir", dir).Bool("includeTests", opts.IncludeTests).Bool("includePrivate", opts.IncludePrivate).Msg("GetGoDoc: parsing package")

	// 解析目录下的 Go 文件，分组到各个包名，避免直接依赖 *ast.Package
	fset := token.NewFileSet()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("read dir failed: %w", err)
	}

	filesByPkg := make(map[string][]*ast.File)
	for _, de := range entries {
		if de.IsDir() {
			continue
		}
		name := de.Name()
		if !strings.HasSuffix(name, ".go") {
			continue
		}
		if !opts.IncludeTests && strings.HasSuffix(name, "_test.go") {
			continue
		}
		full := filepath.Join(dir, name)
		// 解析单文件（包含注释）
		f, parseErr := parser.ParseFile(fset, full, nil, parser.ParseComments)
		if parseErr != nil {
			return "", fmt.Errorf("parse file %s failed: %w", full, parseErr)
		}
		pkgName := f.Name.Name
		filesByPkg[pkgName] = append(filesByPkg[pkgName], f)
	}
	if len(filesByPkg) == 0 {
		return "", fmt.Errorf("no go files found under %s (after filters)", dir)
	}

	// 选择一个目标包：优先非 *_test 包
	var (
		chosenName  string
		chosenFiles []*ast.File
	)
	for name, fs := range filesByPkg {
		if chosenFiles == nil || (!strings.HasSuffix(name, "_test") && strings.HasSuffix(chosenName, "_test")) {
			chosenName = name
			chosenFiles = fs
		}
	}
	if len(chosenFiles) == 0 {
		return "", fmt.Errorf("failed to select package under %s", dir)
	}

	// 构建 *go/doc.Package（通过 NewFromFiles）
	var mode gdoc.Mode
	if opts.IncludePrivate {
		mode |= gdoc.AllDecls | gdoc.AllMethods
	}
	// importPath 这里用相对目录名占位；若需要可改为真实模块路径
	importPath := filepath.Base(dir)
	dpkg, err := gdoc.NewFromFiles(fset, chosenFiles, importPath, mode)
	if err != nil {
		return "", fmt.Errorf("failed to create doc.Package: %w", err)
	}
	str, _ := parseGoDoc(opts, dpkg, fset)

	return str, nil
}

// parseGoDoc 解析 doc.Package ，并结合 opts 生成合适的文档结构
func parseGoDoc(opts Options, dpkg *gdoc.Package, fset *token.FileSet) (string, error) {
	// dispatch by style - currently only plain is implemented
	switch opts.Style {
	case StylePlain:
		return renderPlainDoc(opts, dpkg, fset)
	case StyleMarkdown:
		// TODO: implement Markdown renderer
		return renderPlainDoc(opts, dpkg, fset)
	case StyleHTML:
		// TODO: implement HTML renderer
		return renderPlainDoc(opts, dpkg, fset)
	default:
		return renderPlainDoc(opts, dpkg, fset)
	}
}

// indentLines prefixes each non-empty line in s with pref.
func indentLines(s, pref string) string {
	if s == "" {
		return ""
	}
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		if strings.TrimSpace(l) != "" {
			lines[i] = pref + l
		}
	}
	return strings.Join(lines, "\n")
}

// indentCapture runs fn to get a string and indents it by pref.
func indentCapture(fn func() string, pref string) string {
	return indentLines(fn(), pref)
}

// declPosition returns a short "file:line" for node if available.
func declPosition(n ast.Node, fset *token.FileSet) string {
	if n == nil || fset == nil {
		return ""
	}
	pos := fset.Position(n.Pos())
	if pos.Filename == "" || pos.Line == 0 {
		return ""
	}
	base := filepath.Base(pos.Filename)
	return fmt.Sprintf("%s:%d", base, pos.Line)
}

func renderExamples(buf *strings.Builder, dpkg *gdoc.Package, fset *token.FileSet) {
	if len(dpkg.Examples) == 0 {
		return
	}
	fmt.Fprintf(buf, "Examples:\n")
	for _, ex := range dpkg.Examples {
		name := ex.Name
		if name == "" {
			name = "_"
		}
		fmt.Fprintf(buf, "Example %s:\n", name)
		if strings.TrimSpace(ex.Doc) != "" {
			fmt.Fprintf(buf, "%s\n", strings.TrimSpace(ex.Doc))
		}
		if ex.Code != nil {
			var cb strings.Builder
			_ = printer.Fprint(&cb, fset, ex.Code)
			codeStr := strings.TrimSpace(cb.String())
			if codeStr != "" {
				fmt.Fprintf(buf, "%s\n", codeStr)
			}
		}
		fmt.Fprintln(buf)
	}
}
