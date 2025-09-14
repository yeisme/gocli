package debug

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/yeisme/gocli/pkg/style"
	"github.com/yeisme/gocli/pkg/utils/executor"
)

// MemStaticOptions 用于控制如何运行并展示编译器的内存/逃逸诊断信息。
type MemStaticOptions struct {
	Level   int    // 诊断等级：1 或 2；分别对应 -gcflags=all=-m 与 -m=2
	Only    string // 过滤类别：escape,noescape,inline,other（用逗号分隔）
	Tags    string // 透传给构建的 -tags
	Mod     string // 透传给构建的 -mod（如 mod、vendor）
	JSON    bool   // 以 JSON 输出而非表格
	Verbose bool   // 打印底层 go 命令
}

// MemDiag 表示编译器输出的一条诊断信息。
type MemDiag struct {
	Kind    string `json:"kind"` // 逃逸类别：escape|noescape|inline|other
	Pos     string `json:"pos"`  // 位置：file:line[:col]
	File    string `json:"file"`
	Line    int    `json:"line"`
	Col     int    `json:"col"`
	Message string `json:"message"`
	Pkg     string `json:"pkg"`
}

var (
	// 匹配两类位置格式：
	// 1) file:line:col: message
	// 2) file:line: message
	rePos3 = regexp.MustCompile(`^(.+?):(\d+):(\d+):\s+(.*)$`)
	rePos2 = regexp.MustCompile(`^(.+?):(\d+):\s+(.*)$`)
)

// classify 根据消息内容粗略判定诊断类型。
func classify(msg string) string {
	m := strings.ToLower(msg)
	switch {
	case strings.Contains(m, "escapes to heap"), strings.Contains(m, "moved to heap"):
		return "escape"
	case strings.Contains(m, "does not escape"):
		return "noescape"
	case strings.Contains(m, "can inline"), strings.Contains(m, "inlining call to"), strings.Contains(m, "cannot inline"), strings.Contains(m, "inline call"):
		return "inline"
	default:
		return "other"
	}
}

func parseLine(line string, pkg string) MemDiag {
	if m := rePos3.FindStringSubmatch(line); m != nil {
		ln, _ := strconv.Atoi(m[2])
		col, _ := strconv.Atoi(m[3])
		msg := m[4]
		return MemDiag{
			Kind:    classify(msg),
			Pos:     fmt.Sprintf("%s:%s:%s", m[1], m[2], m[3]),
			File:    m[1],
			Line:    ln,
			Col:     col,
			Message: msg,
			Pkg:     pkg,
		}
	}
	if m := rePos2.FindStringSubmatch(line); m != nil {
		ln, _ := strconv.Atoi(m[2])
		msg := m[3]
		return MemDiag{
			Kind:    classify(msg),
			Pos:     fmt.Sprintf("%s:%s", m[1], m[2]),
			File:    m[1],
			Line:    ln,
			Message: msg,
			Pkg:     pkg,
		}
	}
	// 兜底：整行作为消息处理
	return MemDiag{Kind: classify(line), Pos: "", File: "", Line: 0, Message: strings.TrimSpace(line), Pkg: pkg}
}

// filterKinds 根据 Only 字符串构建过滤集合。
func filterKinds(only string) map[string]struct{} {
	s := strings.TrimSpace(only)
	if s == "" {
		return nil
	}
	set := make(map[string]struct{})
	for _, p := range strings.Split(s, ",") {
		k := strings.ToLower(strings.TrimSpace(p))
		if k != "" {
			set[k] = struct{}{}
		}
	}
	return set
}

