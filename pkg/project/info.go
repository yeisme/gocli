package project

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"sort"

	gctx "github.com/yeisme/gocli/pkg/context"

	"github.com/yeisme/gocli/pkg/models"
	"github.com/yeisme/gocli/pkg/style"
	"github.com/yeisme/gocli/pkg/utils/count"
)

// InfoOptions 是用于获取项目详细信息的选项
type InfoOptions struct {
	count.Options
}

// ExecuteInfoCommand 负责执行业务逻辑（统计 + 输出），与 build/run 的风格保持一致
// 参数说明:
//
//	args: 可能包含一个 root 路径；为空则默认为当前目录 '.'
//	jsonOut: 是否输出 JSON
//	showProjectHeader: 是否在表格前输出 "Project: <root>"（受 quiet 影响）
//	w: 输出目标（通常为 cmd.OutOrStdout()）
func ExecuteInfoCommand(gocliCtx *gctx.GocliContext, opts InfoOptions, args []string, jsonOut bool, showProjectHeader bool, w io.Writer) error {
	_ = gocliCtx

	root := resolveInfoRoot(args)
	res, err := collectProjectAnalysis(root, opts)
	if err != nil {
		return err
	}

	if jsonOut {
		return printInfoJSON(w, res)
	}

	// 语言表
	langHeaders, langRows := buildLanguageTable(res, opts)
	if showProjectHeader {
		_, _ = fmt.Fprintf(w, "Project: %s\n", root)
	}
	if err := style.PrintTable(w, langHeaders, langRows, 0); err != nil {
		log.Error().Err(err).Msg("failed to print info table")
	}

	// 文件表
	if opts.WithFileDetails {
		fileHeaders, fileRows := buildFileTable(res, opts)
		if len(fileRows) > 0 {
			_, _ = fmt.Fprintln(w)
			_, _ = fmt.Fprintln(w, "Files:")
			if err := style.PrintTable(w, fileHeaders, fileRows, 0); err != nil {
				log.Error().Err(err).Msg("failed to print file table")
			}
		}
	}
	return nil
}

// resolveInfoRoot 解析根路径参数
func resolveInfoRoot(args []string) string {
	root := "."
	if len(args) > 0 && args[0] != "" {
		root = args[0]
	}
	if abs, err := filepath.Abs(root); err == nil {
		return abs
	}
	return root
}

// collectProjectAnalysis 调用计数器执行统计
func collectProjectAnalysis(root string, opts InfoOptions) (*models.AnalysisResult, error) {
	ctx := context.Background()
	pc := &count.ProjectCounter{}
	res, err := pc.CountProjectSummary(ctx, root, opts.Options)
	if err != nil {
		return nil, fmt.Errorf("count project summary failed: %w", err)
	}
	return res, nil
}

// printInfoJSON 输出 JSON 结果
func printInfoJSON(w io.Writer, res *models.AnalysisResult) error {
	b, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal project info failed: %w", err)
	}
	_, _ = fmt.Fprintln(w, string(b))
	return nil
}

