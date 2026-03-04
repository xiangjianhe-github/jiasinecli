// Package bridge 提供 FFI 动态库桥接层
// 通过 FFI/CGo 调用 C/Rust/Objective-C/.NET Native AOT 编译的动态库
package bridge

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/xiangjianhe-github/jiasinecli/internal/config"
	"github.com/xiangjianhe-github/jiasinecli/internal/logger"
)

// LibInfo 动态库信息
type LibInfo struct {
	Name     string `json:"name"`
	Type     string `json:"type"`     // c, rust, dotnet
	Platform string `json:"platform"` // windows, darwin, linux
	Path     string `json:"path"`
}

// Manager 桥接管理器
type Manager struct {
	bridges map[string]config.BridgeConfig
}

// NewManager 创建桥接管理器
func NewManager() *Manager {
	cfg := config.Get()
	return &Manager{
		bridges: cfg.Bridges,
	}
}

// List 列出已配置的动态库
func (m *Manager) List() ([]LibInfo, error) {
	var libs []LibInfo

	for name, bridge := range m.bridges {
		libPath := m.resolveLibPath(bridge)
		libs = append(libs, LibInfo{
			Name:     name,
			Type:     bridge.Type,
			Platform: runtime.GOOS,
			Path:     libPath,
		})
	}

	return libs, nil
}

// Call 调用动态库函数
func (m *Manager) Call(libName, funcName string, params []string) (string, error) {
	bridge, ok := m.bridges[libName]
	if !ok {
		return "", fmt.Errorf("动态库 '%s' 未注册", libName)
	}

	libPath := m.resolveLibPath(bridge)

	// 检查文件是否存在
	if _, err := os.Stat(libPath); os.IsNotExist(err) {
		return "", fmt.Errorf("动态库文件不存在: %s", libPath)
	}

	// 校验函数是否在导出列表中
	if !m.isFunctionExported(bridge, funcName) {
		return "", fmt.Errorf("函数 '%s' 不在动态库 '%s' 的导出列表中", funcName, libName)
	}

	logger.Debug("FFI 调用", "lib", libPath, "func", funcName, "params", params)

	// 通过 FFI 调用
	return callNativeFunction(libPath, funcName, params)
}

// resolveLibPath 根据当前平台解析动态库路径
func (m *Manager) resolveLibPath(bridge config.BridgeConfig) string {
	// 优先使用平台特定路径
	if platformPath, ok := bridge.Platform[runtime.GOOS]; ok {
		return platformPath
	}

	// 使用默认路径，添加平台特定的扩展名
	path := bridge.Path
	ext := getLibExtension()

	if !strings.HasSuffix(path, ext) {
		// 尝试添加扩展名
		dir := filepath.Dir(path)
		base := filepath.Base(path)
		base = strings.TrimSuffix(base, filepath.Ext(base))

		switch runtime.GOOS {
		case "windows":
			path = filepath.Join(dir, base+ext)
		case "darwin":
			path = filepath.Join(dir, "lib"+base+ext)
		default: // linux
			path = filepath.Join(dir, "lib"+base+ext)
		}
	}

	return path
}

// isFunctionExported 检查函数是否在导出列表中
func (m *Manager) isFunctionExported(bridge config.BridgeConfig, funcName string) bool {
	if len(bridge.Functions) == 0 {
		return true // 没有限制
	}
	for _, f := range bridge.Functions {
		if f == funcName {
			return true
		}
	}
	return false
}

// getLibExtension 获取当前平台的动态库扩展名
func getLibExtension() string {
	switch runtime.GOOS {
	case "windows":
		return ".dll"
	case "darwin":
		return ".dylib"
	default:
		return ".so"
	}
}
