package configs

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"

	"github.com/spf13/viper"
	"github.com/yeisme/gocli/pkg/tools"
)

var (
	goEnvCache    map[string]string
	loadGoEnvOnce sync.Once
)

// loadGoEnv loads environment variables from `go env` and caches them.
// It runs only once.
func loadGoEnv() {
	loadGoEnvOnce.Do(func() {
		goEnvCache = make(map[string]string)
		// The most reliable source is the `go env` command itself.
		output, err := tools.NewExecutor("go", "env").Output()
		if err != nil {
			// Fallback to reading default go.env file if `go env` fails
			goRoot := os.Getenv("GOROOT")
			if goRoot != "" {
				goEnvFile := filepath.Join(goRoot, "go.env")
				file, err := os.Open(goEnvFile)
				if err == nil {
					defer func() {
						if err := file.Close(); err != nil {
							fmt.Fprintf(os.Stderr, "error closing go.env file: %v\n", err)
						}
					}()
					scanner := bufio.NewScanner(file)
					for scanner.Scan() {
						line := strings.TrimSpace(scanner.Text())
						if line == "" || strings.HasPrefix(line, "#") {
							continue
						}
						parts := strings.SplitN(line, "=", 2)
						if len(parts) == 2 {
							goEnvCache[parts[0]] = parts[1]
						}
					}
				}
			}
			return // Exit if we can't get env from `go env` or file
		}

		scanner := bufio.NewScanner(strings.NewReader(output))
		for scanner.Scan() {
			line := scanner.Text()
			// On Windows, the output might be `set GOROOT=C:\Go`
			if runtime.GOOS == "windows" && strings.HasPrefix(line, "set ") {
				line = strings.TrimPrefix(line, "set ")
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := parts[0]
				// Values can be enclosed in quotes
				value := strings.Trim(parts[1], `"`)
				goEnvCache[key] = value
			}
		}
	})
}

