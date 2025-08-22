package project

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/yeisme/gocli/pkg/configs"
	"github.com/yeisme/gocli/pkg/context"
	"github.com/yeisme/gocli/pkg/style"
	"github.com/yeisme/gocli/pkg/utils/doc"
)

// DocOptions 是文档生成的配置选项，使用 doc.Options 的别名
type (
	DocOptions = doc.Options
)

// RunDoc 执行文档生成
func RunDoc(ctx *context.GocliContext, opts DocOptions, out io.Writer, args []string) error {
	// args 需要校验，至少有一个参数
	if len(args) < 1 {
		return fmt.Errorf("doc: at least one argument is required")
	}

	// 处理输出目标（可能是文件），prepareOutput 返回最终的 writer、可选的关闭函数和 error
	out, closeOut, err := prepareOutput(&opts, out)
	if err != nil {
		return err
	}
	if closeOut != nil {
		defer closeOut()
	}

	// 判断是否标准库/三方库的 import path（非文件系统绝对/相对路径）
	isGoStandardPackage := func(ctx *context.GocliContext, importPath string) bool {
		if importPath == "" {
			return false
		}
		// 标准库一般不包含点号，并且使用正斜杠分隔
		if strings.Contains(importPath, ".") {
			return false
		}
		// 构造 GOROOT/src/<importPath> 目录判断是否存在
		dir := filepath.Join(ctx.Config.Env.GoRoot, "src", filepath.FromSlash(importPath))
		return isDirectory(dir)
	}
	isGoThirdPartyPackage := func(ctx *context.GocliContext, importPath string) bool {
		if importPath == "" {
			return false
		}
		// 三方库 import path 一般包含域名（带点号），如 github.com/...
		if !strings.Contains(importPath, ".") {
			return false
		}
		if dir, ok := resolveThirdPartyPackageDir(ctx.Config.Env.GoModCache, importPath); ok {
			return isDirectory(dir)
		}
		return false
	}

	// 在没有 opts.Root 的情况下，使用 Go module 根目录
	root := configs.GetModuleRoot(ctx.Config.Env.GoMod)

	for _, path := range args {
		// 为每个文件使用 opts 的本地拷贝，避免在循环中恢复原值或影响其他迭代
		cur := opts

		// 优先识别 import path（避免把 "fmt" 等误当作当前目录下的相对路径）
		if looksLikeImportPath(path) {
			// 标准库
			if isGoStandardPackage(ctx, path) {
				cur.Mode = doc.ModeGodoc
				path = filepath.Join(ctx.Config.Env.GoRoot, "src", filepath.FromSlash(path))
				log.Debug().Str("importPath", path).Msg("RunDoc: resolved standard library package directory")
			} else if isGoThirdPartyPackage(ctx, path) { // 三方库
				cur.Mode = doc.ModeGodoc
				if dir, ok := resolveThirdPartyPackageDir(ctx.Config.Env.GoModCache, path); ok {
					path = dir
					root = filepath.Dir(dir) // 更新 root 为三方库所在目录
					log.Debug().Str("importPath", path).Msg("RunDoc: resolved third-party package directory")
				}
			}
		}

		// 若仍然不是 import path 解析成功的目录，则按原有逻辑解析文件系统路径
		if !filepath.IsAbs(path) && !isDirectory(path) && filepath.Ext(path) != ".go" && !isMarkdownExt(path) {
			// 先把相对路径解析为基于当前工作目录的绝对路径，
			// 这样在子目录中执行 `gocli project doc .` 能正确解析到当前目录下的包。
			if wd, err := os.Getwd(); err == nil {
				path = filepath.Join(wd, path)
			} else {
				// 如果获取当前工作目录失败，回退到 module root（保留旧行为的安全回退）
				path = filepath.Join(root, path)
			}
		}

		// 使用绝对路径处理，避免在不同操作系统下路径解析不一致的问题
		if !filepath.IsAbs(path) {
			if abs, err := filepath.Abs(path); err == nil {
				path = abs
			}
		}

		if isMarkdownExt(path) {
			cur.Mode = doc.ModeMarkdown
		}
		// 判断是否为目录（path 已尽量为绝对路径）
		if isDirectory(path) {
			cur.Mode = doc.ModeGodoc
		}
		// 当输入一个 .go 后缀文件，提取所在目录作为包文档
		if filepath.Ext(path) == ".go" {
			cur.Mode = doc.ModeGodoc
			path = filepath.Dir(path)
		}

		str, genErr := doc.GetDoc(log, cur, root, path)
		if genErr != nil {
			return fmt.Errorf("doc: failed to generate documentation %w", genErr)
		}

		switch cur.Mode {
		case doc.ModeGodoc:
			if err := doc.RenderGodoc(out, str, cur); err != nil {
				return fmt.Errorf("doc: failed to render godoc for %q: %w", path, err)
			}
		case doc.ModeMarkdown:
			if err := style.RenderMarkdown(out, str, cur.Width, cur.Theme); err != nil {
				return fmt.Errorf("doc: failed to render markdown for %q: %w", path, err)
			}
		default:
			return fmt.Errorf("doc: unsupported mode %v for %q", cur.Mode, path)
		}
	}
	return nil
}

