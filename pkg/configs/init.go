package configs

import (
	"strings"

	"github.com/spf13/viper"
	"github.com/yeisme/gocli/pkg/models"
	"github.com/yeisme/gocli/pkg/utils/executor"
)

// InitConfig newproject.InitOptions 的别名，简化引用
type InitConfig = models.InitOptions

func setInitConfigDefaults() {
	viper.SetDefault("init.git_init", false)
	viper.SetDefault("init.license", "")
	viper.SetDefault("init.dir", ".")
	viper.SetDefault("init.gocli", true)
	viper.SetDefault("init.goreleaser", false)
	viper.SetDefault("init.go_task", false)
	viper.SetDefault("init.docker", false)
	viper.SetDefault("init.makefile", false)
	viper.SetDefault("init.author", getFromGit("user.name"))
	viper.SetDefault("init.email", getFromGit("user.email"))
}

func getFromGit(key string) string {
	// 尝试按优先级读取 Git 配置：先读取仓库/用户（local or worktree）级别，
	// 如果未找到再尝试读取 global 级别
	// 这样在用户在全局配置中设置了 user.name/user.email 时也能正确读取
	attempts := [][]string{
		{"config", "--get", key},
		{"config", "--global", "--get", key},
	}

	for _, args := range attempts {
		exe := executor.NewExecutor("git", args...)
		output, err := exe.CombinedOutput()
		if err != nil {
			// 如果命令失败，继续尝试下一个来源
			continue
		}
		val := strings.TrimSpace(string(output))
		if val != "" {
			return val
		}
	}
	return ""
}