// buildLanguageTable 构建语言统计表数据（含 TOTAL 行）
func buildLanguageTable(res *models.AnalysisResult, opts InfoOptions) ([]string, [][]string) {
	headers := []string{"language", "files", "code", "comments", "blanks", "code%", "lines"}
	if opts.WithFunctions {
		headers = append(headers, "funcs")
	}
	if opts.WithStructs {
		headers = append(headers, "structs")
	}

	langs := make([]string, 0, len(res.Languages))
	for l := range res.Languages {
		if l == "Unknown" {
			continue
		}
		langs = append(langs, l)
	}
	sort.Strings(langs)

	displayedTotalCode := 0
	for _, l := range langs {
		displayedTotalCode += res.Languages[l].Stats.Code
	}

	rows := make([][]string, 0, len(langs)+1)
	for _, l := range langs {
		ls := res.Languages[l]
		codePct := 0.0
		if displayedTotalCode > 0 {
			codePct = float64(ls.Stats.Code) * 100 / float64(displayedTotalCode)
		}
		row := []string{
			l,
			fmt.Sprintf("%d", ls.FileCount),
			fmt.Sprintf("%d", ls.Stats.Code),
			fmt.Sprintf("%d", ls.Stats.Comments),
			fmt.Sprintf("%d", ls.Stats.Blanks),
			fmt.Sprintf("%.1f%%", codePct),
			fmt.Sprintf("%d", ls.Stats.Code+ls.Stats.Comments+ls.Stats.Blanks),
		}
		if opts.WithFunctions {
			row = append(row, fmt.Sprintf("%d", ls.Functions))
		}
		if opts.WithStructs {
			row = append(row, fmt.Sprintf("%d", ls.Structs))
		}
		rows = append(rows, row)
	}

	if len(langs) > 0 { // TOTAL 行
		totalFiles := 0
		totalComments := 0
		totalBlanks := 0
		totalLines := 0
		totalFuncs := 0
		totalStructs := 0
		for _, l := range langs {
			ls := res.Languages[l]
			totalFiles += ls.FileCount
			totalComments += ls.Stats.Comments
			totalBlanks += ls.Stats.Blanks
			lines := ls.Stats.Code + ls.Stats.Comments + ls.Stats.Blanks
			totalLines += lines
			if opts.WithFunctions {
				totalFuncs += ls.Functions
			}
			if opts.WithStructs {
				totalStructs += ls.Structs
			}
		}
		totalRow := []string{
			"TOTAL",
			fmt.Sprintf("%d", totalFiles),
			fmt.Sprintf("%d", displayedTotalCode),
			fmt.Sprintf("%d", totalComments),
			fmt.Sprintf("%d", totalBlanks),
			"100.0%",
			fmt.Sprintf("%d", totalLines),
		}
		if opts.WithFunctions {
			totalRow = append(totalRow, fmt.Sprintf("%d", totalFuncs))
		}
		if opts.WithStructs {
			totalRow = append(totalRow, fmt.Sprintf("%d", totalStructs))
		}
		rows = append(rows, totalRow)
	}
	return headers, rows
}

// buildFileTable 构建文件明细表数据
func buildFileTable(res *models.AnalysisResult, opts InfoOptions) ([]string, [][]string) {
	files := res.Files
	if len(files) == 0 {
		return nil, nil
	}
	headers := []string{"path", "language", "code", "comments", "blanks", "lines"}
	if opts.WithFunctions {
		headers = append(headers, "funcs")
	}
	if opts.WithStructs {
		headers = append(headers, "structs")
	}
	sort.Slice(files, func(i, j int) bool {
		if files[i].Language == files[j].Language {
			return files[i].Path < files[j].Path
		}
		return files[i].Language < files[j].Language
	})
	rows := make([][]string, 0, len(files))
	for _, f := range files {
		row := []string{f.Path, f.Language, fmt.Sprintf("%d", f.Stats.Code), fmt.Sprintf("%d", f.Stats.Comments), fmt.Sprintf("%d", f.Stats.Blanks), fmt.Sprintf("%d", f.Stats.Code+f.Stats.Comments+f.Stats.Blanks)}
		if opts.WithFunctions || opts.WithStructs {
			if gd, ok := f.LanguageSpecific.(*models.GoDetails); ok && gd != nil {
				if opts.WithFunctions {
					row = append(row, fmt.Sprintf("%d", gd.Functions))
				}
				if opts.WithStructs {
					row = append(row, fmt.Sprintf("%d", gd.Structs))
				}
			} else {
				if opts.WithFunctions {
					row = append(row, "0")
				}
				if opts.WithStructs {
					row = append(row, "0")
				}
			}
		}
		rows = append(rows, row)
	}
	return headers, rows
}
