package tools

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/yeisme/gocli/pkg/configs"
	"github.com/yeisme/gocli/pkg/style"
)

// BatchInstallConfiguredTools installs tools from a Config (deps and global)
func BatchInstallConfiguredTools(cfg *configs.Config, envFlags []string, verbose bool) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}

	depsPath := cfg.Tools.GoCLIToolsPath
	if strings.TrimSpace(depsPath) == "" {
		home, _ := os.UserHomeDir()
		depsPath = filepath.Join(home, ".gocli", "tools")
	}
	globalPath := filepath.Join(mustUserHome(), ".gocli", "tools")

	total := 0
	failed := 0

	// install deps
	for _, t := range cfg.Tools.Deps {
		ok, err := installSingleConfiguredTool(t, depsPath, "dep", envFlags, verbose, cfg.Tools.ToolsConfigDir)
		if err != nil {
			failed++
		}
		if ok {
			total++
		}
	}

	// install globals
	for _, t := range cfg.Tools.Global {
		ok, err := installSingleConfiguredTool(t, globalPath, "global", envFlags, verbose, cfg.Tools.ToolsConfigDir)
		if err != nil {
			failed++
		}
		if ok {
			total++
		}
	}

	if total == 0 && failed == 0 {
		// note: logger not available in package; return a nil error but caller may log
		return nil
	}
	if failed > 0 {
		return fmt.Errorf("%d tool(s) failed", failed)
	}
	return nil
}

// BatchInstallConfiguredGlobalTools installs only global tools to ~/.gocli/tools
func BatchInstallConfiguredGlobalTools(cfg *configs.Config, envFlags []string, verbose bool) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	targetPath := filepath.Join(mustUserHome(), ".gocli", "tools")
	total := 0
	failed := 0
	for _, t := range cfg.Tools.Global {
		ok, err := installSingleConfiguredTool(t, targetPath, "global", envFlags, verbose, cfg.Tools.ToolsConfigDir)
		if err != nil {
			failed++
		}
		if ok {
			total++
		}
	}
	if total == 0 && failed == 0 {
		return nil
	}
	if failed > 0 {
		return fmt.Errorf("%d tool(s) failed", failed)
	}
	return nil
}

// installSingleConfiguredTool attempts to install a single configs.Tool.
// It will try to resolve a matching InstallToolsInfo via ResolveTool using
// various candidate keys (module base name, full module, cmd, clone url).
// If a matching InstallToolsInfo is found, its fields are used to construct
// InstallOptions; otherwise the legacy configs.Tool fields are used.
func installSingleConfiguredTool(t configs.Tool, targetPath, category string, envFlags []string, verbose bool, configDirs []string) (bool, error) {
	// 合并环境变量（用户传入的 envFlags 优先，然后是工具配置内的 env）
	envMerged := mergeEnv(envFlags, t.Env)

	// 生成候选 key 列表，用于在内置/用户工具映射中查找
	candidates := buildCandidatesFromTool(t)

	// 尝试解析到 InstallToolsInfo（优先使用映射定义）
	bi := resolveInstallInfo(candidates, configDirs)
	if bi != nil {
		// 如果有平台约束，先检查是否可安装
		if ok, reason := checkPlatformCompatibility(bi); !ok {
			fmt.Printf("skipped %s: %s\n", bi.Name, reason)
			return false, nil
		}
		// 合并最终环境变量：先外部合并 envMerged，再追加映射内 env
		envFinal := mergeEnv(envMerged, bi.Env)
		return installFromInfo(bi, targetPath, category, envFinal, verbose)
	}

	// 未命中映射，回退到 legacy 行为（使用 configs.Tool 的字段）
	return installFromConfigTool(t, targetPath, category, envMerged, verbose)
}

// mergeEnv 合并两个环境变量切片，返回新的切片（不修改原切片）
// 简单的实现为把第二个切片追加到第一个之后，保留重复项
func mergeEnv(base, extra []string) []string {
	out := append([]string{}, base...)
	if len(extra) > 0 {
		out = append(out, extra...)
	}
	return out
}

