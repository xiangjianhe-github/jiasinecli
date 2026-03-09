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
	kernel32                       = syscall.NewLazyDLL("kernel32.dll")
	ntdll                          = syscall.NewLazyDLL("ntdll.dll")
	user32                         = syscall.NewLazyDLL("user32.dll")
	procGetConsoleMode             = kernel32.NewProc("GetConsoleMode")
	procSetConsoleMode             = kernel32.NewProc("SetConsoleMode")
	procSetConsoleTitleW           = kernel32.NewProc("SetConsoleTitleW")
	procGetConsoleWindow           = kernel32.NewProc("GetConsoleWindow")
	procShowWindow                 = user32.NewProc("ShowWindow")
	procFreeConsole                = kernel32.NewProc("FreeConsole")

	// doubleClicked 标记是否通过双击 .exe 启动（在 init 中判定）
	doubleClicked bool
)

// init 在 main() 之前执行，尽早隐藏并释放控制台窗口，最大限度减少闪屏。
// 双击启动时需要同时满足两个条件：
//   1. GetConsoleProcessList 返回 ≤2（conhost + 本进程）
//   2. 父进程为 explorer.exe（排除 ConPTY/VS Code 等伪终端的误判）
func init() {
	// 先检查父进程 — 快速排除非 Explorer 启动的情况
	parent := getParentProcessName()
	if parent != "explorer.exe" {
		return // 从终端启动，不做任何处理
	}

	// 再检查控制台进程数量（双重确认）
	proc := kernel32.NewProc("GetConsoleProcessList")
	pids := make([]uint32, 64)
	count, _, err := proc.Call(
		uintptr(unsafe.Pointer(&pids[0])),
		uintptr(len(pids)),
	)

	isDoubleClick := false
	if err != nil && err.Error() != "The operation completed successfully." {
		isDoubleClick = true // API 失败但父进程是 Explorer，视为双击
	} else if count <= 2 {
		isDoubleClick = true
	}

	if isDoubleClick {
		doubleClicked = true
		// 立即隐藏控制台窗口（最小化闪屏时间）
		hwnd, _, _ := procGetConsoleWindow.Call()
		if hwnd != 0 {
			procShowWindow.Call(hwnd, 0) // SW_HIDE
		}
		// 释放控制台 — conhost 窗口关闭
		procFreeConsole.Call()
	}
}

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
// 在 init() 中已经判定并缓存到 doubleClicked 变量
func IsDoubleClicked() bool {
	return doubleClicked
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

// isAdmin 检测当前进程是否以管理员权限运行
func isAdmin() bool {
	_, err := os.Open("\\\\.\\PHYSICALDRIVE0")
	return err == nil
}

// isWindowsTerminal 检测是否运行在 Windows Terminal 中
func isWindowsTerminal() bool {
	return os.Getenv("WT_SESSION") != "" || os.Getenv("WT_PROFILE_ID") != ""
}

// RelaunchInWindowsTerminalAdmin 在 Windows Terminal 中以管理员权限重新启动自身
// 返回值: true = 需要退出当前进程（已重新启动）, false = 继续运行当前进程
func RelaunchInWindowsTerminalAdmin() bool {
	// 如果已在管理员模式，无需重启，继续运行
	if isAdmin() {
		return false
	}

	// 如果不是双击启动（从终端启动），也不强制重启，继续运行
	if !IsDoubleClicked() {
		return false
	}

	exe, err := os.Executable()
	if err != nil {
		fmt.Printf("\n⚠ 无法获取程序路径: %v\n", err)
		fmt.Println("将在当前环境继续运行（非管理员权限）")
		return false
	}

	// 优先使用 PowerShell（更稳定）
	fmt.Println("正在请求管理员权限...")
	success := launchInPowerShellAdmin(exe)
	if !success {
		fmt.Println("\n⚠ 未获得管理员权限，将在当前环境继续运行")
		fmt.Println("某些功能可能受限，建议右键选择【以管理员身份运行】")
		fmt.Println()
		return false
	}

	// 成功启动新进程，等待一下确保新窗口打开
	return true
}

// launchInPowerShellAdmin 在 PowerShell 中以管理员权限启动
// 返回值: true = 启动成功, false = 用户取消或失败
func launchInPowerShellAdmin(exePath string) bool {
	shell32 := syscall.NewLazyDLL("shell32.dll")
	procShellExecute := shell32.NewProc("ShellExecuteW")

	verb, _ := syscall.UTF16PtrFromString("runas")
	file, _ := syscall.UTF16PtrFromString("powershell.exe")
	params, _ := syscall.UTF16PtrFromString(fmt.Sprintf("-NoExit -Command \"& '%s'\"", exePath))
	showCmd := int32(1) // SW_SHOWNORMAL

	ret, _, _ := procShellExecute.Call(
		0,
		uintptr(unsafe.Pointer(verb)),
		uintptr(unsafe.Pointer(file)),
		uintptr(unsafe.Pointer(params)),
		0,
		uintptr(showCmd),
	)

	// ShellExecute 返回值说明：
	// > 32: 成功
	// 5: ERROR_ACCESS_DENIED (用户取消 UAC)
	// 其他 <= 32: 其他错误
	return ret > 32
}

// RelaunchInCmd 在 cmd.exe 中重新启动自身
// 保留用于兼容性
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

// RelaunchInPowerShell 双击启动时以管理员权限在 PowerShell 中重新启动自身
// init() 已在极早期隐藏并释放了控制台窗口（无闪屏），
// 此函数只需通过 ShellExecuteW("runas") 启动 PowerShell。
// 返回 true 表示需要退出当前进程
func RelaunchInPowerShell() bool {
	if !doubleClicked {
		return false
	}

	exe, err := os.Executable()
	if err != nil {
		return false
	}

	// 使用 ShellExecuteW + "runas" 以管理员权限启动 powershell.exe 运行本程序
	shell32 := syscall.NewLazyDLL("shell32.dll")
	procShellExecute := shell32.NewProc("ShellExecuteW")

	verb, _ := syscall.UTF16PtrFromString("runas")
	file, _ := syscall.UTF16PtrFromString("powershell.exe")
	// -NoExit 保持窗口不关闭; -Command 执行 exe
	params, _ := syscall.UTF16PtrFromString(fmt.Sprintf("-NoExit -Command \"& '%s'\"", exe))
	showCmd := int32(1) // SW_SHOWNORMAL

	ret, _, _ := procShellExecute.Call(
		0,
		uintptr(unsafe.Pointer(verb)),
		uintptr(unsafe.Pointer(file)),
		uintptr(unsafe.Pointer(params)),
		0,
		uintptr(showCmd),
	)

	if ret > 32 {
		return true // 成功启动新进程，当前进程应退出
	}

	// 用户取消 UAC 或失败 — 分配新控制台继续运行
	procAllocConsole := kernel32.NewProc("AllocConsole")
	procAllocConsole.Call()
	return false
}

// KeepWindowOpen 保持控制台窗口不关闭（在异常退出时使用）
func KeepWindowOpen() {
	if IsDoubleClicked() {
		fmt.Println("\n按 Enter 键关闭窗口...")
		fmt.Scanln()
	}
}

// DetectSystemTheme 检测 Windows 系统主题（暗色/亮色）
// 通过读取注册表 Personalize\AppsUseLightTheme 判断
func DetectSystemTheme() string {
	advapi32 := syscall.NewLazyDLL("advapi32.dll")
	regOpenKeyExW := advapi32.NewProc("RegOpenKeyExW")
	regQueryValueExW := advapi32.NewProc("RegQueryValueExW")
	regCloseKey := advapi32.NewProc("RegCloseKey")

	const hkeyCurrentUser = 0x80000001
	const keyRead = 0x20019

	keyPath, _ := syscall.UTF16PtrFromString(`SOFTWARE\Microsoft\Windows\CurrentVersion\Themes\Personalize`)

	var hKey uintptr
	ret, _, _ := regOpenKeyExW.Call(
		hkeyCurrentUser,
		uintptr(unsafe.Pointer(keyPath)),
		0,
		keyRead,
		uintptr(unsafe.Pointer(&hKey)),
	)
	if ret != 0 {
		return "dark" // 默认暗色
	}
	defer regCloseKey.Call(hKey)

	valueName, _ := syscall.UTF16PtrFromString("AppsUseLightTheme")
	var valType uint32
	var val uint32
	var valLen uint32 = 4

	ret, _, _ = regQueryValueExW.Call(
		hKey,
		uintptr(unsafe.Pointer(valueName)),
		0,
		uintptr(unsafe.Pointer(&valType)),
		uintptr(unsafe.Pointer(&val)),
		uintptr(unsafe.Pointer(&valLen)),
	)
	if ret != 0 {
		return "dark"
	}
	if val == 1 {
		return "light"
	}
	return "dark"
}

// InstallToPath 将当前 exe 复制到 %USERPROFILE%\.jiasine\bin\ 并将该目录添加到用户 PATH
// 使 PowerShell 中可以直接输入 jiasinecli 运行
func InstallToPath() (installDir string, err error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("获取程序路径失败: %w", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("获取用户目录失败: %w", err)
	}

	installDir = filepath.Join(home, ".jiasine", "bin")
	if err := os.MkdirAll(installDir, 0755); err != nil {
		return "", fmt.Errorf("创建安装目录失败: %w", err)
	}

	destPath := filepath.Join(installDir, "jiasinecli.exe")

	// 复制文件（如果不是同一路径）
	srcAbs, _ := filepath.Abs(exe)
	dstAbs, _ := filepath.Abs(destPath)
	if !strings.EqualFold(srcAbs, dstAbs) {
		srcData, err := os.ReadFile(exe)
		if err != nil {
			return "", fmt.Errorf("读取程序文件失败: %w", err)
		}
		if err := os.WriteFile(destPath, srcData, 0755); err != nil {
			return "", fmt.Errorf("复制程序文件失败: %w", err)
		}
	}

	// 将安装目录添加到用户 PATH（通过注册表，永久生效）
	if err := addToUserPath(installDir); err != nil {
		return installDir, fmt.Errorf("添加到 PATH 失败: %w", err)
	}

	return installDir, nil
}

// addToUserPath 将目录添加到用户级 PATH 环境变量（通过注册表）
func addToUserPath(dir string) error {
	advapi32 := syscall.NewLazyDLL("advapi32.dll")
	regOpenKeyExW := advapi32.NewProc("RegOpenKeyExW")
	regQueryValueExW := advapi32.NewProc("RegQueryValueExW")
	regSetValueExW := advapi32.NewProc("RegSetValueExW")
	regCloseKey := advapi32.NewProc("RegCloseKey")

	const hkeyCurrentUser = 0x80000001
	const keyAllAccess = 0xF003F
	const regExpandSz = 2

	keyPath, _ := syscall.UTF16PtrFromString(`Environment`)
	var hKey uintptr
	ret, _, _ := regOpenKeyExW.Call(
		hkeyCurrentUser,
		uintptr(unsafe.Pointer(keyPath)),
		0,
		keyAllAccess,
		uintptr(unsafe.Pointer(&hKey)),
	)
	if ret != 0 {
		return fmt.Errorf("打开注册表键失败: %d", ret)
	}
	defer regCloseKey.Call(hKey)

	// 读取当前 PATH
	valueName, _ := syscall.UTF16PtrFromString("Path")
	var valType uint32
	var bufLen uint32 = 32768
	buf := make([]uint16, bufLen/2)

	ret, _, _ = regQueryValueExW.Call(
		hKey,
		uintptr(unsafe.Pointer(valueName)),
		0,
		uintptr(unsafe.Pointer(&valType)),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&bufLen)),
	)

	currentPath := ""
	if ret == 0 {
		currentPath = syscall.UTF16ToString(buf)
	}

	// 检查是否已经包含目标目录
	dirLower := strings.ToLower(dir)
	for _, p := range strings.Split(currentPath, ";") {
		if strings.ToLower(strings.TrimSpace(p)) == dirLower {
			return nil // 已存在，无需修改
		}
	}

	// 追加目录
	newPath := currentPath
	if newPath != "" && !strings.HasSuffix(newPath, ";") {
		newPath += ";"
	}
	newPath += dir

	// 写入注册表
	newPathUTF16, _ := syscall.UTF16FromString(newPath)
	newPathBytes := (*[1 << 20]byte)(unsafe.Pointer(&newPathUTF16[0]))[:len(newPathUTF16)*2]

	ret, _, _ = regSetValueExW.Call(
		hKey,
		uintptr(unsafe.Pointer(valueName)),
		0,
		regExpandSz,
		uintptr(unsafe.Pointer(&newPathBytes[0])),
		uintptr(len(newPathBytes)),
	)
	if ret != 0 {
		return fmt.Errorf("写入注册表失败: %d", ret)
	}

	// 广播 WM_SETTINGCHANGE 通知其他进程 PATH 已变更
	broadcastSettingChange()

	return nil
}

// broadcastSettingChange 广播环境变量变更通知
func broadcastSettingChange() {
	procSendMessageTimeout := user32.NewProc("SendMessageTimeoutW")
	const HWND_BROADCAST = 0xFFFF
	const WM_SETTINGCHANGE = 0x001A
	const SMTO_ABORTIFHUNG = 0x0002

	env, _ := syscall.UTF16PtrFromString("Environment")
	procSendMessageTimeout.Call(
		HWND_BROADCAST,
		WM_SETTINGCHANGE,
		0,
		uintptr(unsafe.Pointer(env)),
		SMTO_ABORTIFHUNG,
		5000,
		0,
	)
}
