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
func GetGoDoc(opts Options, _, path string) (string, error) {
	// 1. 规范化选项
	opts = normalizeDocOptions(opts)
	// 2. 解析目录
	dir, err := resolveTargetDir(path)
	if err != nil {
		return "", err
	}
	log.Debug().
		Str("dir", dir).
		Bool("includeTests", opts.IncludeTests).
		Bool("includeExamples", opts.IncludeExamples).
		Bool("includePrivate", opts.IncludePrivate).
		Msg("GetGoDoc: parsing package")

	// 3. 解析目录文件（examples 也需要解析 *_test.go）
	includeTestFiles := opts.IncludeTests || opts.IncludeExamples
	fset := token.NewFileSet()
	filesByPkg, err := parseDirectoryFiles(fset, dir, includeTestFiles)
	if err != nil {
		return "", err
	}
	// 4. 选择主包和外部测试包（当 includeTestFiles=true 时，也会返回外部测试包文件）
	mainFiles, extraTestFiles, err := selectPackageFiles(filesByPkg, includeTestFiles)
	if err != nil {
		return "", err
	}
	// 5. 构建 *go/doc.Package
	dpkg, err := buildDocPackage(fset, dir, mainFiles, opts.IncludePrivate)
	if err != nil {
		return "", err
	}
	// 6. 附加测试文件名（仅当需要展示 tests）
	if opts.IncludeTests {
		appendTestFilenames(dpkg, fset, mainFiles, extraTestFiles)
	}
	// 7. 如果只开启 examples（未开启 tests），仍需合并外部测试包 examples
	if opts.IncludeExamples && !opts.IncludeTests && len(extraTestFiles) > 0 {
		importPath := filepath.Base(dir) + "_test"
		if tpkg, e := gdoc.NewFromFiles(fset, extraTestFiles, importPath, gdoc.Mode(0)); e == nil && len(tpkg.Examples) > 0 {
			log.Debug().Int("examples", len(tpkg.Examples)).Msg("GetGoDoc: merged external test package examples")
			dpkg.Examples = append(dpkg.Examples, tpkg.Examples...)
		}
	}
	// 8. 收集测试/benchmark/example 函数（仅 tests 模式）
	var testFuncs []*ast.FuncDecl
	if opts.IncludeTests {
		testFuncs = collectTestFunctions(fset, mainFiles, extraTestFiles)
	}
	// 9. 渲染
	str, _ := parseGoDoc(opts, dpkg, fset, testFuncs)
	return str, nil
}

func normalizeDocOptions(o Options) Options {
	if o.IncludePrivate && !o.IncludeTests { // private implies tests
		o.IncludeTests = true
	}
	if o.IncludeTests && !o.IncludeExamples { // tests imply examples
		o.IncludeExamples = true
	}
	return o
}

func resolveTargetDir(path string) (string, error) {
	dir := path
	fi, err := os.Stat(dir)
	if err != nil {
		return "", fmt.Errorf("stat path failed: %w", err)
	}
	if !fi.IsDir() {
		dir = filepath.Dir(dir)
	}
	return dir, nil
}

func parseDirectoryFiles(fset *token.FileSet, dir string, includeTests bool) (map[string][]*ast.File, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir failed: %w", err)
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
		if !includeTests && strings.HasSuffix(name, "_test.go") {
			continue
		}
		full := filepath.Join(dir, name)
		f, parseErr := parser.ParseFile(fset, full, nil, parser.ParseComments)
		if parseErr != nil {
			return nil, fmt.Errorf("parse file %s failed: %w", full, parseErr)
		}
		filesByPkg[f.Name.Name] = append(filesByPkg[f.Name.Name], f)
	}
	if len(filesByPkg) == 0 {
		return nil, fmt.Errorf("no go files found under %s", dir)
	}
	return filesByPkg, nil
}

func selectPackageFiles(filesByPkg map[string][]*ast.File, includeTests bool) (mainFiles []*ast.File, extraTestFiles []*ast.File, err error) {
	var chosenName string
	for name, fs := range filesByPkg {
		if mainFiles == nil || (!strings.HasSuffix(name, "_test") && strings.HasSuffix(chosenName, "_test")) {
			chosenName = name
			mainFiles = fs
		}
	}
	if len(mainFiles) == 0 {
		return nil, nil, fmt.Errorf("failed to select package")
	}
	if includeTests {
		if ext, ok := filesByPkg[chosenName+"_test"]; ok {
			extraTestFiles = ext
		}
	}
	return
}

func buildDocPackage(fset *token.FileSet, dir string, files []*ast.File, includePrivate bool) (*gdoc.Package, error) {
	var mode gdoc.Mode
	if includePrivate {
		mode |= gdoc.AllDecls | gdoc.AllMethods
	}
	importPath := filepath.Base(dir)
	dpkg, err := gdoc.NewFromFiles(fset, files, importPath, mode)
	if err != nil {
		return nil, fmt.Errorf("failed to create doc.Package: %w", err)
	}
	return dpkg, nil
}

