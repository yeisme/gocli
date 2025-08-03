package configs

import (
	"os"
	"runtime"

	"github.com/spf13/viper"
)

// EnvConfig 环境变量配置
type EnvConfig struct {
	// Go 核心环境变量
	GoRoot     string `mapstructure:"GOROOT"`     // Go 安装路径
	GoPath     string `mapstructure:"GOPATH"`     // Go 工作空间路径
	GoModDir   string `mapstructure:"GOMODDIR"`   // Go modules 目录
	GoModCache string `mapstructure:"GOMODCACHE"` // Go modules 缓存目录
	GoSumDB    string `mapstructure:"GOSUMDB"`    // Go checksum 数据库
	GoProxy    string `mapstructure:"GOPROXY"`    // Go 模块代理
	GoPrivate  string `mapstructure:"GOPRIVATE"`  // 私有模块路径模式
	GoNoProxy  string `mapstructure:"GONOPROXY"`  // 不使用代理的模块路径模式
	GoNoSumDB  string `mapstructure:"GONOSUMDB"`  // 不使用 checksum 数据库的模块路径模式

	// 构建相关环境变量
	GoOS     string `mapstructure:"GOOS"`     // 目标操作系统
	GoArch   string `mapstructure:"GOARCH"`   // 目标架构
	Go386    string `mapstructure:"GO386"`    // 386 架构设置
	GoAMD64  string `mapstructure:"GOAMD64"`  // AMD64 架构设置
	GoARM    string `mapstructure:"GOARM"`    // ARM 架构设置
	GoARM64  string `mapstructure:"GOARM64"`  // ARM64 架构设置
	GoMIPS   string `mapstructure:"GOMIPS"`   // MIPS 架构设置
	GoMIPS64 string `mapstructure:"GOMIPS64"` // MIPS64 架构设置
	GoPPC64  string `mapstructure:"GOPPC64"`  // PPC64 架构设置
	GoWASM   string `mapstructure:"GOWASM"`   // WebAssembly 设置

	// 编译器相关环境变量
	GoGCFlags  string `mapstructure:"GOGCFLAGS"`  // Go 编译器标志
	GoAsmFlags string `mapstructure:"GOASMFLAGS"` // Go 汇编器标志
	GoLDFlags  string `mapstructure:"GOLDFLAGS"`  // Go 链接器标志
	GoFlags    string `mapstructure:"GOFLAGS"`    // Go 命令标志
	GoInsecure string `mapstructure:"GOINSECURE"` // 允许不安全的 scheme

	// CGO 相关环境变量
	CGOEnabled  string `mapstructure:"CGO_ENABLED"`
	CGOCFlags   string `mapstructure:"CGO_CFLAGS"`
	CGOCPPFlags string `mapstructure:"CGO_CPPFLAGS"`
	CGOLDFlags  string `mapstructure:"CGO_LDFLAGS"`
	CGOCXXFlags string `mapstructure:"CGO_CXXFLAGS"`

	// 调试和性能相关环境变量
	GoTrace        string `mapstructure:"GOTRACE"`        // 跟踪执行
	GoDebug        string `mapstructure:"GODEBUG"`        // 调试设置
	GoMemProfile   string `mapstructure:"GOMEMPROFILE"`   // 内存分析文件
	GoCPUProfile   string `mapstructure:"GOCPUPROFILE"`   // CPU 分析文件
	GoBlockProfile string `mapstructure:"GOBLOCKPROFILE"` // 阻塞分析文件
	GoMutexProfile string `mapstructure:"GOMUTEXPROFILE"` // 互斥锁分析文件

	// 工具链相关环境变量
	GoToolchain string `mapstructure:"GOTOOLCHAIN"` // Go 工具链版本
	GoToolDir   string `mapstructure:"GOTOOLDIR"`   // Go 工具目录
	GoCache     string `mapstructure:"GOCACHE"`     // 构建缓存目录
	GoTmpDir    string `mapstructure:"GOTMPDIR"`    // 临时目录
	GoWork      string `mapstructure:"GOWORK"`      // Go 工作空间文件
	GoWorkSum   string `mapstructure:"GOWORKSUM"`   // Go 工作空间校验和文件

	// 实验性功能环境变量
	GoExperiment string `mapstructure:"GOEXPERIMENT"` // 实验性功能
	// 常用的 GOEXPERIMENT 选项：
	// - "rangefunc": 启用 range-over-func 特性
	// - "arenas": 启用 arenas 内存管理实验
	// - "cgocheck2": 启用更严格的 cgo 指针检查
	// - "fieldtrack": 启用字段跟踪功能
	// - "preemptibleloops": 启用可抢占循环
	// - "staticlockranking": 启用静态锁排序检查
	// 可以组合使用，用逗号分隔，如: "rangefunc,arenas"

	// 其他环境变量可以通过 map 存储
	Custom map[string]string `mapstructure:",remain"`
}

