//go:build windows

package shell

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

var (
	kernel32             = syscall.NewLazyDLL("kernel32.dll")
	ntdll                = syscall.NewLazyDLL("ntdll.dll")
	procGetConsoleMode   = kernel32.NewProc("GetConsoleMode")
	procSetConsoleMode   = kernel32.NewProc("SetConsoleMode")
	procSetConsoleTitleW = kernel32.NewProc("SetConsoleTitleW")
)

const (
	enableVirtualTerminalProcessing = 0x0004
	enableProcessedOutput           = 0x0001
)

// enableVirtualTerminal 在 Windows 上启用 ANSI 终端色彩
func enableVirtualTerminal() {
	handle := syscall.Handle(os.Stdout.Fd())

	var mode uint32
	r, _, _ := procGetConsoleMode.Call(uintptr(handle), uintptr(unsafe.Pointer(&mode)))
	if r == 0 {
		return
	}

	mode |= enableVirtualTerminalProcessing | enableProcessedOutput
	procSetConsoleMode.Call(uintptr(handle), uintptr(mode))

	// 设置控制台窗口标题
	title, _ := syscall.UTF16PtrFromString("⚡ Jiasine CLI")
	procSetConsoleTitleW.Call(uintptr(unsafe.Pointer(title)))
}

// IsDoubleClicked 检测是否通过双击 .exe 启动（Windows 特有）
//
// 检测策略（三层保障）：
//  1. GetConsoleProcessList — 双击时仅有 ≤2 个进程 (conhost + app)
//  2. 父进程名 — 双击时父进程为 explorer.exe
//  3. 环境变量 — 从 cmd/PowerShell 启动时有 PROMPT 或 PSModulePath
func IsDoubleClicked() bool {
	// 策略1：检查控制台进程数量
	// 双击启动：conhost.exe + jiasinecli.exe = 1~2
	// 从 cmd 启动：conhost.exe + cmd.exe + jiasinecli.exe = 3+
	// 从 PowerShell 启动：conhost.exe + powershell.exe + jiasinecli.exe = 3+
	proc := kernel32.NewProc("GetConsoleProcessList")
	pids := make([]uint32, 64)
	count, _, err := proc.Call(
		uintptr(unsafe.Pointer(&pids[0])),
		uintptr(len(pids)),
	)
	if err != nil && err.Error() != "The operation completed successfully." {
		// API 调用失败，退回到策略2
		return isParentExplorer()
	}

	if count <= 2 {
		return true // 新建控制台，很可能是双击启动
	}

	return false
}

// isParentExplorer 检测父进程是否为 explorer.exe
func isParentExplorer() bool {
	return getParentProcessName() == "explorer.exe"
}

// getParentProcessName 获取父进程的可执行文件名
func getParentProcessName() string {
	const processBasicInformation = 0

	type PROCESS_BASIC_INFORMATION struct {
		Reserved1       uintptr
		PebBaseAddress  uintptr
		Reserved2       [2]uintptr
		UniqueProcessId uintptr
		ParentProcessId uintptr
	}

	// 获取当前进程的父进程 ID
	ntQueryInformationProcess := ntdll.NewProc("NtQueryInformationProcess")
	handle := uintptr(0xFFFFFFFFFFFFFFFF) // current process pseudo handle

	var pbi PROCESS_BASIC_INFORMATION
	var returnLength uint32

	r, _, _ := ntQueryInformationProcess.Call(
		handle,
		processBasicInformation,
		uintptr(unsafe.Pointer(&pbi)),
		uintptr(unsafe.Sizeof(pbi)),
		uintptr(unsafe.Pointer(&returnLength)),
	)
	if r != 0 {
		return ""
	}

	parentPID := uint32(pbi.ParentProcessId)

	// 打开父进程获取名称
	const PROCESS_QUERY_LIMITED_INFORMATION = 0x1000
	procOpenProcess := kernel32.NewProc("OpenProcess")
	parentHandle, _, _ := procOpenProcess.Call(
		PROCESS_QUERY_LIMITED_INFORMATION,
		0,
		uintptr(parentPID),
	)
	if parentHandle == 0 {
		return ""
	}
	defer syscall.CloseHandle(syscall.Handle(parentHandle))

	// 获取进程可执行路径
	procQueryFullProcessImageNameW := kernel32.NewProc("QueryFullProcessImageNameW")
	var buf [260]uint16
	size := uint32(len(buf))
	r, _, _ = procQueryFullProcessImageNameW.Call(
		parentHandle,
		0,
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&size)),
	)
	if r == 0 {
		return ""
	}

	fullPath := syscall.UTF16ToString(buf[:size])
	return strings.ToLower(filepath.Base(fullPath))
}

// SetConsoleTitle 设置控制台窗口标题
func SetConsoleTitle(title string) {
	t, err := syscall.UTF16PtrFromString(title)
	if err != nil {
		return
	}
	procSetConsoleTitleW.Call(uintptr(unsafe.Pointer(t)))
}

// RelaunchInCmd 在 cmd.exe 中重新启动自身
// 双击 .exe 时调用，确保程序运行在完整的 cmd 终端环境中
func RelaunchInCmd() {
	exe, err := os.Executable()
	if err != nil {
		fmt.Println("无法获取程序路径:", err)
		return
	}
	c := exec.Command("cmd.exe", "/c", exe)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Run()
}

// KeepWindowOpen 保持控制台窗口不关闭（在异常退出时使用）
func KeepWindowOpen() {
	if IsDoubleClicked() {
		fmt.Println("\n按 Enter 键关闭窗口...")
		fmt.Scanln()
	}
}
