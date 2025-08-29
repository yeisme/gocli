package tools

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
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

	totalDeps, failedDeps := InstallConfiguredToolsFromList(cfg.Tools.Deps, depsPath, "dep", envFlags, verbose)
	totalGlobal, failedGlobal := InstallConfiguredToolsFromList(cfg.Tools.Global, globalPath, "global", envFlags, verbose)

	total := totalDeps + totalGlobal
	failed := failedDeps + failedGlobal

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
	total, failed := InstallConfiguredToolsFromList(cfg.Tools.Global, targetPath, "global", envFlags, verbose)
	if total == 0 && failed == 0 {
		return nil
	}
	if failed > 0 {
		return fmt.Errorf("%d tool(s) failed", failed)
	}
	return nil
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
	for _, line := range strings.Split(output, "\n") {
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
func PrintSingleTool(bi *InstallToolsInfo, fmtFlag string, out io.Writer) {
	switch strings.ToLower(fmtFlag) {
	case "json":
		if err := style.PrintJSON(out, bi); err != nil {
			b, _ := json.MarshalIndent(bi, "", "  ")
			fmt.Fprintln(out, string(b))
		}
	case "yaml":
		if err := style.PrintYAML(out, bi); err != nil {
			b, _ := json.MarshalIndent(bi, "", "  ")
			fmt.Fprintln(out, string(b))
		}
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
	default:
		fmt.Fprintf(out, "unsupported format: %s\n", fmtFlag)
	}
}

// mustUserHome 返回用户 home 目录
func mustUserHome() string {
	h, _ := os.UserHomeDir()
	return h
}
