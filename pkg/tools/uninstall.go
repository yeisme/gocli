package tools

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// UninstallCommandOptions 定义卸载命令的所有选项和上下文
//
// 说明（中文注释）：
//   - Args: 要卸载的工具名称列表
//   - Yes: 是否跳过交互提示（等同于 --yes，自动确认删除）
//   - Dry: 是否为模拟运行（--dry），不实际删除文件，仅打印会删除的项
//   - Fuzzy: 是否模糊匹配工具名称（例如包含关系）
//   - All: 是否删除所有匹配位置（比如 GOPATH/bin、~/.gocli/tools 等），
//     注意：All 与交互行为结合时会逐文件提示（除非 --yes）
//   - Verbose / Quiet: 输出控制
//   - GoCLIToolsPath: 可选的 gocli 工具目录覆盖（用于搜索）
//   - ToolsConfigDir: 搜索工具配置文件的目录列表
//   - Input: 用于测试或重定向输入的 io.Reader；若为 nil，使用 os.Stdin
type UninstallCommandOptions struct {
	Args  []string
	Yes   bool
	Dry   bool
	Fuzzy bool
	All   bool

	Verbose bool
	Quiet   bool

	GoCLIToolsPath string
	ToolsConfigDir []string

	// Input reader for interactive prompts; if nil, os.Stdin is used
	Input io.Reader
}

// ExecuteUninstallCommand 执行工具卸载命令的封装函数
// 它会基于传入的选项查找已安装的二进制并按交互/批量规则删除它们
// ExecuteUninstallCommand 执行卸载操作的主入口
//
//  1. 根据 opts.Args（工具名）查找候选的工具定义（通过配置或已知工具列表）
//  2. 对每个候选的 binary name，扫描可能的目录（GOPATH/bin、gocli 工具目录、用户目录）
//     以收集实际存在的可执行文件路径（可能存在多个同名二进制在不同目录）
//  3. 根据 opts.All / opts.Yes / opts.Dry 的组合决定交互方式和实际删除行为：
//     - 如果 --all 且同时提供 --yes：直接删除找到的所有匹配项（不可逆）
//     - 如果 --all 且未提供 --yes：对每个匹配的文件逐一交互确认（更安全）
//     - 如果未指定 --all：默认行为为逐文件交互（除非 --yes）
//     - 如果 --dry：不做实际删除，仅打印将要删除的路径
//
// 该函数同时处理去重（相同 binary name 不重复询问）以及在删除成功后清理内部工具缓存
func ExecuteUninstallCommand(opts UninstallCommandOptions, out io.Writer) error {
	// 基本参数校验：至少需要一个工具名
	if len(opts.Args) == 0 {
		return fmt.Errorf("missing tool name")
	}

	var reader *bufio.Reader
	if opts.Input == nil {
		reader = bufio.NewReader(os.Stdin)
	} else {
		reader = bufio.NewReader(opts.Input)
	}

	// 对每个用户请求的工具名进行处理
	for _, name := range opts.Args {
		// 先从配置/已注册工具中找候选（用于确定 binary name）
		candidates := findCandidatesForName(name, opts)
		if len(candidates) == 0 {
			fmt.Fprintf(out, "no installed binaries found for: %s\n", name)
			continue
		}

		// 去重：同一个 binary name 只处理一次
		seen := map[string]struct{}{}
		for _, c := range candidates {
			bn := c.BinaryName
			if bn == "" {
				bn = name
			}
			key := strings.ToLower(bn)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}

			exeName := bn

			// 循环查找该 exe 在所有可能目录中的存在（因为会有多个目录）
			for {
				matches := findMatchesForExe(exeName, opts)
				if len(matches) == 0 {
					// 仅在第一次找不到时提示，避免重复噪音
					fmt.Fprintf(out, "no installed binary file found for '%s'\n", exeName)
					break
				}

				// 当用户指定 --all 时，行为如下：
				// - 若同时提供 --yes，则直接删除所有匹配（removePaths 会处理 dry-run）
				// - 若未提供 --yes，则对每个匹配路径单独询问（提高安全性，防止误删）
				if opts.All {
					// 先打印出所有找到的路径，帮助用户决策
					fmt.Fprintln(out, "matching files:")
					for _, p := range matches {
						fmt.Fprintf(out, "  %s\n", p)
					}

					if opts.Yes {
						// 非交互：一次性删除所有匹配路径（遵守 --dry）
						removePaths(matches, opts.Dry, out)
						// 处理完当前找到的匹配后，退出内部循环，避免重复提示
						break
					}

					// --all 且没有 --yes：对每个文件逐一交互确认并删除
					for _, p := range matches {
						// 对单个路径进行确认，使用 confirmYes 读取用户输入
						if !confirmYes(reader, out, fmt.Sprintf("Delete %s? [y/N]: ", p)) {
							// 用户选择否：记录 skipped 并继续下一个
							fmt.Fprintf(out, "skipped: %s\n", p)
							continue
						}
						// 用户确认删除：检查 dry-run
						if opts.Dry {
							fmt.Fprintf(out, "[dry-run] would remove: %s\n", p)
							continue
						}
						// 执行删除操作并在成功时清除缓存
						if err := os.Remove(p); err != nil {
							fmt.Fprintf(out, "failed to remove %s: %v\n", p, err)
						} else {
							fmt.Fprintf(out, "removed: %s\n", p)
							ClearToolsCache()
						}
					}
					// 处理完当前找到的匹配后，退出内部循环，避免重复提示
					break
				}

				// 非 --all 情况：默认按文件逐一提示（除非 --yes）
				for _, p := range matches {
					if !opts.Yes {
						// 未指定 --yes 时询问用户是否删除当前文件
						if !confirmYes(reader, out, fmt.Sprintf("Delete %s? [y/N]: ", p)) {
							fmt.Fprintf(out, "skipped: %s\n", p)
							continue
						}
					}
					// 支持 dry-run 模式
					if opts.Dry {
						fmt.Fprintf(out, "[dry-run] would remove: %s\n", p)
						continue
					}
					// 执行删除并报告结果
					if err := os.Remove(p); err != nil {
						fmt.Fprintf(out, "failed to remove %s: %v\n", p, err)
					} else {
						fmt.Fprintf(out, "removed: %s\n", p)
						ClearToolsCache()
					}
				}
				break
			}
		}
	}
	return nil
}

