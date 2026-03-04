// 非 Windows 平台 FFI 调用 stub

//go:build !windows

package bridge

// callWindows 非 Windows 平台不可用
func callWindows(libPath, funcName string, params []string) (string, error) {
	return callUnix(libPath, funcName, params)
}