// RunMemStatic 对给定包运行 `go build -gcflags=all=-m[=2]` 并输出诊断信息。
// 通过将构建产物写入临时文件，避免在工作目录留下可执行文件。
func RunMemStatic(stderr io.Writer, stdout io.Writer, opt MemStaticOptions, pkgs ...string) error {
	if len(pkgs) == 0 {
		pkgs = []string{"."}
	}

	// 组装 gcflags
	level := opt.Level
	if level != 2 {
		level = 1
	}
	gcFlag := "-m"
	if level >= 2 {
		gcFlag = "-m=2"
	}

	// 使用临时输出文件，避免污染工作目录。
	// 在 Windows 上使用 .exe 扩展名，其他平台无扩展名。
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}

	onlySet := filterKinds(opt.Only)
	allDiags := make([]MemDiag, 0, 256)
	var lastErr error

	for _, pkg := range pkgs {
		// 创建临时输出文件路径（使用后删除）
		tmpBase := filepath.Join(os.TempDir(), fmt.Sprintf("gocli-mem-*%s", ext))
		tmpFile, err := os.CreateTemp(filepath.Dir(tmpBase), filepath.Base(tmpBase))
		if err != nil {
			return fmt.Errorf("create temp file: %w", err)
		}
		tmpPath := tmpFile.Name()
		_ = tmpFile.Close()
		defer func() {
			if err := os.Remove(tmpPath); err != nil {
				fmt.Fprintf(stderr, "remove temp file: %v\n", err)
			}
		}()

		// 组装 go build 参数
		args := []string{"build", "-gcflags=all=" + gcFlag, "-o", tmpPath}
		if strings.TrimSpace(opt.Tags) != "" {
			args = append(args, "-tags", opt.Tags)
		}
		if strings.TrimSpace(opt.Mod) != "" {
			args = append(args, "-mod", opt.Mod)
		}
		args = append(args, pkg)

		if opt.Verbose {
			fmt.Fprintf(stderr, "running: go %s\n", strings.Join(args, " "))
		}

		// 诊断信息输出到 stderr；即便失败也要同时捕获 stdout/stderr。
		out, errOut, runErr := executor.NewExecutor("go", args...).Run()
		// 编译器诊断主要在 errOut；但部分 Go 版本可能会混入到 stdout。
		raw := strings.TrimRight(errOut, "\n")
		if raw == "" {
			raw = strings.TrimRight(out, "\n")
		}
		lines := []string{}
		if raw != "" {
			lines = strings.Split(raw, "\n")
		}

		// 解析并收集
		for _, line := range lines {
			l := strings.TrimSpace(line)
			if l == "" {
				continue
			}
			d := parseLine(l, pkg)
			if onlySet != nil {
				if _, ok := onlySet[d.Kind]; !ok {
					continue
				}
			}
			allDiags = append(allDiags, d)
		}

		if runErr != nil {
			// 保留最后一次的错误，但继续处理其它包
			lastErr = runErr
		}
	}

	// 稳定排序：先 Kind，再 File，后 Line
	sort.Slice(allDiags, func(i, j int) bool {
		if allDiags[i].Kind != allDiags[j].Kind {
			return allDiags[i].Kind < allDiags[j].Kind
		}
		if allDiags[i].File != allDiags[j].File {
			return allDiags[i].File < allDiags[j].File
		}
		if allDiags[i].Line != allDiags[j].Line {
			return allDiags[i].Line < allDiags[j].Line
		}
		return allDiags[i].Message < allDiags[j].Message
	})

	// 输出
	if opt.JSON {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(allDiags); err != nil {
			return err
		}
	} else {
		rows := make([][]string, 0, len(allDiags))
		for _, d := range allDiags {
			pos := d.Pos
			if pos == "" && d.File != "" {
				if d.Line > 0 {
					pos = fmt.Sprintf("%s:%d", d.File, d.Line)
				} else {
					pos = d.File
				}
			}
			rows = append(rows, []string{d.Kind, d.Pkg, pos, d.Message})
		}
		if err := style.PrintTable(stdout, []string{"Kind", "Pkg", "Pos", "Message"}, rows, 0); err != nil {
			return err
		}
		if len(allDiags) == 0 {
			fmt.Fprintln(stdout, "没有诊断信息（可尝试使用 --m 2 查看更详细输出）")
		}
	}

	// 若构建发生错误，在打印已有结果后向上返回该错误。
	return lastErr
}