// buildCandidatesFromTool 从 configs.Tool 中提取可能用于在映射中查找的候选字符串
// 包括：完整 module（带 @version）、module 的最后一段（不含 @version）、legacy cmd、clone url
func buildCandidatesFromTool(t configs.Tool) []string {
	var candidates []string
	if strings.TrimSpace(t.Module) != "" {
		candidates = append(candidates, t.Module)
		s := t.Module
		if at := strings.IndexByte(s, '@'); at >= 0 {
			s = s[:at]
		}
		if last := strings.LastIndexByte(s, '/'); last >= 0 && last+1 < len(s) {
			candidates = append(candidates, s[last+1:])
		}
	}
	if strings.TrimSpace(t.Cmd) != "" {
		candidates = append(candidates, t.Cmd)
	}
	if strings.TrimSpace(t.CloneURL) != "" {
		candidates = append(candidates, t.CloneURL)
	}
	return candidates
}

// resolveInstallInfo 使用候选列表尝试在 BuiltinTools（及用户加载的映射）中找到匹配项
// 返回第一个匹配到的 InstallToolsInfo 指针（副本）
func resolveInstallInfo(candidates []string, configDirs []string) *InstallToolsInfo {
	for _, c := range candidates {
		if strings.TrimSpace(c) == "" {
			continue
		}
		if b, _ := ResolveTool(c, configDirs); b != nil {
			return b
		}
	}
	return nil
}

// checkPlatformCompatibility 检查 InstallToolsInfo 中的 InstallType 是否与当前运行平台兼容
// 返回 (true, "") 表示可安装；否则返回 (false, reason)
func checkPlatformCompatibility(bi *InstallToolsInfo) (bool, string) {
	if bi == nil || bi.InstallType == nil {
		return true, ""
	}
	if bi.InstallType.OS != "" && bi.InstallType.OS != runtime.GOOS {
		return false, "incompatible os " + bi.InstallType.OS
	}
	if bi.InstallType.Arch != "" && bi.InstallType.Arch != runtime.GOARCH {
		return false, "incompatible arch " + bi.InstallType.Arch
	}
	return true, ""
}

// installFromInfo 使用 InstallToolsInfo 中的信息进行安装（支持 go install 或 clone 构建）
func installFromInfo(bi *InstallToolsInfo, targetPath, category string, env []string, verbose bool) (bool, error) {
	// prefer URL (go install) over CloneURL
	if strings.TrimSpace(bi.URL) != "" {
		res, err := InstallTool(InstallOptions{
			Spec:         bi.URL,
			Path:         targetPath,
			Env:          env,
			Verbose:      verbose,
			ReleaseBuild: false,
			DebugBuild:   false,
			BinaryName:   bi.BinaryName,
		})
		PrintInstallOutput(res.Output, err, verbose)
		if err != nil {
			return false, err
		}
		if res.InstallDir != "" {
			fmt.Printf("installed %s(go): %s -> %s\n", category, bi.URL, filepath.Clean(res.InstallDir))
		}
		return true, nil
	}

	if strings.TrimSpace(bi.CloneURL) != "" {
		res, err := InstallTool(InstallOptions{
			CloneURL:          bi.CloneURL,
			BuildMethod:       bi.Build,
			MakeTarget:        bi.MakeTarget,
			WorkDir:           bi.WorkDir,
			BinDirs:           bi.BinDirs,
			Env:               env,
			GoreleaserConfig:  bi.GoreleaserConfig,
			BinaryName:        bi.BinaryName,
			RecurseSubmodules: false,
			ReleaseBuild:      false,
			DebugBuild:        false,
			Path:              targetPath,
			Verbose:           verbose,
		})
		PrintInstallOutput(res.Output, err, verbose)
		if err != nil {
			return false, err
		}
		if res.InstallDir != "" {
			fmt.Printf("installed %s(clone): %s -> %s\n", category, bi.CloneURL, filepath.Clean(res.InstallDir))
		}
		return true, nil
	}
	return false, fmt.Errorf("no install method for tool %s", bi.Name)
}

