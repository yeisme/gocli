package models

import (
	"time"
)

// PluginSource defines the source type of a plugin.
//
// The following constants represent the possible sources of a plugin:
// - SourceUserHome: User's home directory.
// - SourceCurrentDir: Current working directory.
// - SourceConfig: Specified in the configuration file.
type PluginSource string

const (
	// SourceUserHome 用户主目录
	SourceUserHome PluginSource = "user-home"
	// SourceCurrentDir 当前目录
	SourceCurrentDir PluginSource = "current-dir"
	// SourceConfig 配置文件指定
	SourceConfig PluginSource = "config"
)

// PluginInfo 插件信息
type PluginInfo struct {
	Name         string       `json:"name"`        // 插件名称
	Path         string       `json:"path"`        // 插件完整路径
	Source       PluginSource `json:"source"`      // 插件来源
	SourcePath   string       `json:"source_path"` // 来源路径
	Size         int64        `json:"size"`        // 文件大小
	ModTime      time.Time    `json:"mod_time"`    // 修改时间
	IsExecutable bool         `json:"executable"`  // 是否可执行
}

// GetDisplayName 获取插件显示名称
func (p *PluginInfo) GetDisplayName() string {
	if len(p.Name) > 6 && p.Name[:6] == "gocli-" {
		return p.Name[6:] // 去掉 "gocli-" 前缀
	}
	return p.Name
}