// EnvConfig 环境变量配置
type EnvConfig struct {
	// Go 核心环境变量
	GoRoot     string `mapstructure:"GOROOT" jsonschema:"title=GOROOT,description=Go installation root directory,nullable"`                       // Go 安装路径
	GoPath     string `mapstructure:"GOPATH" jsonschema:"title=GOPATH,description=Primary workspace path (may contain multiple paths),nullable"`  // Go 工作空间路径
	GoMod      string `mapstructure:"GOMOD" jsonschema:"title=GOMOD,description=Path to current go.mod file,nullable"`                            // Go modules 目录
	GoModCache string `mapstructure:"GOMODCACHE" jsonschema:"title=GOMODCACHE,description=Module download/cache directory,nullable"`              // Go modules 缓存目录
	GoSumDB    string `mapstructure:"GOSUMDB" jsonschema:"title=GOSUMDB,description=Checksum database URL(s),nullable"`                           // Go checksum 数据库
	GoProxy    string `mapstructure:"GOPROXY" jsonschema:"title=GOPROXY,description=Module proxy list,nullable"`                                  // Go 模块代理
	GoPrivate  string `mapstructure:"GOPRIVATE" jsonschema:"title=GOPRIVATE,description=Patterns for private modules (comma separated),nullable"` // 私有模块路径模式
	GoNoProxy  string `mapstructure:"GONOPROXY" jsonschema:"title=GONOPROXY,description=Patterns to bypass proxy,nullable"`                       // 不使用代理的模块路径模式
	GoNoSumDB  string `mapstructure:"GONOSUMDB" jsonschema:"title=GONOSUMDB,description=Patterns to bypass checksum database,nullable"`           // 不使用 checksum 数据库的模块路径模式

	// 构建相关环境变量
	GoOS     string `mapstructure:"GOOS" jsonschema:"title=GOOS,description=Target operating system,nullable"`                                    // 目标操作系统
	GoArch   string `mapstructure:"GOARCH" jsonschema:"title=GOARCH,description=Target architecture,nullable"`                                    // 目标架构
	Go386    string `mapstructure:"GO386" jsonschema:"title=GO386,description=386 architecture features (e.g. sse2),nullable"`                    // 386 架构设置
	GoAMD64  string `mapstructure:"GOAMD64" jsonschema:"title=GOAMD64,description=AMD64 microarchitecture level (v1-v4),nullable"`                // AMD64 架构设置
	GoARM    string `mapstructure:"GOARM" jsonschema:"title=GOARM,description=ARM architecture version (5,6,7),nullable"`                         // ARM 架构设置
	GoARM64  string `mapstructure:"GOARM64" jsonschema:"title=GOARM64,description=ARM64 architecture tuning flags,nullable"`                      // ARM64 架构设置
	GoMIPS   string `mapstructure:"GOMIPS" jsonschema:"title=GOMIPS,description=MIPS architecture settings (hardfloat|softfloat),nullable"`       // MIPS 架构设置
	GoMIPS64 string `mapstructure:"GOMIPS64" jsonschema:"title=GOMIPS64,description=MIPS64 architecture settings (hardfloat|softfloat),nullable"` // MIPS64 架构设置
	GoPPC64  string `mapstructure:"GOPPC64" jsonschema:"title=GOPPC64,description=PPC64 architecture level (power8 etc),nullable"`                // PPC64 架构设置
	GoWASM   string `mapstructure:"GOWASM" jsonschema:"title=GOWASM,description=WebAssembly feature flags,nullable"`                              // WebAssembly 设置

	// 编译器相关环境变量
	GoGCFlags  string `mapstructure:"GOGCFLAGS" jsonschema:"title=GOGCFLAGS,description=Extra gc compiler flags,nullable"`                            // Go 编译器标志
	GoAsmFlags string `mapstructure:"GOASMFLAGS" jsonschema:"title=GOASMFLAGS,description=Extra assembler flags,nullable"`                            // Go 汇编器标志
	GoLDFlags  string `mapstructure:"GOLDFLAGS" jsonschema:"title=GOLDFLAGS,description=Extra linker flags,nullable"`                                 // Go 链接器标志
	GoFlags    string `mapstructure:"GOFLAGS" jsonschema:"title=GOFLAGS,description=Default go command flags,nullable"`                               // Go 命令标志
	GoInsecure string `mapstructure:"GOINSECURE" jsonschema:"title=GOINSECURE,description=Allow insecure (non-HTTPS) module paths patterns,nullable"` // 允许不安全的 scheme

	// CGO 相关环境变量
	CGOEnabled  string `mapstructure:"CGO_ENABLED" jsonschema:"title=CGO_ENABLED,description=Enable CGO (0 or 1),nullable"`
	CGOCFlags   string `mapstructure:"CGO_CFLAGS" jsonschema:"title=CGO_CFLAGS,description=CGO C compiler flags,nullable"`
	CGOCPPFlags string `mapstructure:"CGO_CPPFLAGS" jsonschema:"title=CGO_CPPFLAGS,description=CGO C preprocessor flags,nullable"`
	CGOLDFlags  string `mapstructure:"CGO_LDFLAGS" jsonschema:"title=CGO_LDFLAGS,description=CGO linker flags,nullable"`
	CGOCXXFlags string `mapstructure:"CGO_CXXFLAGS" jsonschema:"title=CGO_CXXFLAGS,description=CGO C++ compiler flags,nullable"`

	// 调试和性能相关环境变量
	GoTrace        string `mapstructure:"GOTRACE" jsonschema:"title=GOTRACE,description=Execution trace output path,nullable"`                // 跟踪执行
	GoDebug        string `mapstructure:"GODEBUG" jsonschema:"title=GODEBUG,description=Runtime debug settings,nullable"`                     // 调试设置
	GoMemProfile   string `mapstructure:"GOMEMPROFILE" jsonschema:"title=GOMEMPROFILE,description=Memory profile output path,nullable"`       // 内存分析文件
	GoCPUProfile   string `mapstructure:"GOCPUPROFILE" jsonschema:"title=GOCPUPROFILE,description=CPU profile output path,nullable"`          // CPU 分析文件
	GoBlockProfile string `mapstructure:"GOBLOCKPROFILE" jsonschema:"title=GOBLOCKPROFILE,description=Blocking profile output path,nullable"` // 阻塞分析文件
	GoMutexProfile string `mapstructure:"GOMUTEXPROFILE" jsonschema:"title=GOMUTEXPROFILE,description=Mutex profile output path,nullable"`    // 互斥锁分析文件

	// 工具链相关环境变量
	GoToolchain string `mapstructure:"GOTOOLCHAIN" jsonschema:"title=GOTOOLCHAIN,description=Go toolchain selection (auto|go1.x|path),nullable"` // Go 工具链版本
	GoToolDir   string `mapstructure:"GOTOOLDIR" jsonschema:"title=GOTOOLDIR,description=Go toolchain binaries directory,nullable"`              // Go 工具目录
	GoCache     string `mapstructure:"GOCACHE" jsonschema:"title=GOCACHE,description=Build cache directory,nullable"`                            // 构建缓存目录
	GoTmpDir    string `mapstructure:"GOTMPDIR" jsonschema:"title=GOTMPDIR,description=Temporary directory for go commands,nullable"`            // 临时目录
	GoWork      string `mapstructure:"GOWORK" jsonschema:"title=GOWORK,description=Go workspace file mode (auto|off|path),nullable"`             // Go 工作空间文件
	GoWorkSum   string `mapstructure:"GOWORKSUM" jsonschema:"title=GOWORKSUM,description=Workspace checksum file path,nullable"`                 // Go 工作空间校验和文件

	// 实验性功能环境变量
	GoExperiment string `mapstructure:"GOEXPERIMENT" jsonschema:"title=GOEXPERIMENT,description=Comma separated experimental feature flags,nullable"` // 实验性功能
	// 常用的 GOEXPERIMENT 选项：
	// - "rangefunc": 启用 range-over-func 特性
	// - "arenas": 启用 arenas 内存管理实验
	// - "cgocheck2": 启用更严格的 cgo 指针检查
	// - "fieldtrack": 启用字段跟踪功能
	// - "preemptibleloops": 启用可抢占循环
	// - "staticlockranking": 启用静态锁排序检查
	// 可以组合使用，用逗号分隔，如: "rangefunc,arenas"

	// 其他环境变量可以通过 map 存储
	Custom map[string]string `mapstructure:",remain" jsonschema:"title=Custom,description=Additional custom environment variables (free-form key/values)"`
}

