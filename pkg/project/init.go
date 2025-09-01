package project

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/yeisme/gocli/pkg/configs"
	"github.com/yeisme/gocli/pkg/context"
	"github.com/yeisme/gocli/pkg/style"
	"github.com/yeisme/gocli/pkg/utils/deps"
	newproject "github.com/yeisme/gocli/pkg/utils/newproject"
)

// InitOptions 定义了项目初始化的选项
type InitOptions struct {
	// 项目的初始化配置选项
	Project newproject.InitOptions

	// 下面是额外命令行选项
	// List of template files
	List bool
	// Format
	Format string
	// JSON
	JSON bool
	// YAML
	YAML bool
	// Plain 是否以纯文本格式输出
	Plain bool
	// Table 是否以表格格式输出
	Table bool

	// LangType 希望初始化的项目类型 （编程语言）
	LangType string

	// Template
	Template string

	// Force 是否强制覆盖已存在的文件 TODO 未完成
	Force bool
}

// ExecuteInitCommand 执行初始化命令
func ExecuteInitCommand(ctx *context.GocliContext, args []string, opts InitOptions, out io.Writer) error {
	initLanguageTemplate(&opts)

	var initGitIgnore []string

	if err := initFormatCfg(&opts); err != nil {
		return err
	}

	// 若 --type 不是显式 go/golang/空，但恰好是一个已注册的模板，且未显式提供 --template，则将其视为模板名并把语言归为 go
	if opts.Template == "" && opts.LangType != "" && opts.LangType != "go" && opts.LangType != "golang" {
		if _, ok := opts.Project.Go.Templates[opts.LangType]; ok {
			opts.Template = opts.LangType
			opts.LangType = "go"
		}
	}

	// 如果 Type 为空，默认为 go
	if opts.LangType == "" {
		opts.LangType = "go"
	}

	// 列出模板（抽象封装）
	if opts.List {
		return listTemplates(&opts, out)
	}

	switch opts.LangType {
	case "go", "golang":
		if err := ExecuteGoInitCommand(ctx, args, opts, out); err != nil {
			return err
		}
		initGitIgnore = append(initGitIgnore, "base-go")
	default:
		// 未知类型，当前版本暂不支持，给出提示
		return fmt.Errorf("unsupported project type: %s", opts.LangType)
	}

	_, err := opts.Project.ExecConfigInit(args, initGitIgnore)
	if err != nil {
		return err
	}

	return nil
}

// ExecuteGoInitCommand 执行 Go 语言项目初始化命令
func ExecuteGoInitCommand(_ *context.GocliContext, args []string, opts InitOptions, out io.Writer) error {
	// 1. 解析项目 module / 名称
	argsPath, err := newproject.NormalizeGoProjectName(args)
	if err != nil {
		return err
	}
	log.Debug().Str("path", argsPath).Msg("Initialize Go project")

	// 2. 确定目标目录：仅当显式提供 --dir 时创建/使用该目录；否则使用当前目录
	targetDir := strings.TrimSpace(opts.Project.Dir)
	log.Debug().Str("dir", targetDir).Msg("Target directory for project initialization")
	if targetDir == "" {
		targetDir = "." // 不创建新目录，go mod init 在当前目录执行
	} else {
		if mkErr := os.MkdirAll(targetDir, 0o755); mkErr != nil {
			return fmt.Errorf("create target dir %s failed: %w", targetDir, mkErr)
		}
	}

	// 3. 若指定模板，则获取并复制到目标目录
	if tmplName := strings.TrimSpace(opts.Template); tmplName != "" {
		fsys, tmplErr := newproject.GetGoTemplateFS(tmplName, opts.Project)
		if tmplErr != nil {
			return fmt.Errorf("load template %q failed: %w", tmplName, tmplErr)
		}
		// empty 类型会返回 nil，跳过复制
		if fsys != nil {
			if cpErr := copyTemplateIntoDir(fsys, targetDir, opts.Force); cpErr != nil {
				return fmt.Errorf("copy template %q failed: %w", tmplName, cpErr)
			}
			log.Debug().Str("template", tmplName).Str("dir", targetDir).Msg("template copied")
		}
	}

	// 4. 运行 go mod init (若 go.mod 已存在则跳过)
	if _, statErr := os.Stat(filepath.Join(targetDir, "go.mod")); statErr == nil {
		log.Debug().Str("dir", targetDir).Msg("go.mod already exists, skip go mod init")
		return nil
	}

	outbuf, gmErr := deps.RunGoModInit(argsPath, targetDir)
	if gmErr != nil {
		return gmErr
	}
	_, _ = out.Write([]byte(outbuf))
	return nil
}

