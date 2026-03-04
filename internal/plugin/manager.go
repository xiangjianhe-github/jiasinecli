// Package plugin 提供插件管理系统（远程优先 + 本地回退）
//
// 插件市场架构:
//   - 优先从远程 GitHub 仓库获取插件列表
//   - 服务器不可达时回退到本地 plugin/ 目录
//   - 每个插件 = <Name>.json 描述文件 + <Name>.7z 压缩包
//   - 安装时下载并解压 .7z 到 plugin/<Name>/ 目录
package plugin

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/xiangjianhe-github/jiasinecli/internal/logger"
)

const (
	// GitHub API 列出 plugin/ 目录下的文件
	githubAPIURL = "https://api.github.com/repos/xiangjianhe-github/jiasinecli/contents/plugin"
	// 下载原始文件
	githubRawURL = "https://raw.githubusercontent.com/xiangjianhe-github/jiasinecli/main/plugin"
	// HTTP 超时
	httpTimeout = 10 * time.Second
)

// Source 插件来源
type Source string

const (
	SourceRemote Source = "remote" // 远程服务器
	SourceLocal  Source = "local"  // 本地目录
)

// Info 插件信息
type Info struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Description string   `json:"description"`
	Author      string   `json:"author"`
	Icon        string   `json:"icon"`        // 图标 (emoji 或图标路径)
	Enabled     bool     `json:"enabled"`
	Type        string   `json:"type"`        // executable, shared_lib
	EntryPoint  string   `json:"entry_point"` // 入口程序 (如 SerialTool.exe)
	Homepage    string   `json:"homepage"`    // 项目主页
	Tags        []string `json:"tags"`        // 标签
	// 内部字段（不序列化到 JSON）
	Dir       string `json:"-"` // 插件所在目录的绝对路径 (已安装时有值)
	Source    Source `json:"-"` // 来源: remote / local
	Installed bool   `json:"-"` // 是否已安装
}

// githubContent GitHub Contents API 响应项
type githubContent struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Type        string `json:"type"` // "file" or "dir"
	DownloadURL string `json:"download_url"`
}

// Manager 插件管理器
type Manager struct {
	pluginDir string // 应用目录下的 plugin/ 目录
	client    *http.Client
}

// NewManager 创建插件管理器实例
func NewManager() *Manager {
	exePath, _ := os.Executable()
	exeDir := filepath.Dir(exePath)
	pluginDir := filepath.Join(exeDir, "plugin")
	return &Manager{
		pluginDir: pluginDir,
		client: &http.Client{
			Timeout: httpTimeout,
		},
	}
}

// PluginDir 返回插件目录路径
func (m *Manager) PluginDir() string {
	return m.pluginDir
}

// ---------------------------------------------------------------------------
// 插件市场：远程优先 + 本地回退
// ---------------------------------------------------------------------------

// Marketplace 获取插件市场列表（远程优先，本地回退）
// 返回可用插件列表和来源标识
func (m *Manager) Marketplace() ([]Info, Source, error) {
	// 1. 尝试远程
	plugins, err := m.fetchRemotePlugins()
	if err == nil && len(plugins) > 0 {
		// 标记已安装状态
		m.markInstalled(plugins)
		return plugins, SourceRemote, nil
	}
	if err != nil {
		logger.Warn("远程插件市场不可达，回退到本地", "error", err)
	}

	// 2. 回退到本地
	plugins, err = m.scanLocalMarketplace()
	if err != nil {
		return nil, SourceLocal, fmt.Errorf("获取插件列表失败: %w", err)
	}
	m.markInstalled(plugins)
	return plugins, SourceLocal, nil
}

// fetchRemotePlugins 从 GitHub API 获取远程插件列表
func (m *Manager) fetchRemotePlugins() ([]Info, error) {
	req, err := http.NewRequest("GET", githubAPIURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "JiasineCLI")

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("连接服务器失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("服务器返回 %d", resp.StatusCode)
	}

	var contents []githubContent
	if err := json.NewDecoder(resp.Body).Decode(&contents); err != nil {
		return nil, fmt.Errorf("解析目录列表失败: %w", err)
	}

	// 收集所有 .json 文件名 (排除 .7z)
	var jsonFiles []string
	for _, c := range contents {
		if c.Type == "file" && strings.HasSuffix(c.Name, ".json") {
			jsonFiles = append(jsonFiles, c.Name)
		}
	}

	// 下载每个 JSON 描述文件
	var plugins []Info
	for _, jf := range jsonFiles {
		info, err := m.downloadPluginJSON(jf)
		if err != nil {
			logger.Warn("下载插件描述失败", "file", jf, "error", err)
			continue
		}
		info.Source = SourceRemote
		plugins = append(plugins, *info)
	}

	return plugins, nil
}

