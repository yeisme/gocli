// Package plugin 提供插件管理功能
package plugin

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/yeisme/gocli/pkg/models"
)

// Manager 插件管理器
type Manager struct {
	configPluginPath string
}

// NewPluginManager 创建插件管理器
func NewPluginManager(configPluginPath string) *Manager {
	return &Manager{
		configPluginPath: configPluginPath,
	}
}

// FindAllPlugins 查找所有插件
func (pm *Manager) FindAllPlugins() ([]*models.PluginInfo, error) {
	var allPlugins []*models.PluginInfo

	// 1. 查找用户主目录插件
	userPlugins, err := pm.findPluginsInUserHome()
	if err == nil {
		allPlugins = append(allPlugins, userPlugins...)
	}

	// 2. 查找当前目录插件
	currentDirPlugins, err := pm.findPluginsInCurrentDir()
	if err == nil {
		allPlugins = append(allPlugins, currentDirPlugins...)
	}

	// 3. 查找配置文件指定目录插件
	configPlugins, err := pm.findPluginsInConfigPath()
	if err == nil {
		allPlugins = append(allPlugins, configPlugins...)
	}

	// 去重处理
	return pm.deduplicatePlugins(allPlugins), nil
}

// findPluginsInUserHome 在用户主目录查找插件
func (pm *Manager) findPluginsInUserHome() ([]*models.PluginInfo, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	pluginDir := filepath.Join(home, ".gocli", "plugins")
	return pm.findPluginsInDirectory(pluginDir, models.SourceUserHome, pluginDir)
}

// findPluginsInCurrentDir 在当前目录查找插件
func (pm *Manager) findPluginsInCurrentDir() ([]*models.PluginInfo, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	pluginDir := filepath.Join(cwd, ".gocli", "plugins")
	return pm.findPluginsInDirectory(pluginDir, models.SourceCurrentDir, pluginDir)
}

// findPluginsInConfigPath 在配置文件指定目录查找插件
func (pm *Manager) findPluginsInConfigPath() ([]*models.PluginInfo, error) {
	if pm.configPluginPath == "" {
		return nil, nil
	}

	return pm.findPluginsInDirectory(pm.configPluginPath, models.SourceConfig, pm.configPluginPath)
}

// findPluginsInDirectory 在指定目录查找插件
func (pm *Manager) findPluginsInDirectory(dir string, source models.PluginSource, sourcePath string) ([]*models.PluginInfo, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var plugins []*models.PluginInfo
	for _, f := range files {
		name := f.Name()
		if !strings.HasPrefix(name, "gocli-") {
			continue
		}

		fullPath := filepath.Join(dir, name)
		if !isExecutable(fullPath) {
			continue
		}

		info, err := f.Info()
		if err != nil {
			continue
		}

		plugin := &models.PluginInfo{
			Name:         name,
			Path:         fullPath,
			Source:       source,
			SourcePath:   sourcePath,
			Size:         info.Size(),
			ModTime:      info.ModTime(),
			IsExecutable: true,
		}

		plugins = append(plugins, plugin)
	}

	return plugins, nil
}

// deduplicatePlugins 去重插件（优先级：用户主目录 > 当前目录 > 配置文件）
func (pm *Manager) deduplicatePlugins(plugins []*models.PluginInfo) []*models.PluginInfo {
	seen := make(map[string]*models.PluginInfo)

	// 定义优先级
	priority := map[models.PluginSource]int{
		models.SourceUserHome:   1,
		models.SourceCurrentDir: 2,
		models.SourceConfig:     3,
	}

	for _, plugin := range plugins {
		existing, exists := seen[plugin.Name]
		if !exists || priority[plugin.Source] < priority[existing.Source] {
			seen[plugin.Name] = plugin
		}
	}

	var result []*models.PluginInfo
	for _, plugin := range seen {
		result = append(result, plugin)
	}

	return result
}

// 判断文件是否可执行
func isExecutable(path string) bool {
	// 检查文件是否存在且是一个文件
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	// Windows 上检查 .exe .com .bat 等可执行文件
	if runtime.GOOS == "windows" {
		return strings.HasSuffix(strings.ToLower(path), ".exe") ||
			strings.HasSuffix(strings.ToLower(path), ".com") ||
			strings.HasSuffix(strings.ToLower(path), ".bat")
	}
	// Unix-like 系统检查文件权限
	return info.Mode()&0111 != 0
}
