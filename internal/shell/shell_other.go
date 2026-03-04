//go:build !windows

package shell

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

// KeepWindowOpen 非 Windows 平台无操作
func KeepWindowOpen() {}
