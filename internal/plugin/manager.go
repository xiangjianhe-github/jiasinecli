// Package plugin 提供插件管理系统
// 支持通过可执行文件或共享库扩展 CLI 功能
package plugin

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/xiangjianhe-github/jiasinecli/internal/config"
	"github.com/xiangjianhe-github/jiasinecli/internal/logger"
)

// Info 插件信息
type Info struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Author      string `json:"author"`
	Enabled     bool   `json:"enabled"`
	Type        string `json:"type"` // executable, shared_lib
	EntryPoint  string `json:"entry_point"`
}

// Manager 插件管理器
type Manager struct {
	pluginDir string
}

// NewManager 创建插件管理器实例
func NewManager() *Manager {
	cfg := config.Get()
	pluginDir := cfg.Plugins.Dir
	if pluginDir == "" {
		home, _ := os.UserHomeDir()
		pluginDir = filepath.Join(home, ".jiasine", "plugins")
	}
	return &Manager{pluginDir: pluginDir}
}

// List 列出已安装的插件
func (m *Manager) List() ([]Info, error) {
	var plugins []Info

	if _, err := os.Stat(m.pluginDir); os.IsNotExist(err) {
		return plugins, nil
	}

	entries, err := os.ReadDir(m.pluginDir)
	if err != nil {
		return nil, fmt.Errorf("读取插件目录失败: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		manifestPath := filepath.Join(m.pluginDir, entry.Name(), "manifest.json")
		info, err := loadManifest(manifestPath)
		if err != nil {
			logger.Warn("加载插件清单失败", "plugin", entry.Name(), "error", err)
			continue
		}
		plugins = append(plugins, *info)
	}

	return plugins, nil
}

// Install 安装插件
func (m *Manager) Install(name string) error {
	pluginPath := filepath.Join(m.pluginDir, name)

	// 创建插件目录
	if err := os.MkdirAll(pluginPath, 0755); err != nil {
		return fmt.Errorf("创建插件目录失败: %w", err)
	}

	// 创建默认清单
	manifest := Info{
		Name:        name,
		Version:     "0.1.0",
		Description: fmt.Sprintf("%s 插件", name),
		Author:      "jiasine",
		Enabled:     true,
		Type:        "executable",
		EntryPoint:  getPluginEntryPoint(name),
	}

	manifestPath := filepath.Join(pluginPath, "manifest.json")
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化清单失败: %w", err)
	}

	if err := os.WriteFile(manifestPath, data, 0644); err != nil {
		return fmt.Errorf("写入清单失败: %w", err)
	}

	logger.Info("插件安装完成", "name", name, "path", pluginPath)
	return nil
}

// Remove 卸载插件
func (m *Manager) Remove(name string) error {
	pluginPath := filepath.Join(m.pluginDir, name)

	if _, err := os.Stat(pluginPath); os.IsNotExist(err) {
		return fmt.Errorf("插件 '%s' 不存在", name)
	}

	if err := os.RemoveAll(pluginPath); err != nil {
		return fmt.Errorf("删除插件目录失败: %w", err)
	}

	logger.Info("插件已卸载", "name", name)
	return nil
}

// loadManifest 加载插件清单文件
func loadManifest(path string) (*Info, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var info Info
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, err
	}

	return &info, nil
}

// getPluginEntryPoint 根据平台获取插件入口
func getPluginEntryPoint(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}