// downloadPluginJSON 从远程下载单个插件 JSON 描述
func (m *Manager) downloadPluginJSON(filename string) (*Info, error) {
	url := githubRawURL + "/" + filename

	resp, err := m.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("下载 %s 失败: %d", filename, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var info Info
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("解析 %s 失败: %w", filename, err)
	}

	// 从文件名推断名称
	if info.Name == "" {
		info.Name = strings.TrimSuffix(filename, ".json")
	}

	return &info, nil
}

// scanLocalMarketplace 扫描本地 plugin/ 目录下的 *.json 文件（平铺格式）
func (m *Manager) scanLocalMarketplace() ([]Info, error) {
	var plugins []Info

	if _, err := os.Stat(m.pluginDir); os.IsNotExist(err) {
		return plugins, nil
	}

	entries, err := os.ReadDir(m.pluginDir)
	if err != nil {
		return nil, fmt.Errorf("读取插件目录失败: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue // 跳过子目录（那是已安装的插件）
		}
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		jsonPath := filepath.Join(m.pluginDir, entry.Name())
		info, err := loadPluginJSON(jsonPath)
		if err != nil {
			logger.Warn("加载本地插件描述失败", "file", entry.Name(), "error", err)
			continue
		}

		if info.Name == "" {
			info.Name = strings.TrimSuffix(entry.Name(), ".json")
		}
		info.Source = SourceLocal
		plugins = append(plugins, *info)
	}

	return plugins, nil
}

// markInstalled 标记市场列表中已安装的插件
func (m *Manager) markInstalled(plugins []Info) {
	for i := range plugins {
		installDir := filepath.Join(m.pluginDir, plugins[i].Name)
		if fi, err := os.Stat(installDir); err == nil && fi.IsDir() {
			plugins[i].Installed = true
			plugins[i].Dir = installDir
		}
	}
}

// ---------------------------------------------------------------------------
// 已安装插件管理
// ---------------------------------------------------------------------------

// ScanInstalled 扫描已安装插件 (plugin/<Name>/<Name>.json 子目录格式)
func (m *Manager) ScanInstalled() ([]Info, error) {
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
			// 也尝试查找 manifest.json (兼容)
			jsonPath = filepath.Join(pluginDir, "manifest.json")
			if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
				continue
			}
		}

		info, err := loadPluginJSON(jsonPath)
		if err != nil {
			logger.Warn("加载已安装插件描述失败", "plugin", dirName, "error", err)
			continue
		}

		if info.Name == "" {
			info.Name = dirName
		}
		info.Dir = pluginDir
		info.Installed = true
		plugins = append(plugins, *info)
	}

	return plugins, nil
}

// Get 获取指定已安装插件信息
func (m *Manager) Get(name string) (*Info, error) {
	plugins, err := m.ScanInstalled()
	if err != nil {
		return nil, err
	}

	nameLower := strings.ToLower(name)
	for _, p := range plugins {
		if strings.ToLower(p.Name) == nameLower {
			return &p, nil
		}
	}

	return nil, fmt.Errorf("插件 '%s' 未安装\n已安装: %s\n提示: 使用 plugin view 查看可用插件，plugin install <名称> 安装", name, m.installedNames())
}

// Open 打开/运行指定已安装插件
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
		return fmt.Errorf("插件入口程序不存在: %s\n提示: 请将 %s 放入 %s", entryPath, info.EntryPoint, info.Dir)
	}

	logger.Info("启动插件", "name", name, "entry", entryPath)

	// 在新的终端窗口中打开
	if runtime.GOOS == "windows" {
		cmd := exec.Command("cmd", "/c", "start", "cmd", "/k", entryPath)
		cmd.Dir = info.Dir
		return cmd.Start()
	}

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

// ---------------------------------------------------------------------------
// 安装 / 卸载
// ---------------------------------------------------------------------------

