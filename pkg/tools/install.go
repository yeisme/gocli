package tools

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
	"github.com/yeisme/gocli/pkg/configs"
)

// goInstallWithEnv 支持传入额外环境变量（如 GOBIN）

// InstallOptions 用于统一描述工具安装的参数
type InstallOptions struct {
	// go install 专用：模块/本地路径规范，如 github.com/owner/repo/cmd/foo@vX 或 ./path
	Spec string

	// clone+make 专用：仓库 URL，可带 #ref
	CloneURL   string
	MakeTarget string

	// 构建方式（在 clone 模式下有效）："make"(默认)、"goreleaser"、后续可扩展
	BuildMethod string
	// 传递给构建器的附加参数，例如 goreleaser 的 flags 或 make 的额外参数
	BuildArgs []string
	// 在仓库中的子目录进行构建（为空则使用仓库根目录）
	WorkDir string
	// goreleaser 可选配置文件路径（相对于仓库根或 WorkDir）
	GoreleaserConfig string

	// 指定安装目录，等价于设置 GOBIN；若为空，沿用默认规则
	Path string

	// 额外构建/运行环境变量，如 CGO_ENABLED=1
	Env []string

	// 当使用 clone+make 时，从这些目录收集可执行产物并复制到安装目录（可多次传入，或使用平台路径分隔符分隔）
	BinDirs []string

	// 是否 verbose
	Verbose bool
	// 预设构建模式
	ReleaseBuild bool
	DebugBuild   bool

	// BinaryName 指定生成的二进制文件名（可选）
	BinaryName string
	// Clone: 是否递归克隆子模块
	RecurseSubmodules bool

	// Force: 强制模型，如果目标目录已存在则覆盖，否则就复用
	Force bool
}

// InstallResult 统一返回值
type InstallResult struct {
	// 原始命令输出（可能为多行）
	Output string
	// 明确的安装目录（当设置 Path 或 go install 传入 GOBIN 时）
	InstallDir string
	// 根据环境与 go env 推断的安装目录（即便 InstallDir 为空也会提供）
	ProbableInstallDir string
	// 执行模式：go_install 或 clone_make
	Mode string
}