// installFromConfigTool 按照旧的 configs.Tool 字段进行安装
func installFromConfigTool(t configs.Tool, targetPath, category string, env []string, verbose bool) (bool, error) {
	ttype := strings.ToLower(strings.TrimSpace(t.Type))
	switch ttype {
	case "", "go":
		spec := strings.TrimSpace(t.Module)
		if spec == "" {
			if strings.TrimSpace(t.Cmd) == "" {
				return false, nil
			}
			s, err := ParseGoInstallSpec(t.Cmd)
			if err != nil {
				return false, err
			}
			spec = s
		}
		res, err := InstallTool(InstallOptions{
			Spec:         spec,
			Path:         targetPath,
			Env:          env,
			Verbose:      verbose,
			ReleaseBuild: t.ReleaseBuild,
			DebugBuild:   t.DebugBuild,
			BinaryName:   t.BinaryName,
		})
		PrintInstallOutput(res.Output, err, verbose)
		if err != nil {
			return false, err
		}
		if res.InstallDir != "" {
			fmt.Printf("installed %s(go): %s -> %s\n", category, spec, filepath.Clean(res.InstallDir))
		}
		return true, nil

	case "clone", "git":
		if strings.TrimSpace(t.CloneURL) == "" {
			return false, fmt.Errorf("clone url empty")
		}
		res, err := InstallTool(InstallOptions{
			CloneURL:          t.CloneURL,
			BuildMethod:       t.Build,
			MakeTarget:        t.MakeTarget,
			WorkDir:           t.WorkDir,
			BinDirs:           t.BinDirs,
			Env:               env,
			GoreleaserConfig:  t.GoreleaserConfig,
			BinaryName:        t.BinaryName,
			RecurseSubmodules: t.RecurseSubmodules,
			ReleaseBuild:      t.ReleaseBuild,
			DebugBuild:        t.DebugBuild,
			Path:              targetPath,
			Verbose:           verbose,
		})
		PrintInstallOutput(res.Output, err, verbose)
		if err != nil {
			return false, err
		}
		if res.InstallDir != "" {
			fmt.Printf("installed %s(clone): %s -> %s\n", category, t.CloneURL, filepath.Clean(res.InstallDir))
		}
		return true, nil

	default:
		return false, fmt.Errorf("unsupported tool type: %s", t.Type)
	}
}

// InstallConfiguredToolsFromList installs a list of configs.Tool entries
func InstallConfiguredToolsFromList(list []configs.Tool, targetPath, category string, envFlags []string, verbose bool) (int, int) {
	total := 0
	failed := 0

	for _, t := range list {
		ttype := strings.ToLower(strings.TrimSpace(t.Type))
		envMerged := append([]string{}, envFlags...)
		if len(t.Env) > 0 {
			envMerged = append(envMerged, t.Env...)
		}

		switch ttype {
		case "", "go":
			// prioritize Module then parse Cmd
			spec := strings.TrimSpace(t.Module)
			if spec == "" {
				if strings.TrimSpace(t.Cmd) == "" {
					continue
				}
				s, err := ParseGoInstallSpec(t.Cmd)
				if err != nil {
					failed++
					continue
				}
				spec = s
			}
			res, err := InstallTool(InstallOptions{
				Spec:         spec,
				Path:         targetPath,
				Env:          envMerged,
				Verbose:      verbose,
				ReleaseBuild: t.ReleaseBuild,
				DebugBuild:   t.DebugBuild,
				BinaryName:   t.BinaryName,
			})
			PrintInstallOutput(res.Output, err, verbose)
			if err != nil {
				failed++
				continue
			}
			if res.InstallDir != "" {
				// best effort log via fmt
				fmt.Printf("installed %s(go): %s -> %s\n", category, spec, filepath.Clean(res.InstallDir))
			}
			total++

		case "clone", "git":
			if strings.TrimSpace(t.CloneURL) == "" {
				failed++
				continue
			}
			res, err := InstallTool(InstallOptions{
				CloneURL:          t.CloneURL,
				BuildMethod:       t.Build,
				MakeTarget:        t.MakeTarget,
				WorkDir:           t.WorkDir,
				BinDirs:           t.BinDirs,
				Env:               envMerged,
				GoreleaserConfig:  t.GoreleaserConfig,
				BinaryName:        t.BinaryName,
				RecurseSubmodules: t.RecurseSubmodules,
				ReleaseBuild:      t.ReleaseBuild,
				DebugBuild:        t.DebugBuild,
				Path:              targetPath,
				Verbose:           verbose,
			})
			PrintInstallOutput(res.Output, err, verbose)
			if err != nil {
				failed++
				continue
			}
			if res.InstallDir != "" {
				fmt.Printf("installed %s(clone): %s -> %s\n", category, t.CloneURL, filepath.Clean(res.InstallDir))
			}
			total++

		default:
			// skip unsupported type
		}
	}
	return total, failed
}

