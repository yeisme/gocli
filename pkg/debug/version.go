package debug

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/yeisme/gocli/pkg/style"
	"github.com/yeisme/gocli/pkg/utils/executor"
)

// PrintVersionTable 运行 `go version -m <exe>` 并将结果以表格形式打印到 w。
// 如果 args 执行失败，表格仍然会被打印，随后返回底层错误以便上层处理。
func PrintVersionTable(w io.Writer, exePath string) error {
	// 走绝对路径，避免输出中的路径因相对路径展示问题
	absExe := exePath
	if p, err := filepath.Abs(exePath); err == nil {
		absExe = p
	}

	// 使用 go version -m -json <exe>
	out, execErr := executor.NewExecutor("go", "version", "-m", "-json", absExe).Output()

	type depInfo struct {
		Path    string `json:"Path"`
		Version string `json:"Version"`
		Sum     string `json:"Sum"`
	}
	type mainInfo struct {
		Path    string `json:"Path"`
		Version string `json:"Version"`
	}
	type ver struct {
		GoVersion string    `json:"GoVersion"`
		Path      string    `json:"Path"`
		Main      mainInfo  `json:"Main"`
		Deps      []depInfo `json:"Deps"`
	}

	var v ver
	if strings.TrimSpace(out) != "" {
		if err := json.Unmarshal([]byte(out), &v); err != nil {
			// 无法解析 JSON，则将原始输出作为 info 行打印
			rows := [][]string{{"raw", strings.TrimSpace(out)}}
			_ = style.PrintTable(w, []string{"Field", "Value"}, rows, 0)
			if execErr != nil {
				return fmt.Errorf("failed to parse json output: %w", execErr)
			}
			return fmt.Errorf("failed to parse json output: %w", err)
		}
	}

	var rows [][]string
	var base []string
	// 基本信息
	if v.GoVersion != "" {
		base = append(base, v.GoVersion)
	}
	if v.Path != "" {
		base = append(base, v.Path)
	}
	if v.Main.Path != "" {
		base = append(base, v.Main.Path)
	}
	if v.Main.Version != "" {
		base = append(base, v.Main.Version)
	}
	if len(base) > 0 {
		// base 是单行，PrintTable 期待 [][]string
		basicRows := [][]string{base}
		if err := style.PrintTable(w, []string{"GoVersion", "Path", "Main.Path", "Main.Version"}, basicRows, 0); err != nil {
			return err
		}
	}

	// Deps 列表: 输出为四列 -> [Dep[i], Path, Version, Sum]
	for i, d := range v.Deps {
		idx := fmt.Sprintf("%d", i)
		rows = append(rows, []string{"Dep[" + idx + "]", d.Path, d.Version, d.Sum})
	}

	if len(rows) == 0 {
		fmt.Fprintln(os.Stderr, "no output from 'go version -m -json'")
		if execErr != nil {
			return execErr
		}
		return nil
	}

	// 根据最大列数选择表头
	maxCols := 0
	for _, r := range rows {
		if len(r) > maxCols {
			maxCols = len(r)
		}
	}

	switch maxCols {
	case 4:
		if err := style.PrintTable(w, []string{"Dep", "Path", "Version", "Sum"}, rows, 0); err != nil {
			return err
		}
	case 3:
		if err := style.PrintTable(w, []string{"", "Path", "Details"}, rows, 0); err != nil {
			return err
		}
	default:
		if err := style.PrintTable(w, []string{"Field", "Value"}, rows, 0); err != nil {
			return err
		}
	}

	if execErr != nil {
		return execErr
	}

	return nil
}