// Install 安装插件（从远程下载或从本地安装）
// 流程: 下载 .json + .7z → 解压到 plugin/<Name>/
func (m *Manager) Install(name string) error {
	// 检查是否已安装
	installDir := filepath.Join(m.pluginDir, name)
	if fi, err := os.Stat(installDir); err == nil && fi.IsDir() {
		return fmt.Errorf("插件 '%s' 已安装 (目录: %s)", name, installDir)
	}

	// 确保 plugin 目录存在
	if err := os.MkdirAll(m.pluginDir, 0755); err != nil {
		return fmt.Errorf("创建插件目录失败: %w", err)
	}

	jsonFile := name + ".json"
	archiveFile := name + ".7z"

	// 尝试从远程下载
	remoteOK := false
	jsonURL := githubRawURL + "/" + jsonFile
	archiveURL := githubRawURL + "/" + archiveFile

	// 1. 下载 JSON 到本地
	localJSON := filepath.Join(m.pluginDir, jsonFile)
	if err := m.downloadFile(jsonURL, localJSON); err != nil {
		logger.Warn("远程下载 JSON 失败，尝试本地", "error", err)
	} else {
		remoteOK = true
	}

	// 2. 下载 7z 到本地
	localArchive := filepath.Join(m.pluginDir, archiveFile)
	if remoteOK {
		if err := m.downloadFile(archiveURL, localArchive); err != nil {
			logger.Warn("远程下载压缩包失败", "error", err)
			// JSON 下载成功但 7z 失败 → 检查本地是否有 7z
			if _, err := os.Stat(localArchive); os.IsNotExist(err) {
				return fmt.Errorf("下载插件压缩包失败: %s 不存在", archiveFile)
			}
		}
	}

	// 3. 检查本地文件是否就绪
	if _, err := os.Stat(localJSON); os.IsNotExist(err) {
		return fmt.Errorf("插件描述文件不存在: %s\n请确认远程或本地 plugin/ 目录下有 %s", jsonFile, jsonFile)
	}
	if _, err := os.Stat(localArchive); os.IsNotExist(err) {
		return fmt.Errorf("插件压缩包不存在: %s\n请确认远程或本地 plugin/ 目录下有 %s", archiveFile, archiveFile)
	}

	// 4. 创建安装目录
	if err := os.MkdirAll(installDir, 0755); err != nil {
		return fmt.Errorf("创建安装目录失败: %w", err)
	}

	// 5. 解压 .7z 到安装目录
	if err := m.extract7z(localArchive, installDir); err != nil {
		// 解压失败 → 清理安装目录
		os.RemoveAll(installDir)
		return fmt.Errorf("解压插件失败: %w", err)
	}

	// 6. 复制 JSON 描述到安装目录
	destJSON := filepath.Join(installDir, jsonFile)
	if _, err := os.Stat(destJSON); os.IsNotExist(err) {
		// 如果 7z 里没有 JSON，从外面复制进去
		data, _ := os.ReadFile(localJSON)
		if err := os.WriteFile(destJSON, data, 0644); err != nil {
			logger.Warn("复制描述文件到安装目录失败", "error", err)
		}
	}

	logger.Info("插件安装完成", "name", name, "path", installDir)
	return nil
}

// Remove 卸载已安装插件
func (m *Manager) Remove(name string) error {
	plugins, err := m.ScanInstalled()
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

	return fmt.Errorf("插件 '%s' 未安装", name)
}

// ---------------------------------------------------------------------------
// 辅助方法
// ---------------------------------------------------------------------------

// downloadFile 下载远程文件到本地路径
func (m *Manager) downloadFile(url, destPath string) error {
	resp, err := m.client.Get(url)
	if err != nil {
		return fmt.Errorf("下载失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载 %s: HTTP %d", url, resp.StatusCode)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}

	return nil
}

// extract7z 解压 .7z 文件到目标目录
// 依赖系统安装的 7z 命令行工具
func (m *Manager) extract7z(archivePath, destDir string) error {
	// 查找 7z 可执行文件
	sevenZip := find7zExecutable()
	if sevenZip == "" {
		return fmt.Errorf("未找到 7z 解压工具\n请安装 7-Zip: https://www.7-zip.org/\n或将 7z.exe 添加到 PATH 环境变量")
	}

	// 7z x archive.7z -oDestDir -y
	cmd := exec.Command(sevenZip, "x", archivePath, "-o"+destDir, "-y")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("7z 解压失败: %w\n输出: %s", err, string(output))
	}

	return nil
}

// find7zExecutable 查找 7z 可执行文件
func find7zExecutable() string {
	// 1. 尝试 PATH 中的 7z
	if p, err := exec.LookPath("7z"); err == nil {
		return p
	}

	// 2. Windows 常见安装路径
	if runtime.GOOS == "windows" {
		candidates := []string{
			`C:\Program Files\7-Zip\7z.exe`,
			`C:\Program Files (x86)\7-Zip\7z.exe`,
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				return c
			}
		}
	}

	return ""
}

// installedNames 返回所有已安装插件名称
func (m *Manager) installedNames() string {
	plugins, _ := m.ScanInstalled()
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