// copyTemplateIntoDir 将模板文件系统复制到目标目录
func copyTemplateIntoDir(fsys fs.FS, target string, force bool) error {
	return newproject.CopyTemplateFSToDir(fsys, target, force)
}

func initLanguageTemplate(opts *InitOptions) {
	// 内置，拥有最高优先级
	opts.Project.Go = newproject.NewGoInitOptions()

	// 构建搜索路径：
	// 1. 当前工作目录向上回溯直到根（保证在子目录下执行仍能找到仓库根的 .gocli/template）
	// 2. 模块根（通过 go env GOMOD 获取）及其 configs 目录
	// 3. 原有的配置搜索路径（HOME 等）
	var searchPaths []string

	// 1. 当前工作目录向上回溯
	if cwd, err := os.Getwd(); err == nil {
		cur := cwd
		for {
			searchPaths = append(searchPaths, cur, filepath.Join(cur, "configs"))
			parent := filepath.Dir(cur)
			if parent == cur { // 到根目录
				break
			}
			cur = parent
		}
	}

	// 2. 模块根
	if moduleRoot := configs.GetModuleRoot(""); moduleRoot != "" {
		searchPaths = append([]string{moduleRoot, filepath.Join(moduleRoot, "configs")}, searchPaths...)
	}

	// 3. 追加原有全局搜索路径
	searchPaths = append(searchPaths, configs.GetConfigSearchPaths()...)

	// 去重，保持前面优先级
	seenSearch := map[string]struct{}{}
	uniq := make([]string, 0, len(searchPaths))
	for _, p := range searchPaths {
		if p == "" {
			continue
		}
		p = os.ExpandEnv(p)
		if _, ok := seenSearch[p]; ok {
			continue
		}
		seenSearch[p] = struct{}{}
		uniq = append(uniq, p)
	}
	searchPaths = uniq

	// 候选模板目录形式
	candidateSuffixes := []string{
		filepath.Join(".gocli", "template"),
		"template",
	}

	seen := map[string]bool{}

	for _, p := range searchPaths {
		for _, suf := range candidateSuffixes {
			full := filepath.Join(p, suf)
			if seen[full] {
				continue
			}
			seen[full] = true

			st, err := os.Stat(full)
			if err != nil || !st.IsDir() {
				continue
			}

			entries, err := os.ReadDir(full)
			if err != nil {
				continue
			}

			// process entries in this template directory
			for _, e := range entries {
				name := e.Name()
				innerFull := filepath.Join(full, name)

				if e.IsDir() {
					// 将子目录视作 file_system 模板
					_ = newproject.AddGoTemplateToOptions(&opts.Project, name, innerFull, "file_system")
					continue
				}

				// 识别 template 描述文件
				lname := strings.ToLower(name)
				if lname == "template.json" || lname == "template.yaml" || lname == "template.yml" {
					b, err := os.ReadFile(innerFull)
					if err != nil {
						continue
					}

					var m map[string]struct {
						Path     string `json:"path"`
						Type     string `json:"type"`
						Language string `json:"language,omitempty"`
					}
					// yaml 由于 yaml 覆盖了 json 的类型，通常使用 yaml 也能解析 json
					if err := yaml.Unmarshal(b, &m); err == nil {
						for k, v := range m {
							p := v.Path
							if p == "" {
								p = filepath.Join(full, k)
							}
							t := v.Type
							if t == "" {
								t = "file_system"
							}
							if err := newproject.AddGoTemplateToOptions(&opts.Project, k, p, t); err != nil {
								log.Warn().Err(err).Str("template", k).Msg("add template failed")
							} else {
								// 覆盖 language (若提供)
								if v.Language != "" {
									tpl := opts.Project.Go.Templates[k]
									tpl.Language = v.Language
									opts.Project.Go.Templates[k] = tpl
								}
							}
						}
					}
				}
			}
		}
	}
	log.Debug().Int("count", len(opts.Project.Go.Templates)).Msg("Go templates loaded")
}

