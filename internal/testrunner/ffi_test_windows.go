// Windows DLL 测试调用

//go:build windows

package testrunner

import (
	"fmt"
	"strings"
	"syscall"
	"unsafe"
)

func testCallDLLPlatform(r *Runner, dllPath string) {
	dll, err := syscall.LoadDLL(dllPath)
	if err != nil {
		r.addResult("FFI", "加载DLL", false, "", fmt.Sprintf("加载失败: %v", err))
		return
	}
	defer dll.Release()
	r.addResult("FFI", "加载DLL", true, dllPath, "")

	// 测试 health()
	if proc, err := dll.FindProc("health"); err == nil {
		ret, _, _ := proc.Call()
		if ret != 0 {
			result := readCStr(ret)
			if strings.Contains(result, "ok") {
				r.addResult("FFI", "DLL/health()", true, result, "")
			} else {
				r.addResult("FFI", "DLL/health()", false, result, "返回值异常")
			}
		}
	} else {
		r.addResult("FFI", "DLL/health()", false, "", "函数未找到")
	}

	// 测试 get_version()
	if proc, err := dll.FindProc("get_version"); err == nil {
		ret, _, _ := proc.Call()
		if ret != 0 {
			result := readCStr(ret)
			if strings.Contains(result, "version") {
				r.addResult("FFI", "DLL/get_version()", true, result, "")
			} else {
				r.addResult("FFI", "DLL/get_version()", false, result, "返回值异常")
			}
		}
	} else {
		r.addResult("FFI", "DLL/get_version()", false, "", "函数未找到")
	}

	// 测试 add("10 20")
	if proc, err := dll.FindProc("add"); err == nil {
		input, _ := syscall.BytePtrFromString("10 20")
		ret, _, _ := proc.Call(uintptr(unsafe.Pointer(input)))
		if ret != 0 {
			result := readCStr(ret)
			if strings.Contains(result, "30") {
				r.addResult("FFI", "DLL/add(10+20=30)", true, result, "")
			} else {
				r.addResult("FFI", "DLL/add()", false, result, "结果不正确")
			}
		}
	} else {
		r.addResult("FFI", "DLL/add()", false, "", "函数未找到")
	}

	// 测试 reverse_string("hello")
	if proc, err := dll.FindProc("reverse_string"); err == nil {
		input, _ := syscall.BytePtrFromString("hello")
		ret, _, _ := proc.Call(uintptr(unsafe.Pointer(input)))
		if ret != 0 {
			result := readCStr(ret)
			if strings.Contains(result, "olleh") {
				r.addResult("FFI", "DLL/reverse_string(hello→olleh)", true, result, "")
			} else {
				r.addResult("FFI", "DLL/reverse_string()", false, result, "结果不正确")
			}
		}
	} else {
		r.addResult("FFI", "DLL/reverse_string()", false, "", "函数未找到")
	}
}

// readCStr 从 C 字符串指针读取
func readCStr(ptr uintptr) string {
	if ptr == 0 {
		return ""
	}
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

// testCallDLLWithLangPlatform 带语言标签的 DLL 调用 (Windows)
func testCallDLLWithLangPlatform(r *Runner, dllPath string, lang string) {
	dll, err := syscall.LoadDLL(dllPath)
	if err != nil {
		r.addResult(lang, "加载DLL", false, "", fmt.Sprintf("加载失败: %v", err))
		return
	}
	defer dll.Release()
	r.addResult(lang, "加载DLL", true, dllPath, "")

	// 测试 health()
	if proc, err := dll.FindProc("health"); err == nil {
		ret, _, _ := proc.Call()
		if ret != 0 {
			result := readCStr(ret)
			if strings.Contains(result, "ok") {
				r.addResult(lang, "DLL/health()", true, result, "")
			} else {
				r.addResult(lang, "DLL/health()", false, result, "返回值异常")
			}
		}
	} else {
		r.addResult(lang, "DLL/health()", false, "", "函数未找到")
	}

	// 测试 get_version()
	if proc, err := dll.FindProc("get_version"); err == nil {
		ret, _, _ := proc.Call()
		if ret != 0 {
			result := readCStr(ret)
			if strings.Contains(result, "version") {
				r.addResult(lang, "DLL/get_version()", true, result, "")
			} else {
				r.addResult(lang, "DLL/get_version()", false, result, "返回值异常")
			}
		}
	} else {
		r.addResult(lang, "DLL/get_version()", false, "", "函数未找到")
	}

	// 测试 add("10 20")
	if proc, err := dll.FindProc("add"); err == nil {
		input, _ := syscall.BytePtrFromString("10 20")
		ret, _, _ := proc.Call(uintptr(unsafe.Pointer(input)))
		if ret != 0 {
			result := readCStr(ret)
			if strings.Contains(result, "30") {
				r.addResult(lang, "DLL/add(10+20=30)", true, result, "")
			} else {
				r.addResult(lang, "DLL/add()", false, result, "结果不正确")
			}
		}
	} else {
		r.addResult(lang, "DLL/add()", false, "", "函数未找到")
	}

	// 测试 reverse_string("hello")
	if proc, err := dll.FindProc("reverse_string"); err == nil {
		input, _ := syscall.BytePtrFromString("hello")
		ret, _, _ := proc.Call(uintptr(unsafe.Pointer(input)))
		if ret != 0 {
			result := readCStr(ret)
			if strings.Contains(result, "olleh") {
				r.addResult(lang, "DLL/reverse_string(hello→olleh)", true, result, "")
			} else {
				r.addResult(lang, "DLL/reverse_string()", false, result, "结果不正确")
			}
		}
	} else {
		r.addResult(lang, "DLL/reverse_string()", false, "", "函数未找到")
	}
}