// setEnvConfigDefaults 设置环境变量配置的默认值
func setEnvConfigDefaults() {
	// Go 核心环境变量默认值 - 优先通过 go env 获取
	viper.SetDefault("env.GOROOT", getGoEnvOrDefault("GOROOT", ""))
	viper.SetDefault("env.GOPATH", getGoEnvOrDefault("GOPATH", ""))

	// 模块相关环境变量默认值
	viper.SetDefault("env.GOPROXY", getGoEnvOrDefault("GOPROXY", "https://proxy.golang.org,direct"))
	viper.SetDefault("env.GOSUMDB", getGoEnvOrDefault("GOSUMDB", "sum.golang.org"))
	viper.SetDefault("env.GOPRIVATE", getGoEnvOrDefault("GOPRIVATE", ""))
	viper.SetDefault("env.GONOPROXY", getGoEnvOrDefault("GONOPROXY", ""))
	viper.SetDefault("env.GONOSUMDB", getGoEnvOrDefault("GONOSUMDB", ""))
	viper.SetDefault("env.GOMOD", getGoEnvOrDefault("GOMOD", ""))
	// GOMODCACHE: 优先取 `go env GOMODCACHE`，否则回退到 GOPATH/pkg/mod
	viper.SetDefault(
		"env.GOMODCACHE",
		func() string {
			if v := getGoEnvOrDefault("GOMODCACHE", ""); v != "" {
				return v
			}
			gp := getGoEnvOrDefault("GOPATH", "")
			if gp == "" {
				return ""
			}
			return filepath.Join(gp, "pkg", "mod")
		}(),
	)

	// 构建相关环境变量默认值 - 优先通过 go env 获取
	viper.SetDefault("env.GOOS", getGoEnvOrDefault("GOOS", runtime.GOOS))
	viper.SetDefault("env.GOARCH", getGoEnvOrDefault("GOARCH", runtime.GOARCH))
	viper.SetDefault("env.GO386", getGoEnvOrDefault("GO386", "sse2"))
	viper.SetDefault("env.GOAMD64", getGoEnvOrDefault("GOAMD64", "v1"))
	viper.SetDefault("env.GOARM", getGoEnvOrDefault("GOARM", "6"))
	viper.SetDefault("env.GOMIPS", getGoEnvOrDefault("GOMIPS", "hardfloat"))
	viper.SetDefault("env.GOMIPS64", getGoEnvOrDefault("GOMIPS64", "hardfloat"))
	viper.SetDefault("env.GOPPC64", getGoEnvOrDefault("GOPPC64", "power8"))

	// 编译器相关环境变量默认值
	viper.SetDefault("env.GOFLAGS", getGoEnvOrDefault("GOFLAGS", ""))
	viper.SetDefault("env.GOGCFLAGS", getGoEnvOrDefault("GOGCFLAGS", ""))
	viper.SetDefault("env.GOASMFLAGS", getGoEnvOrDefault("GOASMFLAGS", ""))
	viper.SetDefault("env.GOLDFLAGS", getGoEnvOrDefault("GOLDFLAGS", ""))
	viper.SetDefault("env.GOINSECURE", getGoEnvOrDefault("GOINSECURE", ""))

	// CGO 相关环境变量默认值
	viper.SetDefault("env.CGO_ENABLED", getGoEnvOrDefault("CGO_ENABLED", "1"))
	viper.SetDefault("env.CGO_CFLAGS", getGoEnvOrDefault("CGO_CFLAGS", "-g -O2"))
	viper.SetDefault("env.CGO_CPPFLAGS", getGoEnvOrDefault("CGO_CPPFLAGS", "-g -O2"))
	viper.SetDefault("env.CGO_LDFLAGS", getGoEnvOrDefault("CGO_LDFLAGS", "-g -O2"))
	viper.SetDefault("env.CGO_CXXFLAGS", getGoEnvOrDefault("CGO_CXXFLAGS", "-g -O2"))

	// 调试和性能相关环境变量默认值
	viper.SetDefault("env.GODEBUG", getGoEnvOrDefault("GODEBUG", ""))
	viper.SetDefault("env.GOTRACE", getGoEnvOrDefault("GOTRACE", ""))

	// 工具链相关环境变量默认值
	viper.SetDefault("env.GOTOOLCHAIN", getGoEnvOrDefault("GOTOOLCHAIN", "auto"))
	viper.SetDefault("env.GOTOOLDIR", getGoEnvOrDefault("GOTOOLDIR", ""))
	viper.SetDefault("env.GOCACHE", getGoEnvOrDefault("GOCACHE", ""))
	viper.SetDefault("env.GOTMPDIR", getGoEnvOrDefault("GOTMPDIR", ""))
	viper.SetDefault("env.GOWORK", getGoEnvOrDefault("GOWORK", "auto"))
	viper.SetDefault("env.GOWORKSUM", getGoEnvOrDefault("GOWORKSUM", ""))

	// 实验性功能环境变量默认值
	viper.SetDefault("env.GOEXPERIMENT", getGoEnvOrDefault("GOEXPERIMENT", ""))
}

