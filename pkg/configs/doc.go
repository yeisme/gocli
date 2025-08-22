package configs

import (
	"github.com/spf13/viper"
	"github.com/yeisme/gocli/pkg/utils/doc"
)

// DocConfig is an alias to the shared doc.Options to avoid duplication.
type DocConfig = doc.Options

func setDocConfigDefaults() {
	// 设置文档配置的默认值
	viper.SetDefault("doc.output", "")
	viper.SetDefault("doc.style", doc.StylePlain)
	viper.SetDefault("doc.mode", doc.ModeGodoc)
	viper.SetDefault("doc.include_private", false)
	viper.SetDefault("doc.title", "")
	viper.SetDefault("doc.toc", true)
	viper.SetDefault("doc.verbose", false)
	viper.SetDefault("doc.detailed", false)
	viper.SetDefault("doc.theme", "")
	viper.SetDefault("doc.width", 0)
	viper.SetDefault("doc.include_tests", false)
	viper.SetDefault("doc.include_examples", false)
}