func initFormatCfg(opts *InitOptions) error {
	// format / json / yaml 不能同时设置
	cut := 0
	if opts.Format != "" {
		cut++
	}
	if opts.JSON {
		cut++
	}
	if opts.YAML {
		cut++
	}
	if opts.Plain {
		cut++
	}
	if opts.Table {
		cut++
	}
	if cut > 1 {
		return fmt.Errorf("format / json / yaml / plain / table cannot be set at the same time")
	}
	if cut == 0 {
		// 默认 plain
		opts.Plain = true
	}

	if opts.Format != "" {
		switch opts.Format {
		case "json":
			opts.JSON = true
		case "yaml":
			opts.YAML = true
		case "plain":
			opts.Plain = true
		case "table":
			opts.Table = true
		default:
			return fmt.Errorf("unknown format: %s", opts.Format)
		}
	}

	return nil
}

// listTemplates 根据当前语言类型输出模板列表
// 支持三种输出：JSON / YAML / Plain
func listTemplates(opts *InitOptions, out io.Writer) error {
	lang := opts.LangType
	switch lang {
	case "go", "golang":
		return listGoTemplates(opts, out)
	default:
		return fmt.Errorf("template listing for language %q not implemented", lang)
	}
}

// listGoTemplates 输出 Go 模板
func listGoTemplates(opts *InitOptions, out io.Writer) error {
	tm := opts.Project.Go.Templates
	if opts.JSON {
		if _, err := fmt.Fprintf(out, "Available templates for language %q:\n", opts.LangType); err != nil {
			return fmt.Errorf("write output failed: %w", err)
		}
		b, err := json.MarshalIndent(tm, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal templates failed: %v", err)
		}
		_ = style.PrintJSON(out, b)
		return nil
	}
	if opts.YAML {
		if _, err := fmt.Fprintf(out, "Available templates for language %q:\n", opts.LangType); err != nil {
			return fmt.Errorf("write output failed: %w", err)
		}
		b, err := yaml.Marshal(tm)
		if err != nil {
			return fmt.Errorf("marshal templates failed: %v", err)
		}
		_ = style.PrintYAML(out, b)
		return nil
	}
	if opts.Table {
		if _, err := fmt.Fprintf(out, "Available templates for language %q:\n", opts.LangType); err != nil {
			return fmt.Errorf("write output failed: %w", err)
		}
		// 构建表格数据
		headers := []string{"Name", "Language", "Type", "Path"}
		var rows [][]string

		// 按名称排序
		names := make([]string, 0, len(tm))
		for k := range tm {
			names = append(names, k)
		}
		sort.Strings(names)

		for _, name := range names {
			t := tm[name]
			lang := t.Language
			if lang == "" {
				lang = "go"
			}
			rows = append(rows, []string{name, lang, t.Type, t.Path})
		}

		return style.PrintTable(out, headers, rows, 0)
	}
	// Plain 输出：按名称排序
	if _, err := fmt.Fprintf(out, "Available templates for language %q:\n", opts.LangType); err != nil {
		return fmt.Errorf("write output failed: %w", err)
	}
	names := make([]string, 0, len(tm))
	for k := range tm {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, name := range names {
		t := tm[name]
		lang := t.Language
		if lang == "" {
			lang = "go"
		}
		if _, err := fmt.Fprintf(out, "  - %s\t(lang=%s type=%s path=%s)\n", name, lang, t.Type, t.Path); err != nil {
			return fmt.Errorf("write output failed: %w", err)
		}
	}
	return nil
}
