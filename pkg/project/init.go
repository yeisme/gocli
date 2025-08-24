package project

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
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

	// Type
	Type string
}

// ExecuteInitCommand 执行初始化命令
func ExecuteInitCommand(ctx *context.GocliContext, args []string, opts InitOptions, out io.Writer) error {
	initLanguageTemplate(&opts)

	// format / json / yaml 不能同时设置
	if opts.Format != "" && (opts.JSON || opts.YAML) {
		return fmt.Errorf("format / json / yaml cannot be set at the same time")
	}

	if opts.Format != "" {
		switch opts.Format {
		case "json":
			opts.JSON = true
		case "yaml":
			opts.YAML = true
		default:
			return fmt.Errorf("unknown format: %s", opts.Format)
		}
	}

	// 列出模板
	if opts.List {
		if opts.JSON {
			b, err := json.MarshalIndent(opts.Project.Go.Templates, "", "  ")
			if err != nil {
				return fmt.Errorf("marshal templates failed: %v", err)
			}
			_ = style.PrintJSON(out, b)
			return nil
		} else if opts.YAML {
			b, err := yaml.Marshal(opts.Project.Go.Templates)
			if err != nil {
				return fmt.Errorf("marshal templates failed: %v", err)
			}
			_ = style.PrintYAML(out, b)
			return nil
		}
	}

	switch opts.Type {
	case "go", "golang", "":
		err := ExecuteGoInitCommand(ctx, args, opts, out)
		if err != nil {
			return err
		}
	}

	return nil
}

// ExecuteGoInitCommand 执行 Go 语言项目初始化命令
func ExecuteGoInitCommand(_ *context.GocliContext, args []string, opts InitOptions, out io.Writer) error {
	argsPath, err := newproject.NormalizeGoProjectName(args)
	if err != nil {
		return err
	}
	log.Debug().Str("path", argsPath).Msg("Initialize Go project")

	outbuf, err := deps.RunGoModInit(argsPath, opts.Project.Dir)
	if err != nil {
		return err
	}
	_, _ = out.Write([]byte(outbuf))

	return nil
}

func initLanguageTemplate(opts *InitOptions) {
	// 内置，拥有最高优先级
	opts.Project.Go = newproject.NewGoInitOptions()

	// 复用配置包的搜索路径
	searchPaths := configs.GetConfigSearchPaths()

	// 候选模板目录形式
	candidateSuffixes := []string{
		filepath.Join(".gocli", "template"),
		filepath.Join("gocli", "template"),
		"template",
	}

	seen := map[string]bool{}

	for _, p := range searchPaths {
		p = os.ExpandEnv(p)
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
						Path string `json:"path" yaml:"path"`
						Type string `json:"type" yaml:"type"`
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
								fmt.Printf("add template %s failed: %v\n", k, err)
							}
						}
					}
				}
			}
		}
	}
}
