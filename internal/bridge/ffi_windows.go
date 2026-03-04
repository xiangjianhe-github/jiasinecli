// Windows 平台 FFI 调用实现
// 使用 syscall.LoadDLL / LoadLibrary 加载 DLL

//go:build windows

package bridge

import (
	"fmt"
	"strings"
	"syscall"
	"unsafe"
)

// callWindows Windows 平台 DLL 调用
func callWindows(libPath, funcName string, params []string) (string, error) {
	// 加载 DLL
	dll, err := syscall.LoadDLL(libPath)
	if err != nil {
		return "", fmt.Errorf("加载 DLL 失败 [%s]: %w", libPath, err)
	}
	defer dll.Release()

	// 查找函数
	proc, err := dll.FindProc(funcName)
	if err != nil {
		return "", fmt.Errorf("查找函数失败 [%s]: %w", funcName, err)
	}

	// 根据参数数量调用
	// 对于简单场景：将所有参数拼接为一个字符串传入
	if len(params) == 0 {
		ret, _, callErr := proc.Call()
		if callErr != nil && callErr.Error() != "The operation completed successfully." {
			return "", fmt.Errorf("调用失败: %w", callErr)
		}
		if ret != 0 {
			// 尝试将返回值解释为字符串指针
			result := readCString(ret)
			if result != "" {
				return result, nil
			}
			return fmt.Sprintf("%d", ret), nil
		}
		return "OK", nil
	}

	// 将参数转为 C 字符串
	input := strings.Join(params, " ")
	inputPtr, err := syscall.BytePtrFromString(input)
	if err != nil {
		return "", fmt.Errorf("参数编码失败: %w", err)
	}

	ret, _, callErr := proc.Call(uintptr(unsafe.Pointer(inputPtr)))
	if callErr != nil && callErr.Error() != "The operation completed successfully." {
		return "", fmt.Errorf("调用失败: %w", callErr)
	}

	if ret != 0 {
		result := readCString(ret)
		if result != "" {
			return result, nil
		}
		return fmt.Sprintf("%d", ret), nil
	}

	return "OK", nil
}

// readCString 从指针读取 C 字符串
func readCString(ptr uintptr) string {
	if ptr == 0 {
		return ""
	}

	// 安全地读取字符串，最大 4KB
	var buf []byte
	for i := 0; i < 4096; i++ {
		b := *(*byte)(unsafe.Pointer(ptr + uintptr(i)))
		if b == 0 {
			break
		}
		buf = append(buf, b)
	}
	return string(buf)
}