// setEnvConfigDefaults 设置环境变量配置的默认值
func setEnvConfigDefaults() {
	// Go 核心环境变量默认值 - 从当前环境或Go SDK获取
	if goroot := os.Getenv("GOROOT"); goroot != "" {
		viper.SetDefault("env.GOROOT", goroot)
	}
	if gopath := os.Getenv("GOPATH"); gopath != "" {
		viper.SetDefault("env.GOPATH", gopath)
	}

	// 模块相关环境变量默认值
	viper.SetDefault("env.GOPROXY", getEnvWithDefault("GOPROXY", "https://proxy.golang.org,direct"))
	viper.SetDefault("env.GOSUMDB", getEnvWithDefault("GOSUMDB", "sum.golang.org"))
	viper.SetDefault("env.GOPRIVATE", os.Getenv("GOPRIVATE"))
	viper.SetDefault("env.GONOPROXY", os.Getenv("GONOPROXY"))
	viper.SetDefault("env.GONOSUMDB", os.Getenv("GONOSUMDB"))

	// 构建相关环境变量默认值 - 从runtime获取当前值
	viper.SetDefault("env.GOOS", getEnvWithDefault("GOOS", runtime.GOOS))
	viper.SetDefault("env.GOARCH", getEnvWithDefault("GOARCH", runtime.GOARCH))
	viper.SetDefault("env.GO386", getEnvWithDefault("GO386", "sse2"))
	viper.SetDefault("env.GOAMD64", getEnvWithDefault("GOAMD64", "v1"))
	viper.SetDefault("env.GOARM", getEnvWithDefault("GOARM", "6"))
	viper.SetDefault("env.GOMIPS", getEnvWithDefault("GOMIPS", "hardfloat"))
	viper.SetDefault("env.GOMIPS64", getEnvWithDefault("GOMIPS64", "hardfloat"))
	viper.SetDefault("env.GOPPC64", getEnvWithDefault("GOPPC64", "power8"))

	// 编译器相关环境变量默认值
	viper.SetDefault("env.GOFLAGS", os.Getenv("GOFLAGS"))
	viper.SetDefault("env.GOGCFLAGS", os.Getenv("GOGCFLAGS"))
	viper.SetDefault("env.GOASMFLAGS", os.Getenv("GOASMFLAGS"))
	viper.SetDefault("env.GOLDFLAGS", os.Getenv("GOLDFLAGS"))
	viper.SetDefault("env.GOINSECURE", os.Getenv("GOINSECURE"))

	// CGO 相关环境变量默认值
	viper.SetDefault("env.CGO_ENABLED", getEnvWithDefault("CGO_ENABLED", "1"))
	viper.SetDefault("env.CGO_CFLAGS", getEnvWithDefault("CGO_CFLAGS", "-g -O2"))
	viper.SetDefault("env.CGO_CPPFLAGS", getEnvWithDefault("CGO_CPPFLAGS", "-g -O2"))
	viper.SetDefault("env.CGO_LDFLAGS", getEnvWithDefault("CGO_LDFLAGS", "-g -O2"))
	viper.SetDefault("env.CGO_CXXFLAGS", getEnvWithDefault("CGO_CXXFLAGS", "-g -O2"))

	// 调试和性能相关环境变量默认值
	viper.SetDefault("env.GODEBUG", os.Getenv("GODEBUG"))
	viper.SetDefault("env.GOTRACE", os.Getenv("GOTRACE"))

	// 工具链相关环境变量默认值
	viper.SetDefault("env.GOTOOLCHAIN", getEnvWithDefault("GOTOOLCHAIN", "auto"))
	viper.SetDefault("env.GOTOOLDIR", os.Getenv("GOTOOLDIR"))
	viper.SetDefault("env.GOCACHE", os.Getenv("GOCACHE"))
	viper.SetDefault("env.GOTMPDIR", os.Getenv("GOTMPDIR"))
	viper.SetDefault("env.GOWORK", getEnvWithDefault("GOWORK", "auto"))
	viper.SetDefault("env.GOWORKSUM", os.Getenv("GOWORKSUM"))

	// 实验性功能环境变量默认值
	viper.SetDefault("env.GOEXPERIMENT", os.Getenv("GOEXPERIMENT"))
}

