package project

import (
	"fmt"
	"io"

	"github.com/yeisme/gocli/pkg/utils/deps"
)

// DepsOptions defines options for listing dependencies.
type DepsOptions struct {
	// 输出样式
	Graph bool // 生成依赖关系图
	Tree  bool // 生成依赖树
	JSON  bool // JSON 输出格式

	Update  bool // 检查可用的更新
	Verbose bool
}

// RunDeps executes go list command and pretty output
func RunDeps(options DepsOptions, out io.Writer, args []string) error {
	output, err := deps.RunGoModList(args, struct {
		JSON   bool
		Update bool
	}{
		JSON:   options.JSON,
		Update: options.Update,
	})
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(out, "%s", output)
	return nil
}