// ParseGoInstallSpec extracts spec from "go install ..." command string
func ParseGoInstallSpec(cmd string) (string, error) {
	fields := strings.Fields(cmd)
	if len(fields) < 3 || fields[0] != "go" || fields[1] != "install" {
		return "", fmt.Errorf("unsupported go tool cmd: %s", cmd)
	}
	for _, f := range fields[2:] {
		if strings.HasPrefix(f, "-") {
			continue
		}
		return f, nil
	}
	return "", fmt.Errorf("cannot find spec in: %s", cmd)
}

// PrintInstallOutput prints install output depending on verbose/err level
func PrintInstallOutput(output string, err error, verbose bool) {
	if strings.TrimSpace(output) == "" {
		return
	}
	for line := range strings.SplitSeq(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if err != nil {
			fmt.Println(line)
		} else if verbose {
			fmt.Println(line)
		} else {
			fmt.Println(line)
		}
	}
}

// PrintSingleTool prints a single tool info in requested format
func PrintSingleTool(bi *InstallToolsInfo, fmtFlag string, out io.Writer) error {
	switch strings.ToLower(fmtFlag) {
	case "json":
		if err := style.PrintJSON(out, bi); err != nil {
			b, _ := json.MarshalIndent(bi, "", "  ")
			fmt.Fprintln(out, string(b))
		}
		return nil
	case "yaml":
		if err := style.PrintYAML(out, bi); err != nil {
			b, _ := json.MarshalIndent(bi, "", "  ")
			fmt.Fprintln(out, string(b))
		}
		return nil
	case "table":
		kv := func(k, v string) []string { return []string{k, v} }
		rows := make([][]string, 0, 16)
		add := func(k, v string) {
			if strings.TrimSpace(v) == "" {
				return
			}
			rows = append(rows, kv(k, v))
		}
		add("Name", bi.Name)
		add("URL", bi.URL)
		add("CloneURL", bi.CloneURL)
		add("Build", bi.Build)
		add("MakeTarget", bi.MakeTarget)
		add("WorkDir", bi.WorkDir)
		if len(bi.BinDirs) > 0 {
			add("BinDirs", strings.Join(bi.BinDirs, ", "))
		}
		if len(bi.Env) > 0 {
			add("Env", strings.Join(bi.Env, ", "))
		}
		add("GoreleaserConfig", bi.GoreleaserConfig)
		add("BinaryName", bi.BinaryName)
		if bi.InstallType != nil {
			add("InstallType.Name", bi.InstallType.Name)
			add("InstallType.OS", bi.InstallType.OS)
			add("InstallType.Arch", bi.InstallType.Arch)
		}
		if len(rows) == 0 {
			rows = append(rows, kv("Name", ""))
		}
		if err := style.PrintTable(out, []string{"Field", "Value"}, rows, 0); err != nil {
			for _, r := range rows {
				fmt.Fprintf(out, "%s: %s\n", r[0], r[1])
			}
		}
		return nil
	default:
		fmt.Fprintf(out, "unsupported format: %s\n", fmtFlag)
		return nil
	}
}

// mustUserHome 返回用户 home 目录
func mustUserHome() string {
	h, _ := os.UserHomeDir()
	return h
}
