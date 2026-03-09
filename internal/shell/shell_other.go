//go:build !windows

package shell

import "fmt"

// enableVirtualTerminal 非 Windows 平台无需特殊处理
func enableVirtualTerminal() {}

// IsDoubleClicked 非 Windows 平台返回 false
func IsDoubleClicked() bool {
	return false
}

// SetConsoleTitle 非 Windows 平台无操作
func SetConsoleTitle(title string) {}

// RelaunchInCmd 非 Windows 平台无操作
func RelaunchInCmd() {}

// RelaunchInPowerShell 非 Windows 平台无操作
func RelaunchInPowerShell() bool {
	return false
}

// RelaunchInWindowsTerminalAdmin 非 Windows 平台无操作
func RelaunchInWindowsTerminalAdmin() bool {
	return false
}

// KeepWindowOpen 非 Windows 平台无操作
func KeepWindowOpen() {}

// DetectSystemTheme 非 Windows 平台默认返回暗色
func DetectSystemTheme() string {
	return "dark"
}

// InstallToPath 非 Windows 平台：创建符号链接到 ~/.local/bin/
func InstallToPath() (string, error) {
	return "", fmt.Errorf("当前平台请手动将程序添加到 PATH")
}