// InstallTool 统一入口：根据是否传入 CloneURL 决定使用 go install 或 clone+make
func InstallTool(opts InstallOptions) (InstallResult, error) {
	res := InstallResult{}
	verbose := opts.Verbose

	// 预处理 env 与 Path -> GOBIN
	env := append([]string{}, opts.Env...)
	finalDir := ""
	if opts.Path != "" {
		p := expandPath(opts.Path)
		if err := os.MkdirAll(p, 0o755); err != nil {
			return res, fmt.Errorf("create install dir %s failed: %w", p, err)
		}
		abs, _ := filepath.Abs(p)
		finalDir = abs
		env = append(env, fmt.Sprintf("GOBIN=%s", abs))
	}

	if opts.CloneURL != "" {
		// 将 Release/Debug 预设尽力通过 GOFLAGS 传递给 make/go build/goreleaser
		var goflags []string
		if opts.ReleaseBuild {
			goflags = append(goflags, "-trimpath", "-ldflags=-s -w")
		}
		if opts.DebugBuild {
			// 注意：GOFLAGS 无法很好地表达带空格的值，这里为最佳努力；部分构建脚本可能不生效
			goflags = append(goflags, "-gcflags=all=-N -l")
		}
		if len(goflags) > 0 {
			env = append(env, "GOFLAGS="+strings.Join(goflags, " "))
		}

		// 由构建器自身提供默认 BinDirs；若用户显式传入则优先生效
		binDirs := append([]string{}, opts.BinDirs...)
		method := strings.ToLower(strings.TrimSpace(opts.BuildMethod))

		out, err := CloneAndBuildInstall(CloneBuildOptions{
			CloneURL:          opts.CloneURL,
			InstallDir:        opts.Path,
			Env:               env,
			Verbose:           verbose,
			RecurseSubmodules: opts.RecurseSubmodules,
			BuildMethod:       method,
			MakeTarget:        opts.MakeTarget,
			BuildArgs:         opts.BuildArgs,
			WorkDir:           opts.WorkDir,
			GoreleaserConfig:  opts.GoreleaserConfig,
			BinDirs:           binDirs,
			BinaryName:        opts.BinaryName,
			Force:             opts.Force,
		})
		res.Output = out
		res.Mode = "clone_build"
		res.InstallDir = finalDir
		// 推断 probable 安装目录
		if finalDir != "" {
			res.ProbableInstallDir = finalDir
		} else {
			p := expandPath(viper.GetString("tools.path"))
			if p == "" {
				if home, e := os.UserHomeDir(); e == nil {
					p = filepath.Join(home, ".gocli", "tools")
				}
			}
			if p != "" {
				if abs, _ := filepath.Abs(p); abs != "" {
					res.ProbableInstallDir = abs
				} else {
					res.ProbableInstallDir = p
				}
			}
		}
		return res, err
	}

	// go install 路径
	var buildArgs []string
	if opts.ReleaseBuild {
		buildArgs = append(buildArgs, "-trimpath", "-ldflags=-s -w")
	}
	if opts.DebugBuild {
		buildArgs = append(buildArgs, "-gcflags=all=-N -l")
	}
	// 在 go install 前快照目标安装目录（若可确定），以便安装后重命名
	var preSnap map[string]time.Time
	var targetDir string
	if finalDir != "" {
		targetDir = finalDir
	} else {
		// 尝试从 go env 推断 GOBIN（为空则回退 GOPATH/bin）
		targetDir = DetermineGoBinDir()
	}
	if targetDir != "" && opts.BinaryName != "" {
		preSnap = SnapshotExecutables(targetDir)
	}

	out, dir, err := InstallGoTool(opts.Spec, opts.Path, env, verbose, buildArgs)
	res.Output = out
	res.Mode = "go_install"
	res.InstallDir = dir
	if dir != "" {
		res.ProbableInstallDir = dir
	}
	// 安装成功并要求自定义二进制名时，尝试重命名
	if err == nil && opts.BinaryName != "" {
		// 优先使用明确安装目录
		renameDir := dir
		if renameDir == "" {
			renameDir = targetDir
		}
		if renameDir != "" {
			if e := RenameInstalledBinary(renameDir, preSnap, opts.BinaryName, verbose); e != nil {
				// 不阻断主流程，仅附加到输出中
				res.Output += "\n[rename] warning: " + e.Error()
			}
		}
	}
	return res, err
}

// InstallCommandOptions 定义了install命令的选项和上下文
type InstallCommandOptions struct {
	// 命令行参数
	Args []string
	// 嵌入 InstallOptions 以复用安装相关字段
	InstallOptions

	// 额外命令行上下文字段
	// Global indicates --global was set on the install command
	Global         bool
	Quiet          bool
	GoCLIToolsPath string
	ToolsConfigDir []string

	// Yes: 非交互模式直接执行
	Yes bool
	// Input: 交互输入源（默认 os.Stdin）
	Input io.Reader
}

// ExecuteInstallCommand 执行install命令的封装函数
func ExecuteInstallCommand(opts InstallCommandOptions, outputWriter io.Writer) error {
	if err := validateInstallCmdOptions(opts); err != nil {
		return err
	}
	if isBatchInstallCase(opts) {
		return executeBatchInstall(opts)
	}

	pathFlag, msg, err := resolveInstallPath(opts)
	if err != nil {
		return err
	}
	if msg != "" {
		fmt.Fprintln(outputWriter, msg)
	}

	cloneURL, makeTarget, envFlags, binDirs, releaseBuild, debugBuild, v := prepareInstallVariables(opts)
	spec := firstArg(opts.Args)
	spec, cloneURL, makeTarget, binDirs, envFlags, addBuildMethod, workDir, goreleaserConfig, binaryName := mapBuiltinToolIfNeeded(spec, cloneURL, makeTarget, binDirs, envFlags, opts.ToolsConfigDir, v, outputWriter)
	if err = maybeSuggestUnknownShortName(spec, opts, outputWriter); err != nil {
		return err
	}
	if err = checkMutualExclusion(cloneURL, spec); err != nil {
		return err
	}
	installOpts := buildInstallOptions(spec, cloneURL, makeTarget, pathFlag, envFlags, binDirs,
		v, releaseBuild, debugBuild, binaryName, addBuildMethod, opts.BuildArgs, workDir, goreleaserConfig, opts)
	if err = validateFinalInstallOptions(installOpts); err != nil {
		return err
	}
	if !opts.Yes {
		proceed, confirmErr := confirmInstall(installOpts, opts, outputWriter)
		if confirmErr != nil {
			return confirmErr
		}
		if !proceed {
			fmt.Fprintln(outputWriter, "aborted.")
			return nil
		}
	}
	res, err := InstallTool(installOpts)
	printInstallResult(res, err, outputWriter)
	return err
}