func appendTestFilenames(dpkg *gdoc.Package, fset *token.FileSet, mainFiles, extraTestFiles []*ast.File) {
	existing := make(map[string]struct{}, len(dpkg.Filenames))
	for _, fn := range dpkg.Filenames {
		existing[fn] = struct{}{}
	}
	added := 0
	add := func(fs []*ast.File) {
		for _, f := range fs {
			pos := fset.Position(f.Pos())
			if strings.HasSuffix(pos.Filename, "_test.go") {
				if _, ok := existing[pos.Filename]; !ok {
					dpkg.Filenames = append(dpkg.Filenames, pos.Filename)
					existing[pos.Filename] = struct{}{}
					added++
				}
			}
		}
	}
	add(mainFiles)
	add(extraTestFiles)
	if added > 0 {
		log.Debug().Int("added", added).Msg("GetGoDoc: appended test filenames for rendering")
	}
}

func collectTestFunctions(fset *token.FileSet, mainFiles, extraTestFiles []*ast.File) []*ast.FuncDecl {
	var testFuncs []*ast.FuncDecl
	collect := func(fs []*ast.File) {
		for _, f := range fs {
			pos := fset.Position(f.Pos())
			if !strings.HasSuffix(pos.Filename, "_test.go") {
				continue
			}
			for _, decl := range f.Decls {
				fd, ok := decl.(*ast.FuncDecl)
				if !ok || fd.Recv != nil {
					continue
				}
				name := fd.Name.Name
				if strings.HasPrefix(name, "Test") || strings.HasPrefix(name, "Benchmark") || strings.HasPrefix(name, "Example") {
					testFuncs = append(testFuncs, fd)
				}
			}
		}
	}
	collect(mainFiles)
	collect(extraTestFiles)
	if len(testFuncs) > 0 {
		log.Debug().Int("tests", len(testFuncs)).Msg("GetGoDoc: collected test/benchmark/example functions")
	}
	return testFuncs
}

// parseGoDoc 解析 doc.Package ，并结合 opts 生成合适的文档结构
func parseGoDoc(opts Options, dpkg *gdoc.Package, fset *token.FileSet, testFuncs []*ast.FuncDecl) (string, error) {
	// dispatch by style - currently only plain is implemented
	switch opts.Style {
	case StylePlain:
		return renderPlainDoc(opts, dpkg, fset, testFuncs)
	case StyleMarkdown:
		// TODO: implement Markdown renderer
		return renderPlainDoc(opts, dpkg, fset, testFuncs)
	case StyleHTML:
		// TODO: implement HTML renderer
		return renderPlainDoc(opts, dpkg, fset, testFuncs)
	default:
		return renderPlainDoc(opts, dpkg, fset, testFuncs)
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

// renderExamples 输出 examples 列表，支持简洁模式与 detailed 模式
func renderExamples(buf *strings.Builder, dpkg *gdoc.Package, fset *token.FileSet, opts Options) {
	if len(dpkg.Examples) == 0 {
		return
	}

	// 简洁模式：只输出名称 + 首行摘要，用箭头连接
	if !opts.Detailed {
		fmt.Fprintf(buf, "Examples:\n")
		for _, ex := range dpkg.Examples {
			name := ex.Name
			if name == "" {
				name = "_"
			}
			summary := ""
			if ex.Doc != "" {
				summary = strings.SplitN(strings.TrimSpace(ex.Doc), "\n", 2)[0]
			}
			if summary != "" {
				fmt.Fprintf(buf, "    Example %s —> %s\n", name, summary)
			} else {
				fmt.Fprintf(buf, "    Example %s\n", name)
			}
		}
		fmt.Fprintln(buf)
		return
	}

	// Detailed 模式：输出完整文档、位置以及代码（签名 / 示例代码）
	fmt.Fprintf(buf, "=== Examples ===\n\n")
	for _, ex := range dpkg.Examples {
		name := ex.Name
		if name == "" {
			name = "_"
		}
		fmt.Fprintf(buf, "Example %s:\n", name)
		if ex.Doc != "" {
			fmt.Fprintf(buf, "%s\n", indentLines(strings.TrimSpace(ex.Doc), "    "))
		}
		if pos := declPosition(ex.Code, fset); pos != "" { // 代码位置（如果能获取）
			fmt.Fprintf(buf, "    // defined at %s\n", pos)
		}
		if ex.Code != nil { // 打印代码（缩进）
			var cb strings.Builder
			_ = printer.Fprint(&cb, fset, ex.Code)
			codeStr := strings.TrimSpace(cb.String())
			if codeStr != "" {
				// 为代码中每一行添加前缀
				fmt.Fprintf(buf, "%s\n", indentLines(codeStr, "    "))
			}
		}
		if ex.Output != "" { // 如果 example 有 Output，展示出来
			out := strings.TrimRight(ex.Output, "\n")
			if out != "" {
				fmt.Fprintf(buf, "    // Output:\n")
				for _, line := range strings.Split(out, "\n") {
					if strings.TrimSpace(line) == "" {
						continue
					}
					fmt.Fprintf(buf, "    //   %s\n", line)
				}
			}
		}
		fmt.Fprintln(buf)
	}
}
