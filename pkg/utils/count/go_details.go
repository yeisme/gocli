package count

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"

	"github.com/yeisme/gocli/pkg/models"
)

// GoDetailsCounter 提供对 Go 文件的语言特定统计实现
type GoDetailsCounter struct {
	SingleFileCounter
}

// CountGoDetails 解析 Go 源文件，返回包名、导入、函数数与结构体数
func (s *GoDetailsCounter) CountGoDetails(ctx context.Context, filePath string) (*models.GoDetails, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// 简单保护：仅对 .go 文件处理
	if !strings.EqualFold(filepath.Ext(filePath), ".go") {
		return &models.GoDetails{}, nil
	}

	f, err := parseGoFile(filePath)
	if err != nil {
		return nil, err
	}

	// 收集 imports（去掉引号）
	imports := make([]string, 0, len(f.Imports))
	for _, imp := range f.Imports {
		if imp.Path == nil {
			continue
		}
		p := strings.Trim(imp.Path.Value, "`\"")
		if p != "" {
			imports = append(imports, p)
		}
	}

	funcs := countGoFunctions(f)
	structs := countGoStructs(f)

	details := &models.GoDetails{
		PackageName: f.Name.Name,
		Imports:     imports,
		Functions:   funcs,
		Structs:     structs,
	}
	return details, nil
}

// CountFunctions 统计 Go 文件中的函数与方法数量
func (s *GoDetailsCounter) CountFunctions(ctx context.Context, filePath string) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if !strings.EqualFold(filepath.Ext(filePath), ".go") {
		return 0, nil
	}
	f, err := parseGoFile(filePath)
	if err != nil {
		return 0, err
	}
	return countGoFunctions(f), nil
}

// CountStructs 统计 Go 文件中的结构体类型数量
func (s *GoDetailsCounter) CountStructs(ctx context.Context, filePath string) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if !strings.EqualFold(filepath.Ext(filePath), ".go") {
		return 0, nil
	}
	f, err := parseGoFile(filePath)
	if err != nil {
		return 0, err
	}
	return countGoStructs(f), nil
}

// parseGoFile 使用 go/parser 解析 Go 源文件
func parseGoFile(filePath string) (*ast.File, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func countGoFunctions(f *ast.File) int {
	count := 0
	ast.Inspect(f, func(n ast.Node) bool {
		if n == nil {
			return true
		}
		if fn, ok := n.(*ast.FuncDecl); ok {
			// 包含普通函数与方法（Recv != nil）
			if fn.Name != nil && fn.Name.Name != "" {
				count++
			}
			// 不必深入函数体
			return false
		}
		return true
	})
	return count
}

func countGoStructs(f *ast.File) int {
	count := 0
	ast.Inspect(f, func(n ast.Node) bool {
		gd, ok := n.(*ast.GenDecl)
		if !ok || gd.Tok != token.TYPE {
			return true
		}
		for _, spec := range gd.Specs {
			if ts, ok := spec.(*ast.TypeSpec); ok {
				if _, ok := ts.Type.(*ast.StructType); ok {
					count++
				}
			}
		}
		return true
	})
	return count
}

var _ GoFile = (*GoDetailsCounter)(nil)