// findCandidatesForName 查找并返回与给定名称相关的候选工具信息集合
//
// 这个函数会优先尝试从配置/工具索引中查找匹配的条目（SearchTools），
// 并使用工具列表（FindTools）来将配置条目映射到已知的工具名称（用于确定 binary name）
// 若找不到配置条目，则基于 opts.Fuzzy 决定是否进行模糊匹配或精确匹配
func findCandidatesForName(name string, opts UninstallCommandOptions) []InstallToolsInfo {
	var candidates []InstallToolsInfo
	// 先尝试从工具配置中查找（例如 docs/config_schema 等）
	bi := SearchTools(name, opts.ToolsConfigDir)
	if bi != nil {
		bn := bi.BinaryName
		if bn == "" {
			bn = name
		}
		// FindTools 返回已注册/已知的工具列表，用于匹配名称
		tools := FindTools(opts.Verbose, opts.GoCLIToolsPath)
		for _, t := range tools {
			if strings.EqualFold(t.Name, bn) {
				candidates = append(candidates, InstallToolsInfo{Name: t.Name, CloneURL: "", URL: "", BinaryName: t.Name})
			}
		}
	}

	// 如果通过配置没有找到候选，则直接在已知工具列表中查找（精确或模糊）
	if len(candidates) == 0 {
		tools := FindTools(opts.Verbose, opts.GoCLIToolsPath)
		lname := strings.ToLower(name)
		if opts.Fuzzy {
			// 模糊匹配：名称包含目标字符串即可
			for _, t := range tools {
				if strings.Contains(strings.ToLower(t.Name), lname) {
					candidates = append(candidates, InstallToolsInfo{Name: t.Name, BinaryName: t.Name})
				}
			}
		} else {
			// 精确匹配（不区分大小写）
			for _, t := range tools {
				if strings.EqualFold(t.Name, name) {
					candidates = append(candidates, InstallToolsInfo{Name: t.Name, BinaryName: t.Name})
				}
			}
		}
	}
	return candidates
}

// findMatchesForExe 返回与 exeName 匹配的已安装可执行文件路径集合
//
// FindTools 会按名称去重，但为了发现同名二进制在不同目录存在的情况，
// 这里直接扫描可能的目录集合（例如 GOPATH/bin、gocli 自身的 tools 目录、用户目录）
func findMatchesForExe(exeName string, opts UninstallCommandOptions) []string {
	// FindTools does de-duplicate by name; to discover multiple same-named
	// binaries in different directories, scan candidate directories directly.
	var matches []string

	// 收集候选目录：GOPATH/bin 条目 + 配置的 gocli tools 路径 + 用户 ~/.gocli/tools
	dirs := []struct {
		path   string
		source toolSourceType
	}{}

	for _, gp := range getGoPaths() {
		if gp == "" {
			continue
		}
		dirs = append(dirs, struct {
			path   string
			source toolSourceType
		}{path: joinPath(gp, "bin"), source: goPath})
	}
	if p := getUserToolsDir(opts.GoCLIToolsPath); p != "" {
		dirs = append(dirs, struct {
			path   string
			source toolSourceType
		}{path: p, source: goCliPath})
	}
	if p := getUserToolsDir(""); p != "" {
		dirs = append(dirs, struct {
			path   string
			source toolSourceType
		}{path: p, source: goUserCliPath})
	}

	// 防止重复扫描相同目录
	seenDirs := map[string]struct{}{}
	for _, d := range dirs {
		if _, ok := seenDirs[d.path]; ok {
			continue
		}
		seenDirs[d.path] = struct{}{}
		list := listExecutablesInDir(d.path, d.source)
		for _, ti := range list {
			if strings.EqualFold(ti.Name, exeName) {
				matches = append(matches, ti.Path)
			}
		}
	}
	return matches
}

// confirmYes 从 reader 读取用户输入并返回是否确认（用户输入 y 或 yes 为真）
//
// 该函数对输入做简单清洗（去除空白并转小写），仅接受 "y" 和 "yes" 为肯定
func confirmYes(reader *bufio.Reader, out io.Writer, prompt string) bool {
	fmt.Fprint(out, prompt)
	txt, _ := reader.ReadString('\n')
	txt = strings.TrimSpace(strings.ToLower(txt))
	return txt == "y" || txt == "yes"
}

// removePaths 删除给定路径集合，遵守 dry-run 模式，并在每次成功删除后触发缓存清理
//
// 此 helper 用于一次性删除多个路径（例如在 --all --yes 场景下），
// 会打印每个操作的结果若 dry 为 true，则不会执行实际删除
func removePaths(paths []string, dry bool, out io.Writer) {
	for _, p := range paths {
		if dry {
			fmt.Fprintf(out, "[dry-run] would remove: %s\n", p)
			continue
		}
		if err := os.Remove(p); err != nil {
			fmt.Fprintf(out, "failed to remove %s: %v\n", p, err)
		} else {
			fmt.Fprintf(out, "removed: %s\n", p)
			ClearToolsCache()
		}
	}
}