func isMarkdownExt(path string) bool {
	lower := strings.ToLower(path)
	return strings.HasSuffix(lower, ".md") || strings.HasSuffix(lower, ".markdown")
}

func isDirectory(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// looksLikeImportPath 粗略判断一个参数是否像 import path
func looksLikeImportPath(s string) bool {
	if s == "" {
		return false
	}
	// 文件系统特征：绝对路径/包含反斜杠/以 . 开头 的相对路径
	if filepath.IsAbs(s) || strings.Contains(s, "\\") || strings.HasPrefix(s, ".") {
		return false
	}
	// import path 使用正斜杠，且不能是单个 path.Base 指向当前目录的占位
	return strings.Contains(s, "/") || (!strings.Contains(s, ".") && s != "." && s != "..")
}

// resolveThirdPartyPackageDir 尝试在 GOMODCACHE 下解析三方库 import path 对应目录
// 例如：github.com/charmbracelet/glamour[/sub] -> $GOMODCACHE/github.com/charmbracelet/glamour@<ver>[/sub]
func resolveThirdPartyPackageDir(goModCache, importPath string) (string, bool) {
	if goModCache == "" || importPath == "" {
		return "", false
	}
	segs := strings.Split(importPath, "/")
	// 从最长路径逐级回退，找到存在的模块根（带 @version）
	for i := len(segs); i >= 1; i-- {
		module := strings.Join(segs[:i], "/")
		sub := strings.Join(segs[i:], "/")
		// 模块根目录形如 <module>@*
		prefix := filepath.Join(goModCache, filepath.FromSlash(module)) + "@"
		matches, _ := filepath.Glob(prefix + "*")
		if len(matches) == 0 {
			continue
		}
		best := pickMostRecentDir(matches)
		if best == "" {
			continue
		}
		dir := best
		if sub != "" {
			dir = filepath.Join(best, filepath.FromSlash(sub))
		}
		if isDirectory(dir) {
			return dir, true
		}
	}
	return "", false
}

// pickMostRecentDir 在多个候选模块版本目录中选择最近修改的目录
func pickMostRecentDir(paths []string) string {
	var best string
	var bestModTime int64
	for _, p := range paths {
		fi, err := os.Stat(p)
		if err != nil || !fi.IsDir() {
			continue
		}
		mt := fi.ModTime().UnixNano()
		if mt > bestModTime {
			bestModTime = mt
			best = p
		}
	}
	return best
}

// prepareOutput 根据 opts.Output 决定最终的输出 io.Writer，并返回一个可选的关闭函数（当输出为文件时）
// 返回值: (writer, closeFunc, error)
func prepareOutput(opts *DocOptions, defaultOut io.Writer) (io.Writer, func(), error) {
	if opts.Output == "" {
		return defaultOut, nil, nil
	}
	// TODO 使用 switch 判断 opts.Output 类型进行不同的分支

	file, err := os.Create(opts.Output)
	if err != nil {
		return nil, nil, fmt.Errorf("doc: failed to create output file %q: %w", opts.Output, err)
	}
	closeFn := func() {
		if err := file.Close(); err != nil {
			log.Error().Err(err).Msg("failed to close output file")
		}
	}
	return file, closeFn, nil
}