// isBatchInstallCase returns true if we should perform batch installation logic
func isBatchInstallCase(opts InstallCommandOptions) bool {
	return opts.CloneURL == "" && len(opts.Args) == 0
}

// executeBatchInstall performs batch installation of configured tools
func executeBatchInstall(opts InstallCommandOptions) error {
	for _, p := range opts.ToolsConfigDir {
		_ = LoadUserTools(p)
	}
	cfg := configs.GetConfig()
	if opts.Global {
		return BatchInstallConfiguredGlobalTools(cfg, opts.Env, opts.Verbose)
	}
	return BatchInstallConfiguredTools(cfg, opts.Env, opts.Verbose)
}

// prepareInstallVariables extracts frequently used mutable copies
func prepareInstallVariables(opts InstallCommandOptions) (cloneURL, makeTarget string, envFlags, binDirs []string, releaseBuild, debugBuild, verbose bool) {
	cloneURL = opts.CloneURL
	makeTarget = opts.MakeTarget
	envFlags = append([]string{}, opts.Env...)
	binDirs = append([]string{}, opts.BinDirs...)
	releaseBuild = opts.ReleaseBuild
	debugBuild = opts.DebugBuild
	verbose = opts.Verbose
	return
}

// firstArg returns first argument or empty string
func firstArg(args []string) string {
	if len(args) > 0 {
		return args[0]
	}
	return ""
}

// maybeSuggestUnknownShortName prints fuzzy suggestions if short name not found; returns error when suggestions shown
func maybeSuggestUnknownShortName(spec string, opts InstallCommandOptions, out io.Writer) error {
	if spec == "" || strings.ContainsAny(spec, "/\\") {
		return nil
	}
	for _, p := range opts.ToolsConfigDir {
		_ = LoadUserTools(p)
	}
	if SearchTools(spec, opts.ToolsConfigDir) != nil {
		return nil
	}
	matches := FindToolsFuzzy(spec, opts.ToolsConfigDir)
	if len(matches) == 0 {
		return nil
	}
	fmt.Fprintf(out, "Tool '%s' not found. Did you mean one of these?\n", spec)
	for _, m := range matches {
		fmt.Fprintf(out, "  - %s", m.Name)
		if strings.TrimSpace(m.URL) != "" {
			fmt.Fprintf(out, " (url=%s)", m.URL)
		}
		if strings.TrimSpace(m.CloneURL) != "" {
			fmt.Fprintf(out, " (clone=%s)", m.CloneURL)
		}
		fmt.Fprintln(out)
	}
	return fmt.Errorf("unknown tool: %s", spec)
}

// checkMutualExclusion validates cloneURL and spec are not both set
func checkMutualExclusion(cloneURL, spec string) error {
	if cloneURL != "" && spec != "" {
		return fmt.Errorf("please specify either a module/local path or --clone, not both")
	}
	return nil
}

// validateFinalInstallOptions ensures we have something to install
func validateFinalInstallOptions(opts InstallOptions) error {
	if opts.CloneURL == "" && opts.Spec == "" {
		return fmt.Errorf("missing tool spec provide a module path or use --clone")
	}
	return nil
}

