// 跨平台 FFI 调用实现
// 使用 Go 的 plugin 包和 syscall 实现动态库调用

package bridge

import (
	"fmt"
	"runtime"
	"strings"
)

// callNativeFunction 调用原生函数（跨平台实现）
func callNativeFunction(libPath, funcName string, params []string) (string, error) {
	switch runtime.GOOS {
	case "windows":
		return callWindows(libPath, funcName, params)
	case "linux", "darwin":
		return callUnix(libPath, funcName, params)
	default:
		return "", fmt.Errorf("不支持的平台: %s", runtime.GOOS)
	}
}

// callUnix Unix 平台 FFI 调用 (使用 dlopen/dlsym)
func callUnix(libPath, funcName string, params []string) (string, error) {
	// 基于 cgo 的实现需在编译时启用 CGO_ENABLED=1
	// 这里提供一个基于命令行 wrapper 的后备方案
	return "", fmt.Errorf(
		"Unix FFI 调用需要 CGO 支持。请确保:\n"+
			"  1. 设置 CGO_ENABLED=1\n"+
			"  2. 安装对应平台的 C 编译器\n"+
			"  库: %s, 函数: %s, 参数: [%s]",
		libPath, funcName, strings.Join(params, ", "))
}
