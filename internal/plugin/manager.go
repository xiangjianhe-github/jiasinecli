// Package plugin 提供插件管理系统
// 插件存放在应用目录下的 plugin/ 子目录中
// 每个插件是一个目录，包含 <PluginName>.json 描述文件
// 例如: plugin/SerialTool/SerialTool.json
package plugin

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/xiangjianhe-github/jiasinecli/internal/logger"
)

// Info 插件信息
type Info struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Author      string `json:"author"`
	Icon        string `json:"icon"`        // 图标 (emoji 或图标路径)
	Enabled     bool   `json:"enabled"`
	Type        string `json:"type"`        // executable, shared_lib
	EntryPoint  string `json:"entry_point"` // 入口程序 (如 SerialTool.exe)
	Homepage    string `json:"homepage"`    // 项目主页
	Tags        []string `json:"tags"`      // 标签
	// 内部字段（不序列化到 JSON）
	Dir string `json:"-"` // 插件所在目录的绝对路径
}

// Manager 插件管理器
type Manager struct {
	pluginDir string // 应用目录下的 plugin/ 目录
}

// NewManager 创建插件管理器实例
// pluginDir 使用应用程序所在目录下的 plugin/ 子目录
func NewManager() *Manager {
	exePath, _ := os.Executable()
	exeDir := filepath.Dir(exePath)
	pluginDir := filepath.Join(exeDir, "plugin")
	return &Manager{pluginDir: pluginDir}
}

// PluginDir 返回插件目录路径
func (m *Manager) PluginDir() string {
	return m.pluginDir
}

// Scan 扫描插件目录，发现所有可用插件
// 查找格式: plugin/<Name>/<Name>.json
func (m *Manager) Scan() ([]Info, error) {
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

		dirName := entry.Name()
		pluginDir := filepath.Join(m.pluginDir, dirName)

		// 查找 <DirName>.json
		jsonPath := filepath.Join(pluginDir, dirName+".json")
		if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
			// 也尝试查找 manifest.json (兼容旧格式)
			jsonPath = filepath.Join(pluginDir, "manifest.json")
			if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
				continue
			}
		}

		info, err := loadPluginJSON(jsonPath)
		if err != nil {
			logger.Warn("加载插件描述失败", "plugin", dirName, "error", err)
			continue
		}

		// 确保名称一致
		if info.Name == "" {
			info.Name = dirName
		}
		info.Dir = pluginDir
		plugins = append(plugins, *info)
	}

	return plugins, nil
}

// Get 获取指定插件信息
func (m *Manager) Get(name string) (*Info, error) {
	plugins, err := m.Scan()
	if err != nil {
		return nil, err
	}

	nameLower := strings.ToLower(name)
	for _, p := range plugins {
		if strings.ToLower(p.Name) == nameLower {
			return &p, nil
		}
	}

	return nil, fmt.Errorf("插件 '%s' 不存在\n可用插件: %s", name, m.availableNames())
}

// Open 打开/运行指定插件
// 如果插件有 entry_point，在新的 cmd 窗口中启动
func (m *Manager) Open(name string) error {
	info, err := m.Get(name)
	if err != nil {
		return err
	}

	if !info.Enabled {
		return fmt.Errorf("插件 '%s' 已禁用", name)
	}

	if info.EntryPoint == "" {
		return fmt.Errorf("插件 '%s' 未定义入口程序 (entry_point)", name)
	}

	entryPath := filepath.Join(info.Dir, info.EntryPoint)
	if _, err := os.Stat(entryPath); os.IsNotExist(err) {
		return fmt.Errorf("插件入口程序不存在: %s", entryPath)
	}

	logger.Info("启动插件", "name", name, "entry", entryPath)

	// 在新的 cmd 窗口中打开
	if runtime.GOOS == "windows" {
		cmd := exec.Command("cmd", "/c", "start", "cmd", "/k", entryPath)
		cmd.Dir = info.Dir
		return cmd.Start()
	}

	// macOS / Linux
	if runtime.GOOS == "darwin" {
		cmd := exec.Command("open", "-a", "Terminal", entryPath)
		cmd.Dir = info.Dir
		return cmd.Start()
	}

	// Linux: 尝试常见终端
	for _, term := range []string{"gnome-terminal", "xterm", "konsole"} {
		if _, err := exec.LookPath(term); err == nil {
			cmd := exec.Command(term, "--", entryPath)
			cmd.Dir = info.Dir
			return cmd.Start()
		}
	}

	// 后备：直接启动
	cmd := exec.Command(entryPath)
	cmd.Dir = info.Dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Start()
}

// Install 从插件目录安装插件 (创建示例结构)
func (m *Manager) Install(name string) error {
	pluginPath := filepath.Join(m.pluginDir, name)

	// 检查是否已存在
	jsonPath := filepath.Join(pluginPath, name+".json")
	if _, err := os.Stat(jsonPath); err == nil {
		return fmt.Errorf("插件 '%s' 已存在", name)
	}

	// 创建插件目录
	if err := os.MkdirAll(pluginPath, 0755); err != nil {
		return fmt.Errorf("创建插件目录失败: %w", err)
	}

	// 创建插件描述文件
	info := Info{
		Name:        name,
		Version:     "1.0.0",
		Description: fmt.Sprintf("%s 插件", name),
		Author:      "",
		Icon:        "🔌",
		Enabled:     true,
		Type:        "executable",
		EntryPoint:  getPluginEntryPoint(name),
		Tags:        []string{},
	}

	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化描述文件失败: %w", err)
	}

	if err := os.WriteFile(jsonPath, data, 0644); err != nil {
		return fmt.Errorf("写入描述文件失败: %w", err)
	}

	logger.Info("插件安装完成", "name", name, "path", pluginPath)
	return nil
}

// Remove 卸载插件
func (m *Manager) Remove(name string) error {
	plugins, err := m.Scan()
	if err != nil {
		return err
	}

	nameLower := strings.ToLower(name)
	for _, p := range plugins {
		if strings.ToLower(p.Name) == nameLower {
			if err := os.RemoveAll(p.Dir); err != nil {
				return fmt.Errorf("删除插件目录失败: %w", err)
			}
			logger.Info("插件已卸载", "name", name)
			return nil
		}
	}

	return fmt.Errorf("插件 '%s' 不存在", name)
}

// availableNames 返回所有可用插件名称
func (m *Manager) availableNames() string {
	plugins, _ := m.Scan()
	if len(plugins) == 0 {
		return "(无)"
	}
	names := make([]string, len(plugins))
	for i, p := range plugins {
		names[i] = p.Name
	}
	return strings.Join(names, ", ")
}

// loadPluginJSON 加载插件 JSON 描述文件
func loadPluginJSON(path string) (*Info, error) {
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

// getPluginEntryPoint 根据平台获取默认入口名
func getPluginEntryPoint(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}