// confirmInstall prints plan and asks for confirmation; returns proceed
func confirmInstall(installOpts InstallOptions, opts InstallCommandOptions, outputWriter io.Writer) (bool, error) {
	reader := bufio.NewReader(opts.Input)
	if reader == nil {
		reader = bufio.NewReader(os.Stdin)
	}
	fmt.Fprintln(outputWriter, "Planned installation:")
	if installOpts.CloneURL != "" {
		fmt.Fprintf(outputWriter, "  Mode      : clone_build (%s)\n", firstNonEmpty(installOpts.BuildMethod, "make"))
		fmt.Fprintf(outputWriter, "  CloneURL  : %s\n", installOpts.CloneURL)
		if installOpts.MakeTarget != "" {
			fmt.Fprintf(outputWriter, "  MakeTarget: %s\n", installOpts.MakeTarget)
		}
	} else {
		fmt.Fprintln(outputWriter, "  Mode      : go_install")
		fmt.Fprintf(outputWriter, "  Spec      : %s\n", installOpts.Spec)
	}
	if installOpts.Path != "" {
		fmt.Fprintf(outputWriter, "  InstallDir: %s\n", filepath.Clean(installOpts.Path))
	}
	if installOpts.BinaryName != "" {
		fmt.Fprintf(outputWriter, "  BinaryName: %s\n", installOpts.BinaryName)
	}
	if len(installOpts.Env) > 0 {
		fmt.Fprintf(outputWriter, "  Env       : %s\n", strings.Join(installOpts.Env, ", "))
	}
	if len(installOpts.BinDirs) > 0 {
		fmt.Fprintf(outputWriter, "  BinDirs   : %s\n", strings.Join(installOpts.BinDirs, ", "))
	}
	if installOpts.ReleaseBuild {
		fmt.Fprintln(outputWriter, "  Flags     : release-build")
	}
	if installOpts.DebugBuild {
		fmt.Fprintln(outputWriter, "  Flags     : debug-build")
	}
	fmt.Fprint(outputWriter, "Proceed? [y/N]: ")
	ans, _ := reader.ReadString('\n')
	ans = strings.TrimSpace(strings.ToLower(ans))
	return ans == "y" || ans == "yes", nil
}

// validateInstallCmdOptions checks mutual exclusions and batch-case handling
func validateInstallCmdOptions(opts InstallCommandOptions) error {
	if opts.ReleaseBuild && opts.DebugBuild {
		return fmt.Errorf("--release-build and --debug-build cannot be used together")
	}
	if opts.CloneURL == "" && len(opts.Args) == 0 {
		return fmt.Errorf("batch install not implemented in pkg/tools yet")
	}
	return nil
}

// resolveInstallPath returns effective pathFlag and optional message to print
func resolveInstallPath(opts InstallCommandOptions) (string, string, error) {
	pathFlag := opts.Path
	if pathFlag == "" {
		if opts.Global {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", "", fmt.Errorf("failed to get user home: %w", err)
			}
			pathFlag = filepath.Join(home, ".gocli", "tools")
			return pathFlag, fmt.Sprintf("--global selected: default install path -> %s", filepath.Clean(pathFlag)), nil
		}
		pathFlag = opts.GoCLIToolsPath
	}
	return pathFlag, "", nil
}