// ApplyEnvVars 应用环境变量到当前进程
func (e *EnvConfig) ApplyEnvVars() {
	// 使用反射获取结构体字段
	v := reflect.ValueOf(*e)
	t := reflect.TypeOf(*e)

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		// 跳过非字符串类型字段
		if field.Kind() != reflect.String {
			continue
		}

		// 获取字段值和对应的环境变量名
		value := field.String()
		key := fieldType.Tag.Get("mapstructure")
		if key == "" || value == "" {
			continue
		}

		// 设置环境变量并检查错误
		if err := os.Setenv(key, value); err != nil {
			fmt.Printf("Failed to set environment variable %s: %v", key, err)
		}
	}

	// 应用自定义环境变量
	for key, value := range e.Custom {
		if value != "" {
			if err := os.Setenv(key, value); err != nil {
				fmt.Printf("Failed to set custom environment variable %s: %v", key, err)
			}
		}
	}
}

// Validate 验证环境变量配置的有效性
func (e *EnvConfig) Validate() []string {
	var errors []string

	// 验证 GOEXPERIMENT
	if invalid := ValidateGoExperiment(e.GoExperiment); len(invalid) > 0 {
		for _, exp := range invalid {
			errors = append(errors, "未知的 GOEXPERIMENT 选项: "+exp)
		}
	}

	// 验证 GOOS 和 GOARCH 组合
	if e.GoOS != "" && e.GoArch != "" {
		if !isValidOSArchCombination(e.GoOS, e.GoArch) {
			errors = append(errors, "不支持的 GOOS/GOARCH 组合: "+e.GoOS+"/"+e.GoArch)
		}
	}

	return errors
}