// ApplyEnvVars 应用环境变量到当前进程
func (e *EnvConfig) ApplyEnvVars() {
	// 应用 Go 核心环境变量
	if e.GoRoot != "" {
		os.Setenv("GOROOT", e.GoRoot)
	}
	if e.GoPath != "" {
		os.Setenv("GOPATH", e.GoPath)
	}
	if e.GoModDir != "" {
		os.Setenv("GOMODDIR", e.GoModDir)
	}
	if e.GoModCache != "" {
		os.Setenv("GOMODCACHE", e.GoModCache)
	}
	if e.GoSumDB != "" {
		os.Setenv("GOSUMDB", e.GoSumDB)
	}
	if e.GoProxy != "" {
		os.Setenv("GOPROXY", e.GoProxy)
	}
	if e.GoPrivate != "" {
		os.Setenv("GOPRIVATE", e.GoPrivate)
	}
	if e.GoNoProxy != "" {
		os.Setenv("GONOPROXY", e.GoNoProxy)
	}
	if e.GoNoSumDB != "" {
		os.Setenv("GONOSUMDB", e.GoNoSumDB)
	}

	// 应用构建相关环境变量
	if e.GoOS != "" {
		os.Setenv("GOOS", e.GoOS)
	}
	if e.GoArch != "" {
		os.Setenv("GOARCH", e.GoArch)
	}
	if e.Go386 != "" {
		os.Setenv("GO386", e.Go386)
	}
	if e.GoAMD64 != "" {
		os.Setenv("GOAMD64", e.GoAMD64)
	}
	if e.GoARM != "" {
		os.Setenv("GOARM", e.GoARM)
	}
	if e.GoARM64 != "" {
		os.Setenv("GOARM64", e.GoARM64)
	}
	if e.GoMIPS != "" {
		os.Setenv("GOMIPS", e.GoMIPS)
	}
	if e.GoMIPS64 != "" {
		os.Setenv("GOMIPS64", e.GoMIPS64)
	}
	if e.GoPPC64 != "" {
		os.Setenv("GOPPC64", e.GoPPC64)
	}
	if e.GoWASM != "" {
		os.Setenv("GOWASM", e.GoWASM)
	}

	// 应用编译器相关环境变量
	if e.GoGCFlags != "" {
		os.Setenv("GOGCFLAGS", e.GoGCFlags)
	}
	if e.GoAsmFlags != "" {
		os.Setenv("GOASMFLAGS", e.GoAsmFlags)
	}
	if e.GoLDFlags != "" {
		os.Setenv("GOLDFLAGS", e.GoLDFlags)
	}
	if e.GoFlags != "" {
		os.Setenv("GOFLAGS", e.GoFlags)
	}
	if e.GoInsecure != "" {
		os.Setenv("GOINSECURE", e.GoInsecure)
	}

	// 应用 CGO 相关环境变量
	if e.CGOEnabled != "" {
		os.Setenv("CGO_ENABLED", e.CGOEnabled)
	}
	if e.CGOCFlags != "" {
		os.Setenv("CGO_CFLAGS", e.CGOCFlags)
	}
	if e.CGOCPPFlags != "" {
		os.Setenv("CGO_CPPFLAGS", e.CGOCPPFlags)
	}
	if e.CGOLDFlags != "" {
		os.Setenv("CGO_LDFLAGS", e.CGOLDFlags)
	}
	if e.CGOCXXFlags != "" {
		os.Setenv("CGO_CXXFLAGS", e.CGOCXXFlags)
	}

	// 应用调试和性能相关环境变量
	if e.GoTrace != "" {
		os.Setenv("GOTRACE", e.GoTrace)
	}
	if e.GoDebug != "" {
		os.Setenv("GODEBUG", e.GoDebug)
	}
	if e.GoMemProfile != "" {
		os.Setenv("GOMEMPROFILE", e.GoMemProfile)
	}
	if e.GoCPUProfile != "" {
		os.Setenv("GOCPUPROFILE", e.GoCPUProfile)
	}
	if e.GoBlockProfile != "" {
		os.Setenv("GOBLOCKPROFILE", e.GoBlockProfile)
	}
	if e.GoMutexProfile != "" {
		os.Setenv("GOMUTEXPROFILE", e.GoMutexProfile)
	}

	// 应用工具链相关环境变量
	if e.GoToolchain != "" {
		os.Setenv("GOTOOLCHAIN", e.GoToolchain)
	}
	if e.GoToolDir != "" {
		os.Setenv("GOTOOLDIR", e.GoToolDir)
	}
	if e.GoCache != "" {
		os.Setenv("GOCACHE", e.GoCache)
	}
	if e.GoTmpDir != "" {
		os.Setenv("GOTMPDIR", e.GoTmpDir)
	}
	if e.GoWork != "" {
		os.Setenv("GOWORK", e.GoWork)
	}
	if e.GoWorkSum != "" {
		os.Setenv("GOWORKSUM", e.GoWorkSum)
	}

	// 应用实验性功能环境变量
	if e.GoExperiment != "" {
		os.Setenv("GOEXPERIMENT", e.GoExperiment)
	}

	// 应用自定义环境变量
	for key, value := range e.Custom {
		if value != "" {
			os.Setenv(key, value)
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

// getEnvWithDefault 获取环境变量，如果不存在则返回默认值
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// InitEnvDefaults 初始化环境变量默认值（公开函数）
func InitEnvDefaults() {
	setEnvConfigDefaults()
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