// mapBuiltinToolIfNeeded maps short builtin names to spec/clone and augments flags
func mapBuiltinToolIfNeeded(spec, cloneURL, makeTarget string, binDirs, envFlags []string, toolsConfigDir []string, v bool, outputWriter io.Writer) (string, string, string, []string, []string, string, string, string, string) {
	var buildMethod, workDir, goreleaserConfig, binaryName string
	if spec == "" {
		return spec, cloneURL, makeTarget, binDirs, envFlags, buildMethod, workDir, goreleaserConfig, binaryName
	}
	if strings.Contains(spec, "/") || strings.Contains(spec, "\\") {
		return spec, cloneURL, makeTarget, binDirs, envFlags, buildMethod, workDir, goreleaserConfig, binaryName
	}

	// load user tools
	for _, p := range toolsConfigDir {
		_ = LoadUserTools(p)
	}

	orig := spec
	name := orig
	ver := ""
	if i := strings.Index(orig, "@"); i > 0 {
		name = orig[:i]
		ver = orig[i:]
	}

	bi := SearchTools(name, toolsConfigDir)
	if bi == nil {
		for _, t := range BuiltinTools {
			if t.Name == name {
				tt := t
				bi = &tt
				break
			}
		}
	}

	if bi == nil {
		return spec, cloneURL, makeTarget, binDirs, envFlags, buildMethod, workDir, goreleaserConfig, binaryName
	}

	if strings.TrimSpace(bi.CloneURL) != "" {
		clone := bi.CloneURL
		if ver != "" {
			ref := ver[1:]
			if p := strings.Index(clone, "#"); p >= 0 {
				clone = clone[:p]
			}
			clone = clone + "#" + ref
		}
		cloneURL = clone
		if bi.Build != "" {
			buildMethod = bi.Build
		}
		if bi.MakeTarget != "" {
			makeTarget = bi.MakeTarget
		}
		if bi.WorkDir != "" {
			workDir = bi.WorkDir
		}
		if len(bi.BinDirs) > 0 {
			binDirs = append(binDirs, bi.BinDirs...)
		}
		if len(bi.Env) > 0 {
			envFlags = append(envFlags, bi.Env...)
		}
		if bi.GoreleaserConfig != "" {
			goreleaserConfig = bi.GoreleaserConfig
		}
		if bi.BinaryName != "" {
			binaryName = bi.BinaryName
		}
		spec = ""
		if v {
			fmt.Fprintf(outputWriter, "mapped builtin tool %s -> clone %s (build=%s)\n", orig, cloneURL, buildMethod)
		}
		return spec, cloneURL, makeTarget, binDirs, envFlags, buildMethod, workDir, goreleaserConfig, binaryName
	}

	// go install 模式
	mapped := bi.URL + ver
	spec = mapped
	if bi.BinaryName != "" {
		binaryName = bi.BinaryName
	}
	if v {
		fmt.Fprintf(outputWriter, "mapped builtin tool %s -> %s\n", orig, spec)
	}
	return spec, cloneURL, makeTarget, binDirs, envFlags, buildMethod, workDir, goreleaserConfig, binaryName
}

// buildInstallOptions builds InstallOptions from resolved inputs
func buildInstallOptions(spec, cloneURL, makeTarget, pathFlag string, envFlags, binDirs []string, verbose bool, releaseBuild, debugBuild bool, binaryName, buildMethod string, buildArgs []string, workDir, goreleaserConfig string, opts InstallCommandOptions) InstallOptions {
	return InstallOptions{
		Spec:              spec,
		CloneURL:          cloneURL,
		MakeTarget:        makeTarget,
		Path:              pathFlag,
		Env:               envFlags,
		BinDirs:           binDirs,
		Verbose:           verbose,
		ReleaseBuild:      releaseBuild,
		DebugBuild:        debugBuild,
		BinaryName:        binaryName,
		BuildMethod:       buildMethod,
		BuildArgs:         buildArgs,
		WorkDir:           workDir,
		GoreleaserConfig:  goreleaserConfig,
		RecurseSubmodules: opts.RecurseSubmodules,
		Force:             opts.Force,
	}
}

// printInstallResult prints the install output and locations
func printInstallResult(res InstallResult, err error, out io.Writer) {
	if strings.TrimSpace(res.Output) != "" {
		fmt.Fprint(out, res.Output)
	}
	if err != nil {
		return
	}
	if res.InstallDir != "" {
		fmt.Fprintf(out, "installed to: %s\n", filepath.Clean(res.InstallDir))
	}
	if res.ProbableInstallDir != "" && res.InstallDir == "" {
		fmt.Fprintf(out, "probable install dir: %s\n", filepath.Clean(res.ProbableInstallDir))
	}
}

// firstNonEmpty returns first non-empty string else fallback
func firstNonEmpty(s ...string) string {
	for _, v := range s {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