// isValidOSArchCombination 检查操作系统和架构组合是否有效
func isValidOSArchCombination(goos, goarch string) bool {
	validCombinations := map[string][]string{
		"linux":     {"386", "amd64", "arm", "arm64", "mips", "mips64", "mips64le", "mipsle", "ppc64", "ppc64le", "riscv64", "s390x"},
		"darwin":    {"amd64", "arm64"},
		"windows":   {"386", "amd64", "arm", "arm64"},
		"freebsd":   {"386", "amd64", "arm", "arm64", "riscv64"},
		"openbsd":   {"386", "amd64", "arm", "arm64", "mips64"},
		"netbsd":    {"386", "amd64", "arm", "arm64"},
		"dragonfly": {"amd64"},
		"plan9":     {"386", "amd64", "arm"},
		"solaris":   {"amd64"},
		"android":   {"386", "amd64", "arm", "arm64"},
		"ios":       {"arm64"},
		"js":        {"wasm"},
		"wasip1":    {"wasm"},
	}

	if archs, exists := validCombinations[goos]; exists {
		for _, arch := range archs {
			if arch == goarch {
				return true
			}
		}
	}
	return false
}

// getGoEnvOrDefault gets a value for a Go environment variable with fallback.
// It prioritizes:
// 1. Value from the `go env` cache.
// 2. Value from the operating system's environment variables.
// 3. The provided default value.
func getGoEnvOrDefault(key, defaultValue string) string {
	loadGoEnv() // Ensures the cache is populated on first call.

	// 1. Check our `go env` cache.
	if value, ok := goEnvCache[key]; ok && value != "" {
		return value
	}
	// 2. Check the actual OS environment variables.
	if v := os.Getenv(key); v != "" {
		return v
	}
	// 3. Fallback to the default value.
	return defaultValue
}

// GetModuleRoot returns the directory containing the provided go.mod path.
// If the provided goMod is empty, it attempts to use the GOMOD value from
// cached `go env` or environment variables; if still empty, returns an empty string.
func GetModuleRoot(goMod string) string {
	if goMod == "" {
		goMod = getGoEnvOrDefault("GOMOD", "")
	}
	if goMod == "" {
		return ""
	}
	return filepath.Dir(goMod)
}

// GetAvailableGoExperiments 获取当前Go版本支持的实验性功能列表
func GetAvailableGoExperiments() map[string]string {
	// 这些是常见的 GOEXPERIMENT 选项及其说明
	experiments := map[string]string{
		"rangefunc":         "启用 range-over-func 特性 (Go 1.22+)",
		"arenas":            "启用 arenas 内存管理实验",
		"cgocheck2":         "启用更严格的 cgo 指针检查",
		"fieldtrack":        "启用字段跟踪功能，用于分析结构体字段使用情况",
		"preemptibleloops":  "启用可抢占循环，改善调度器性能",
		"staticlockranking": "启用静态锁排序检查，帮助检测死锁",
		"boringcrypto":      "启用 BoringSSL 加密库支持",
		"unified":           "启用统一的类型检查器 (Go 1.18+)",
		"typeparams":        "启用泛型类型参数支持 (Go 1.18+)",
		"pacer":             "启用新的 GC pacer 算法",
		"checkptr":          "启用指针检查（runtime 调试）",
		"asyncpreempt":      "启用异步抢占",
		"newinliner":        "启用新的内联器",
		"coverageredesign":  "启用覆盖率重新设计",
	}
	return experiments
}

// ValidateGoExperiment 验证 GOEXPERIMENT 设置是否有效
func ValidateGoExperiment(experiment string) []string {
	if experiment == "" {
		return nil
	}

	available := GetAvailableGoExperiments()
	var invalid []string

	// 解析逗号分隔的实验选项
	for _, exp := range splitExperiment(experiment) {
		exp = trimExperiment(exp)
		if exp == "" {
			continue
		}

		// 检查是否是有效的实验选项
		if _, exists := available[exp]; !exists {
			invalid = append(invalid, exp)
		}
	}

	return invalid
}

// splitExperiment 分割实验选项字符串
func splitExperiment(experiment string) []string {
	var result []string
	start := 0
	for i, r := range experiment {
		if r == ',' {
			if i > start {
				result = append(result, experiment[start:i])
			}
			start = i + 1
		}
	}
	if len(experiment) > start {
		result = append(result, experiment[start:])
	}
	return result
}

// trimExperiment 清理实验选项字符串
func trimExperiment(exp string) string {
	// 简单的 trim 实现
	start := 0
	end := len(exp)

	for start < end && (exp[start] == ' ' || exp[start] == '\t' || exp[start] == '\n') {
		start++
	}
	for end > start && (exp[end-1] == ' ' || exp[end-1] == '\t' || exp[end-1] == '\n') {
		end--
	}

	return exp[start:end]
}
