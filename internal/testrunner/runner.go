// Package testrunner 提供多语言集成测试运行器
package testrunner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Runner 测试运行器
type Runner struct {
	testsDir string
	client   *http.Client
	results  []TestResult
}

// TestResult 测试结果
type TestResult struct {
	Language string
	TestName string
	Passed   bool
	Output   string
	Error    string
}

// EnvStatus 环境状态
type EnvStatus struct {
	Language string
	Ready    bool
	Status   string
	Detail   string
}

// New 创建测试运行器
func New(testsDir string) *Runner {
	return &Runner{
		testsDir: testsDir,
		client:   &http.Client{Timeout: 10 * time.Second},
	}
}

// RunAll 运行所有语言测试
func (r *Runner) RunAll() error {
	fmt.Println("╔═══════════════════════════════════════════════════╗")
	fmt.Println("║     Jiasine CLI - 多语言集成测试                  ║")
	fmt.Println("╚═══════════════════════════════════════════════════╝")
	fmt.Println()

	languages := []string{"c", "python", "rust", "csharp", "js", "typescript", "java", "swift", "objc"}
	for _, lang := range languages {
		r.RunLanguage(lang)
		fmt.Println()
	}

	// 汇总
	r.printSummary()
	return nil
}

// RunLanguage 运行指定语言的测试
func (r *Runner) RunLanguage(lang string) error {
	switch lang {
	case "c":
		return r.testC()
	case "python":
		return r.testPython()
	case "rust":
		return r.testRust()
	case "csharp", "cs":
		return r.testCSharp()
	case "js", "javascript":
		return r.testJS()
	case "ts", "typescript":
		return r.testTypeScript()
	case "java":
		return r.testJava()
	case "swift":
		return r.testSwift()
	case "objc", "objective-c":
		return r.testObjC()
	default:
		return fmt.Errorf("不支持的语言: %s (可选: c/python/rust/csharp/js/typescript/java/swift/objc)", lang)
	}
}

// CheckEnvironment 检查各语言测试环境
func (r *Runner) CheckEnvironment() []EnvStatus {
	var status []EnvStatus

	// C 编译器
	if checkCommand("gcc", "--version") {
		status = append(status, EnvStatus{"C/C++", true, "就绪", "gcc 已安装"})
	} else if checkCommand("cl", "") {
		status = append(status, EnvStatus{"C/C++", true, "就绪", "MSVC cl 已安装"})
	} else {
		status = append(status, EnvStatus{"C/C++", false, "未安装", "需要 gcc 或 MSVC"})
	}

	// Python
	if checkCommand("python", "--version") {
		status = append(status, EnvStatus{"Python", true, "就绪", "python 已安装"})
	} else if checkCommand("python3", "--version") {
		status = append(status, EnvStatus{"Python", true, "就绪", "python3 已安装"})
	} else {
		status = append(status, EnvStatus{"Python", false, "未安装", "需要 Python 3.x"})
	}

	// Rust
	if checkCommand("rustc", "--version") {
		status = append(status, EnvStatus{"Rust", true, "就绪", "rustc 已安装"})
	} else {
		status = append(status, EnvStatus{"Rust", false, "未安装", "需要 rustup/rustc"})
	}

	// C# / .NET
	if checkCommand("dotnet", "--version") {
		status = append(status, EnvStatus{"C#", true, "就绪", "dotnet 已安装"})
	} else {
		status = append(status, EnvStatus{"C#", false, "未安装", "需要 .NET SDK 8.0+"})
	}

	// JavaScript (Node.js)
	if checkCommand("node", "--version") {
		status = append(status, EnvStatus{"JavaScript", true, "就绪", "node 已安装"})
	} else {
		status = append(status, EnvStatus{"JavaScript", false, "未安装", "需要 Node.js"})
	}

	// TypeScript (Node.js + npx tsx)
	if checkCommand("node", "--version") && checkCommand("npx", "--version") {
		status = append(status, EnvStatus{"TypeScript", true, "就绪", "node + npx 已安装"})
	} else {
		status = append(status, EnvStatus{"TypeScript", false, "未安装", "需要 Node.js + npx"})
	}

	// Java
	if checkCommand("javac", "-version") {
		status = append(status, EnvStatus{"Java", true, "就绪", "javac 已安装"})
	} else {
		status = append(status, EnvStatus{"Java", false, "未安装", "需要 JDK (javac)"})
	}

	// Swift
	if checkCommand("swiftc", "--version") {
		status = append(status, EnvStatus{"Swift", true, "就绪", "swiftc 已安装"})
	} else {
		status = append(status, EnvStatus{"Swift", false, "未安装", "需要 Swift 工具链"})
	}

	// Objective-C (使用 gcc/clang)
	if checkCommand("gcc", "--version") || checkCommand("clang", "--version") {
		status = append(status, EnvStatus{"Objective-C", true, "就绪", "gcc/clang 可编译 .m"})
	} else {
		status = append(status, EnvStatus{"Objective-C", false, "未安装", "需要 gcc 或 clang"})
	}

	return status
}

// ═══════════════ C 测试 ═══════════════

func (r *Runner) testC() error {
	r.printLangHeader("C/C++", "FFI 动态库测试")

	cDir := filepath.Join(r.testsDir, "c")
	srcFile := filepath.Join(cDir, "jiasine_c_test.c")

	if _, err := os.Stat(srcFile); os.IsNotExist(err) {
		r.addResult("C", "源文件检查", false, "", "jiasine_c_test.c 不存在")
		return nil
	}

	// 编译
	var dllPath string
	var compileCmd *exec.Cmd
	if runtime.GOOS == "windows" {
		dllPath = filepath.Join(cDir, "jiasine_c_test.dll")
		if checkCommand("gcc", "--version") {
			compileCmd = exec.Command("gcc", "-shared", "-o", dllPath, srcFile)
		} else {
			r.addResult("C", "编译", false, "", "未找到 gcc 编译器")
			return nil
		}
	} else if runtime.GOOS == "darwin" {
		dllPath = filepath.Join(cDir, "libjiasine_c_test.dylib")
		compileCmd = exec.Command("gcc", "-shared", "-fPIC", "-o", dllPath, srcFile)
	} else {
		dllPath = filepath.Join(cDir, "libjiasine_c_test.so")
		compileCmd = exec.Command("gcc", "-shared", "-fPIC", "-o", dllPath, srcFile)
	}

	compileCmd.Dir = cDir
	output, err := compileCmd.CombinedOutput()
	if err != nil {
		r.addResult("C", "编译动态库", false, string(output), err.Error())
		return nil
	}
	r.addResult("C", "编译动态库", true, fmt.Sprintf("成功: %s", filepath.Base(dllPath)), "")

	// 通过 bridge 调用测试
	// 直接用 FFI 调用测试
	r.testCFFI(dllPath)

	return nil
}

func (r *Runner) testCFFI(dllPath string) {
	if runtime.GOOS != "windows" {
		r.addResult("C", "FFI 调用", false, "", "当前仅支持 Windows DLL 直接测试，其他平台请使用 CGO")
		return
	}

	// 使用 Go 测试程序验证 DLL
	// 由于我们在主程序中已有 bridge 模块，直接调用
	testCallDLL(r, dllPath)
}

// ═══════════════ Python 测试 ═══════════════

func (r *Runner) testPython() error {
	r.printLangHeader("Python", "HTTP 服务 + 进程调用测试")

	pyDir := filepath.Join(r.testsDir, "python")
	python := findPython()
	if python == "" {
		r.addResult("Python", "环境检查", false, "", "未找到 Python")
		return nil
	}
	r.addResult("Python", "环境检查", true, fmt.Sprintf("使用 %s", python), "")

	// 测试 1: 进程调用
	r.testPythonProcess(python, pyDir)

	// 测试 2: HTTP 服务
	r.testPythonHTTP(python, pyDir)

	return nil
}

func (r *Runner) testPythonProcess(python, pyDir string) {
	scriptPath := filepath.Join(pyDir, "jiasine_py_process.py")

	// 测试 add
	cmd := exec.Command(python, scriptPath, "add", "42", "58")
	output, err := cmd.CombinedOutput()
	if err != nil {
		r.addResult("Python", "进程调用/add", false, string(output), err.Error())
		return
	}
	outStr := strings.TrimSpace(string(output))
	if strings.Contains(outStr, `"result": 100`) {
		r.addResult("Python", "进程调用/add(42+58=100)", true, outStr, "")
	} else {
		r.addResult("Python", "进程调用/add", false, outStr, "结果不正确")
	}

	// 测试 reverse
	cmd = exec.Command(python, scriptPath, "reverse", "Jiasine")
	output, err = cmd.CombinedOutput()
	if err != nil {
		r.addResult("Python", "进程调用/reverse", false, string(output), err.Error())
		return
	}
	outStr = strings.TrimSpace(string(output))
	if strings.Contains(outStr, `"reversed": "enisaiJ"`) {
		r.addResult("Python", "进程调用/reverse(Jiasine)", true, outStr, "")
	} else {
		r.addResult("Python", "进程调用/reverse", false, outStr, "结果不正确")
	}

	// 测试 fibonacci
	cmd = exec.Command(python, scriptPath, "fibonacci", "8")
	output, err = cmd.CombinedOutput()
	if err != nil {
		r.addResult("Python", "进程调用/fibonacci", false, string(output), err.Error())
		return
	}
	outStr = strings.TrimSpace(string(output))
	if strings.Contains(outStr, "fibonacci") && strings.Contains(outStr, "0, 1, 1, 2, 3, 5, 8, 13") {
		r.addResult("Python", "进程调用/fibonacci(8)", true, outStr, "")
	} else {
		r.addResult("Python", "进程调用/fibonacci", false, outStr, "结果不正确")
	}
}

func (r *Runner) testPythonHTTP(python, pyDir string) {
	scriptPath := filepath.Join(pyDir, "jiasine_py_test.py")
	port := findFreePort()

	// 启动 HTTP 服务
	cmd := exec.Command(python, scriptPath, "--port", fmt.Sprintf("%d", port))
	cmd.Dir = pyDir
	if err := cmd.Start(); err != nil {
		r.addResult("Python", "HTTP服务启动", false, "", err.Error())
		return
	}
	defer func() {
		cmd.Process.Kill()
		cmd.Wait()
	}()

	// 等待服务就绪
	addr := fmt.Sprintf("http://127.0.0.1:%d", port)
	if !waitForService(addr+"/health", 5*time.Second) {
		r.addResult("Python", "HTTP服务启动", false, "", "服务启动超时")
		return
	}
	r.addResult("Python", "HTTP服务启动", true, fmt.Sprintf("端口 %d", port), "")

	// 测试 health
	resp, err := r.client.Get(addr + "/health")
	if err != nil {
		r.addResult("Python", "HTTP/health", false, "", err.Error())
		return
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode == 200 && strings.Contains(string(body), "ok") {
		r.addResult("Python", "HTTP/health", true, string(body), "")
	} else {
		r.addResult("Python", "HTTP/health", false, string(body), "状态码非200")
	}

	// 测试 add
	payload := `{"params": ["25", "75"]}`
	resp, err = r.client.Post(addr+"/add", "application/json", bytes.NewBufferString(payload))
	if err != nil {
		r.addResult("Python", "HTTP/add", false, "", err.Error())
		return
	}
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	if strings.Contains(string(body), `"result": 100`) {
		r.addResult("Python", "HTTP/add(25+75=100)", true, string(body), "")
	} else {
		r.addResult("Python", "HTTP/add", false, string(body), "结果不正确")
	}

	// 测试 fibonacci
	payload = `{"params": ["6"]}`
	resp, err = r.client.Post(addr+"/fibonacci", "application/json", bytes.NewBufferString(payload))
	if err != nil {
		r.addResult("Python", "HTTP/fibonacci", false, "", err.Error())
		return
	}
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	if strings.Contains(string(body), "fibonacci") {
		r.addResult("Python", "HTTP/fibonacci(6)", true, string(body), "")
	} else {
		r.addResult("Python", "HTTP/fibonacci", false, string(body), "结果不正确")
	}
}

// ═══════════════ Rust 测试 ═══════════════

func (r *Runner) testRust() error {
	r.printLangHeader("Rust", "FFI 动态库测试")

	rustDir := filepath.Join(r.testsDir, "rust")
	cargoToml := filepath.Join(rustDir, "Cargo.toml")

	if _, err := os.Stat(cargoToml); os.IsNotExist(err) {
		r.addResult("Rust", "项目检查", false, "", "Cargo.toml 不存在")
		return nil
	}

	if !checkCommand("cargo", "--version") {
		r.addResult("Rust", "环境检查", false, "", "未找到 cargo, 请安装 Rust 工具链")
		return nil
	}
	r.addResult("Rust", "环境检查", true, "cargo 已安装", "")

	// 编译
	cmd := exec.Command("cargo", "build", "--release")
	cmd.Dir = rustDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		r.addResult("Rust", "编译动态库", false, string(output), err.Error())
		return nil
	}
	r.addResult("Rust", "编译动态库", true, "cargo build --release 成功", "")

	// 找到编译产物
	var libPath string
	switch runtime.GOOS {
	case "windows":
		libPath = filepath.Join(rustDir, "target", "release", "jiasine_rust_test.dll")
	case "darwin":
		libPath = filepath.Join(rustDir, "target", "release", "libjiasine_rust_test.dylib")
	default:
		libPath = filepath.Join(rustDir, "target", "release", "libjiasine_rust_test.so")
	}

	if _, err := os.Stat(libPath); os.IsNotExist(err) {
		r.addResult("Rust", "产物检查", false, "", fmt.Sprintf("未找到: %s", libPath))
		return nil
	}
	r.addResult("Rust", "产物检查", true, filepath.Base(libPath), "")

	// FFI 调用测试
	if runtime.GOOS == "windows" {
		testCallDLL(r, libPath)
	} else {
		r.addResult("Rust", "FFI 调用", false, "", "非 Windows 平台需要 CGO 支持")
	}

	return nil
}

// ═══════════════ C# 测试 ═══════════════

func (r *Runner) testCSharp() error {
	r.printLangHeader("C#", "HTTP 服务测试")

	csDir := filepath.Join(r.testsDir, "csharp")
	csproj := filepath.Join(csDir, "JiasineCsharpTest.csproj")

	if _, err := os.Stat(csproj); os.IsNotExist(err) {
		r.addResult("C#", "项目检查", false, "", "JiasineCsharpTest.csproj 不存在")
		return nil
	}

	if !checkCommand("dotnet", "--version") {
		r.addResult("C#", "环境检查", false, "", "未找到 dotnet SDK")
		return nil
	}
	r.addResult("C#", "环境检查", true, "dotnet SDK 已安装", "")

	// 编译
	cmd := exec.Command("dotnet", "build", "-c", "Release", "--nologo", "-v", "q")
	cmd.Dir = csDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		r.addResult("C#", "编译项目", false, string(output), err.Error())
		return nil
	}
	r.addResult("C#", "编译项目", true, "dotnet build 成功", "")

	// 启动服务
	port := findFreePort()
	svcCmd := exec.Command("dotnet", "run", "--no-build", "-c", "Release",
		"--urls", fmt.Sprintf("http://127.0.0.1:%d", port))
	svcCmd.Dir = csDir
	if err := svcCmd.Start(); err != nil {
		r.addResult("C#", "服务启动", false, "", err.Error())
		return nil
	}
	defer func() {
		svcCmd.Process.Kill()
		svcCmd.Wait()
	}()

	addr := fmt.Sprintf("http://127.0.0.1:%d", port)
	if !waitForService(addr+"/health", 15*time.Second) {
		r.addResult("C#", "服务启动", false, "", "服务启动超时 (15s)")
		return nil
	}
	r.addResult("C#", "服务启动", true, fmt.Sprintf("端口 %d", port), "")

	// 测试 health
	resp, err := r.client.Get(addr + "/health")
	if err != nil {
		r.addResult("C#", "HTTP/health", false, "", err.Error())
		return nil
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	if resp.StatusCode == 200 && strings.Contains(string(body), "ok") {
		r.addResult("C#", "HTTP/health", true, string(body), "")
	} else {
		r.addResult("C#", "HTTP/health", false, string(body), "")
	}

	// 测试 add
	payload := `{"params": ["123", "877"]}`
	resp, err = r.client.Post(addr+"/add", "application/json", bytes.NewBufferString(payload))
	if err != nil {
		r.addResult("C#", "HTTP/add", false, "", err.Error())
		return nil
	}
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	if strings.Contains(string(body), "1000") {
		r.addResult("C#", "HTTP/add(123+877=1000)", true, string(body), "")
	} else {
		r.addResult("C#", "HTTP/add", false, string(body), "结果不正确")
	}

	// 测试 factorial
	payload = `{"params": ["10"]}`
	resp, err = r.client.Post(addr+"/factorial", "application/json", bytes.NewBufferString(payload))
	if err != nil {
		r.addResult("C#", "HTTP/factorial", false, "", err.Error())
		return nil
	}
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	if strings.Contains(string(body), "3628800") {
		r.addResult("C#", "HTTP/factorial(10!=3628800)", true, string(body), "")
	} else {
		r.addResult("C#", "HTTP/factorial", false, string(body), "结果不正确")
	}

	return nil
}

// ═══════════════ JavaScript 测试 ═══════════════

func (r *Runner) testJS() error {
	r.printLangHeader("JavaScript", "HTTP 服务 + 进程调用测试")

	jsDir := filepath.Join(r.testsDir, "js")

	if !checkCommand("node", "--version") {
		r.addResult("JavaScript", "环境检查", false, "", "未找到 Node.js")
		return nil
	}
	r.addResult("JavaScript", "环境检查", true, "node 已安装", "")

	// 测试 1: 进程调用
	r.testJSProcess(jsDir)

	// 测试 2: HTTP 服务
	r.testJSHTTP(jsDir)

	return nil
}

func (r *Runner) testJSProcess(jsDir string) {
	scriptPath := filepath.Join(jsDir, "jiasine_js_process.js")

	// 测试 add
	cmd := exec.Command("node", scriptPath, "add", "42", "58")
	output, err := cmd.CombinedOutput()
	if err != nil {
		r.addResult("JavaScript", "进程调用/add", false, string(output), err.Error())
		return
	}
	outStr := strings.TrimSpace(string(output))
	if strings.Contains(outStr, `"result":100`) || strings.Contains(outStr, `"result": 100`) {
		r.addResult("JavaScript", "进程调用/add(42+58=100)", true, outStr, "")
	} else {
		r.addResult("JavaScript", "进程调用/add", false, outStr, "结果不正确")
	}

	// 测试 reverse
	cmd = exec.Command("node", scriptPath, "reverse", "Jiasine")
	output, err = cmd.CombinedOutput()
	if err != nil {
		r.addResult("JavaScript", "进程调用/reverse", false, string(output), err.Error())
		return
	}
	outStr = strings.TrimSpace(string(output))
	if strings.Contains(outStr, "enisaiJ") {
		r.addResult("JavaScript", "进程调用/reverse(Jiasine)", true, outStr, "")
	} else {
		r.addResult("JavaScript", "进程调用/reverse", false, outStr, "结果不正确")
	}

	// 测试 fibonacci
	cmd = exec.Command("node", scriptPath, "fibonacci", "8")
	output, err = cmd.CombinedOutput()
	if err != nil {
		r.addResult("JavaScript", "进程调用/fibonacci", false, string(output), err.Error())
		return
	}
	outStr = strings.TrimSpace(string(output))
	if strings.Contains(outStr, "fibonacci") && strings.Contains(outStr, "0,1,1,2,3,5,8,13") || strings.Contains(outStr, "0, 1, 1, 2, 3, 5, 8, 13") {
		r.addResult("JavaScript", "进程调用/fibonacci(8)", true, outStr, "")
	} else {
		r.addResult("JavaScript", "进程调用/fibonacci", false, outStr, "结果不正确")
	}
}

func (r *Runner) testJSHTTP(jsDir string) {
	scriptPath := filepath.Join(jsDir, "jiasine_js_test.js")
	port := findFreePort()

	cmd := exec.Command("node", scriptPath, "--port", fmt.Sprintf("%d", port))
	cmd.Dir = jsDir
	if err := cmd.Start(); err != nil {
		r.addResult("JavaScript", "HTTP服务启动", false, "", err.Error())
		return
	}
	defer func() {
		cmd.Process.Kill()
		cmd.Wait()
	}()

	addr := fmt.Sprintf("http://127.0.0.1:%d", port)
	if !waitForService(addr+"/health", 5*time.Second) {
		r.addResult("JavaScript", "HTTP服务启动", false, "", "服务启动超时")
		return
	}
	r.addResult("JavaScript", "HTTP服务启动", true, fmt.Sprintf("端口 %d", port), "")

	// 测试 health
	resp, err := r.client.Get(addr + "/health")
	if err != nil {
		r.addResult("JavaScript", "HTTP/health", false, "", err.Error())
		return
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode == 200 && strings.Contains(string(body), "ok") {
		r.addResult("JavaScript", "HTTP/health", true, string(body), "")
	} else {
		r.addResult("JavaScript", "HTTP/health", false, string(body), "状态码非200")
	}

	// 测试 add
	payload := `{"params": ["25", "75"]}`
	resp, err = r.client.Post(addr+"/add", "application/json", bytes.NewBufferString(payload))
	if err != nil {
		r.addResult("JavaScript", "HTTP/add", false, "", err.Error())
		return
	}
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	if strings.Contains(string(body), "100") {
		r.addResult("JavaScript", "HTTP/add(25+75=100)", true, string(body), "")
	} else {
		r.addResult("JavaScript", "HTTP/add", false, string(body), "结果不正确")
	}

	// 测试 factorial
	payload = `{"params": ["10"]}`
	resp, err = r.client.Post(addr+"/factorial", "application/json", bytes.NewBufferString(payload))
	if err != nil {
		r.addResult("JavaScript", "HTTP/factorial", false, "", err.Error())
		return
	}
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	if strings.Contains(string(body), "3628800") {
		r.addResult("JavaScript", "HTTP/factorial(10!=3628800)", true, string(body), "")
	} else {
		r.addResult("JavaScript", "HTTP/factorial", false, string(body), "结果不正确")
	}
}

// ═══════════════ TypeScript 测试 ═══════════════

func (r *Runner) testTypeScript() error {
	r.printLangHeader("TypeScript", "HTTP 服务 + 进程调用测试")

	tsDir := filepath.Join(r.testsDir, "typescript")

	if !checkCommand("node", "--version") {
		r.addResult("TypeScript", "环境检查", false, "", "未找到 Node.js")
		return nil
	}
	if !checkCommand("npx", "--version") {
		r.addResult("TypeScript", "环境检查", false, "", "未找到 npx")
		return nil
	}
	r.addResult("TypeScript", "环境检查", true, "node + npx 已安装", "")

	// 确保依赖已安装
	nodeModules := filepath.Join(tsDir, "node_modules")
	if _, err := os.Stat(nodeModules); os.IsNotExist(err) {
		r.addResult("TypeScript", "安装依赖", false, "", "请先运行 cd tests/typescript && npm install")
		// 尝试自动安装
		installCmd := exec.Command("npm", "install", "--prefer-offline", "--no-audit", "--no-fund")
		installCmd.Dir = tsDir
		output, err := installCmd.CombinedOutput()
		if err != nil {
			r.addResult("TypeScript", "自动安装依赖", false, string(output), err.Error())
			return nil
		}
		r.addResult("TypeScript", "自动安装依赖", true, "npm install 成功", "")
	}

	// 测试 1: 进程调用
	r.testTSProcess(tsDir)

	// 测试 2: HTTP 服务
	r.testTSHTTP(tsDir)

	return nil
}

func (r *Runner) testTSProcess(tsDir string) {
	scriptPath := filepath.Join(tsDir, "jiasine_ts_process.ts")

	// 测试 add
	cmd := exec.Command("npx", "tsx", scriptPath, "add", "42", "58")
	cmd.Dir = tsDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		r.addResult("TypeScript", "进程调用/add", false, string(output), err.Error())
		return
	}
	outStr := strings.TrimSpace(string(output))
	if strings.Contains(outStr, `"result":100`) || strings.Contains(outStr, `"result": 100`) {
		r.addResult("TypeScript", "进程调用/add(42+58=100)", true, outStr, "")
	} else {
		r.addResult("TypeScript", "进程调用/add", false, outStr, "结果不正确")
	}

	// 测试 reverse
	cmd = exec.Command("npx", "tsx", scriptPath, "reverse", "Jiasine")
	cmd.Dir = tsDir
	output, err = cmd.CombinedOutput()
	if err != nil {
		r.addResult("TypeScript", "进程调用/reverse", false, string(output), err.Error())
		return
	}
	outStr = strings.TrimSpace(string(output))
	if strings.Contains(outStr, "enisaiJ") {
		r.addResult("TypeScript", "进程调用/reverse(Jiasine)", true, outStr, "")
	} else {
		r.addResult("TypeScript", "进程调用/reverse", false, outStr, "结果不正确")
	}

	// 测试 fibonacci
	cmd = exec.Command("npx", "tsx", scriptPath, "fibonacci", "8")
	cmd.Dir = tsDir
	output, err = cmd.CombinedOutput()
	if err != nil {
		r.addResult("TypeScript", "进程调用/fibonacci", false, string(output), err.Error())
		return
	}
	outStr = strings.TrimSpace(string(output))
	if strings.Contains(outStr, "fibonacci") && strings.Contains(outStr, "0,1,1,2,3,5,8,13") || strings.Contains(outStr, "0, 1, 1, 2, 3, 5, 8, 13") {
		r.addResult("TypeScript", "进程调用/fibonacci(8)", true, outStr, "")
	} else {
		r.addResult("TypeScript", "进程调用/fibonacci", false, outStr, "结果不正确")
	}
}

func (r *Runner) testTSHTTP(tsDir string) {
	scriptPath := filepath.Join(tsDir, "jiasine_ts_test.ts")
	port := findFreePort()

	cmd := exec.Command("npx", "tsx", scriptPath, "--port", fmt.Sprintf("%d", port))
	cmd.Dir = tsDir
	if err := cmd.Start(); err != nil {
		r.addResult("TypeScript", "HTTP服务启动", false, "", err.Error())
		return
	}
	defer func() {
		cmd.Process.Kill()
		cmd.Wait()
	}()

	addr := fmt.Sprintf("http://127.0.0.1:%d", port)
	if !waitForService(addr+"/health", 10*time.Second) {
		r.addResult("TypeScript", "HTTP服务启动", false, "", "服务启动超时")
		return
	}
	r.addResult("TypeScript", "HTTP服务启动", true, fmt.Sprintf("端口 %d", port), "")

	// 测试 health
	resp, err := r.client.Get(addr + "/health")
	if err != nil {
		r.addResult("TypeScript", "HTTP/health", false, "", err.Error())
		return
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode == 200 && strings.Contains(string(body), "ok") {
		r.addResult("TypeScript", "HTTP/health", true, string(body), "")
	} else {
		r.addResult("TypeScript", "HTTP/health", false, string(body), "状态码非200")
	}

	// 测试 add
	payload := `{"params": ["25", "75"]}`
	resp, err = r.client.Post(addr+"/add", "application/json", bytes.NewBufferString(payload))
	if err != nil {
		r.addResult("TypeScript", "HTTP/add", false, "", err.Error())
		return
	}
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	if strings.Contains(string(body), "100") {
		r.addResult("TypeScript", "HTTP/add(25+75=100)", true, string(body), "")
	} else {
		r.addResult("TypeScript", "HTTP/add", false, string(body), "结果不正确")
	}

	// 测试 factorial
	payload = `{"params": ["10"]}`
	resp, err = r.client.Post(addr+"/factorial", "application/json", bytes.NewBufferString(payload))
	if err != nil {
		r.addResult("TypeScript", "HTTP/factorial", false, "", err.Error())
		return
	}
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	if strings.Contains(string(body), "3628800") {
		r.addResult("TypeScript", "HTTP/factorial(10!=3628800)", true, string(body), "")
	} else {
		r.addResult("TypeScript", "HTTP/factorial", false, string(body), "结果不正确")
	}
}

// ═══════════════ Java 测试 ═══════════════

func (r *Runner) testJava() error {
	r.printLangHeader("Java", "HTTP 服务 + 进程调用测试")

	javaDir := filepath.Join(r.testsDir, "java")

	if !checkCommand("javac", "-version") {
		r.addResult("Java", "环境检查", false, "", "未找到 javac, 请安装 JDK")
		return nil
	}
	if !checkCommand("java", "--version") && !checkCommand("java", "-version") {
		r.addResult("Java", "环境检查", false, "", "未找到 java 运行时")
		return nil
	}
	r.addResult("Java", "环境检查", true, "javac + java 已安装", "")

	// 编译所有 .java 文件
	compileCmd := exec.Command("javac", "JiasineJavaTest.java", "JiasineJavaProcess.java")
	compileCmd.Dir = javaDir
	output, err := compileCmd.CombinedOutput()
	if err != nil {
		r.addResult("Java", "编译", false, string(output), err.Error())
		return nil
	}
	r.addResult("Java", "编译", true, "javac 编译成功", "")

	// 测试 1: 进程调用
	r.testJavaProcess(javaDir)

	// 测试 2: HTTP 服务
	r.testJavaHTTP(javaDir)

	return nil
}

func (r *Runner) testJavaProcess(javaDir string) {
	// 测试 add
	cmd := exec.Command("java", "-cp", javaDir, "JiasineJavaProcess", "add", "42", "58")
	output, err := cmd.CombinedOutput()
	if err != nil {
		r.addResult("Java", "进程调用/add", false, string(output), err.Error())
		return
	}
	outStr := strings.TrimSpace(string(output))
	if strings.Contains(outStr, `"result": 100`) {
		r.addResult("Java", "进程调用/add(42+58=100)", true, outStr, "")
	} else {
		r.addResult("Java", "进程调用/add", false, outStr, "结果不正确")
	}

	// 测试 reverse
	cmd = exec.Command("java", "-cp", javaDir, "JiasineJavaProcess", "reverse", "Jiasine")
	output, err = cmd.CombinedOutput()
	if err != nil {
		r.addResult("Java", "进程调用/reverse", false, string(output), err.Error())
		return
	}
	outStr = strings.TrimSpace(string(output))
	if strings.Contains(outStr, "enisaiJ") {
		r.addResult("Java", "进程调用/reverse(Jiasine)", true, outStr, "")
	} else {
		r.addResult("Java", "进程调用/reverse", false, outStr, "结果不正确")
	}

	// 测试 fibonacci
	cmd = exec.Command("java", "-cp", javaDir, "JiasineJavaProcess", "fibonacci", "8")
	output, err = cmd.CombinedOutput()
	if err != nil {
		r.addResult("Java", "进程调用/fibonacci", false, string(output), err.Error())
		return
	}
	outStr = strings.TrimSpace(string(output))
	if strings.Contains(outStr, "fibonacci") && strings.Contains(outStr, "0, 1, 1, 2, 3, 5, 8, 13") {
		r.addResult("Java", "进程调用/fibonacci(8)", true, outStr, "")
	} else {
		r.addResult("Java", "进程调用/fibonacci", false, outStr, "结果不正确")
	}
}

func (r *Runner) testJavaHTTP(javaDir string) {
	port := findFreePort()

	cmd := exec.Command("java", "-cp", javaDir, "JiasineJavaTest", "--port", fmt.Sprintf("%d", port))
	if err := cmd.Start(); err != nil {
		r.addResult("Java", "HTTP服务启动", false, "", err.Error())
		return
	}
	defer func() {
		cmd.Process.Kill()
		cmd.Wait()
	}()

	addr := fmt.Sprintf("http://127.0.0.1:%d", port)
	if !waitForService(addr+"/health", 10*time.Second) {
		r.addResult("Java", "HTTP服务启动", false, "", "服务启动超时")
		return
	}
	r.addResult("Java", "HTTP服务启动", true, fmt.Sprintf("端口 %d", port), "")

	// 测试 health
	resp, err := r.client.Get(addr + "/health")
	if err != nil {
		r.addResult("Java", "HTTP/health", false, "", err.Error())
		return
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode == 200 && strings.Contains(string(body), "ok") {
		r.addResult("Java", "HTTP/health", true, string(body), "")
	} else {
		r.addResult("Java", "HTTP/health", false, string(body), "状态码非200")
	}

	// 测试 add
	payload := `{"params": ["123", "877"]}`
	resp, err = r.client.Post(addr+"/add", "application/json", bytes.NewBufferString(payload))
	if err != nil {
		r.addResult("Java", "HTTP/add", false, "", err.Error())
		return
	}
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	if strings.Contains(string(body), "1000") {
		r.addResult("Java", "HTTP/add(123+877=1000)", true, string(body), "")
	} else {
		r.addResult("Java", "HTTP/add", false, string(body), "结果不正确")
	}

	// 测试 factorial
	payload = `{"params": ["10"]}`
	resp, err = r.client.Post(addr+"/factorial", "application/json", bytes.NewBufferString(payload))
	if err != nil {
		r.addResult("Java", "HTTP/factorial", false, "", err.Error())
		return
	}
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	if strings.Contains(string(body), "3628800") {
		r.addResult("Java", "HTTP/factorial(10!=3628800)", true, string(body), "")
	} else {
		r.addResult("Java", "HTTP/factorial", false, string(body), "结果不正确")
	}
}

// ═══════════════ Swift 测试 ═══════════════

func (r *Runner) testSwift() error {
	r.printLangHeader("Swift", "编译 + 进程调用测试")

	swiftDir := filepath.Join(r.testsDir, "swift")

	if !checkCommand("swiftc", "--version") {
		r.addResult("Swift", "环境检查", false, "", "未找到 swiftc, 请安装 Swift 工具链")
		return nil
	}
	r.addResult("Swift", "环境检查", true, "swiftc 已安装", "")

	// 编译进程调用工具
	var binPath string
	if runtime.GOOS == "windows" {
		binPath = filepath.Join(swiftDir, "jiasine_swift_process.exe")
	} else {
		binPath = filepath.Join(swiftDir, "jiasine_swift_process")
	}

	compileCmd := exec.Command("swiftc", "-o", binPath, filepath.Join(swiftDir, "JiasineSwiftProcess.swift"))
	compileCmd.Dir = swiftDir
	output, err := compileCmd.CombinedOutput()
	if err != nil {
		r.addResult("Swift", "编译(Process)", false, string(output), err.Error())
		return nil
	}
	r.addResult("Swift", "编译(Process)", true, "swiftc 编译成功", "")

	// 测试进程调用
	r.testSwiftProcess(binPath)

	// 编译 HTTP 服务
	var svcBinPath string
	if runtime.GOOS == "windows" {
		svcBinPath = filepath.Join(swiftDir, "jiasine_swift_test.exe")
	} else {
		svcBinPath = filepath.Join(swiftDir, "jiasine_swift_test")
	}

	compileCmd = exec.Command("swiftc", "-o", svcBinPath, filepath.Join(swiftDir, "JiasineSwiftTest.swift"))
	compileCmd.Dir = swiftDir
	output, err = compileCmd.CombinedOutput()
	if err != nil {
		r.addResult("Swift", "编译(HTTP)", false, string(output), err.Error())
		return nil
	}
	r.addResult("Swift", "编译(HTTP)", true, "swiftc 编译成功", "")

	// 测试 HTTP 服务
	r.testSwiftHTTP(svcBinPath)

	return nil
}

func (r *Runner) testSwiftProcess(binPath string) {
	// 测试 add
	cmd := exec.Command(binPath, "add", "42", "58")
	output, err := cmd.CombinedOutput()
	if err != nil {
		r.addResult("Swift", "进程调用/add", false, string(output), err.Error())
		return
	}
	outStr := strings.TrimSpace(string(output))
	if strings.Contains(outStr, "100") && strings.Contains(outStr, "Swift") {
		r.addResult("Swift", "进程调用/add(42+58=100)", true, outStr, "")
	} else {
		r.addResult("Swift", "进程调用/add", false, outStr, "结果不正确")
	}

	// 测试 reverse
	cmd = exec.Command(binPath, "reverse", "Jiasine")
	output, err = cmd.CombinedOutput()
	if err != nil {
		r.addResult("Swift", "进程调用/reverse", false, string(output), err.Error())
		return
	}
	outStr = strings.TrimSpace(string(output))
	if strings.Contains(outStr, "enisaiJ") {
		r.addResult("Swift", "进程调用/reverse(Jiasine)", true, outStr, "")
	} else {
		r.addResult("Swift", "进程调用/reverse", false, outStr, "结果不正确")
	}

	// 测试 fibonacci
	cmd = exec.Command(binPath, "fibonacci", "8")
	output, err = cmd.CombinedOutput()
	if err != nil {
		r.addResult("Swift", "进程调用/fibonacci", false, string(output), err.Error())
		return
	}
	outStr = strings.TrimSpace(string(output))
	if strings.Contains(outStr, "fibonacci") {
		r.addResult("Swift", "进程调用/fibonacci(8)", true, outStr, "")
	} else {
		r.addResult("Swift", "进程调用/fibonacci", false, outStr, "结果不正确")
	}
}

func (r *Runner) testSwiftHTTP(binPath string) {
	port := findFreePort()

	cmd := exec.Command(binPath, "--port", fmt.Sprintf("%d", port))
	if err := cmd.Start(); err != nil {
		r.addResult("Swift", "HTTP服务启动", false, "", err.Error())
		return
	}
	defer func() {
		cmd.Process.Kill()
		cmd.Wait()
	}()

	addr := fmt.Sprintf("http://127.0.0.1:%d", port)
	if !waitForService(addr+"/health", 5*time.Second) {
		r.addResult("Swift", "HTTP服务启动", false, "", "服务启动超时")
		return
	}
	r.addResult("Swift", "HTTP服务启动", true, fmt.Sprintf("端口 %d", port), "")

	// 测试 health
	resp, err := r.client.Get(addr + "/health")
	if err != nil {
		r.addResult("Swift", "HTTP/health", false, "", err.Error())
		return
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode == 200 && strings.Contains(string(body), "ok") {
		r.addResult("Swift", "HTTP/health", true, string(body), "")
	} else {
		r.addResult("Swift", "HTTP/health", false, string(body), "状态码非200")
	}

	// 测试 add
	payload := `{"params": ["25", "75"]}`
	resp, err = r.client.Post(addr+"/add", "application/json", bytes.NewBufferString(payload))
	if err != nil {
		r.addResult("Swift", "HTTP/add", false, "", err.Error())
		return
	}
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	if strings.Contains(string(body), "100") {
		r.addResult("Swift", "HTTP/add(25+75=100)", true, string(body), "")
	} else {
		r.addResult("Swift", "HTTP/add", false, string(body), "结果不正确")
	}
}

// ═══════════════ Objective-C 测试 ═══════════════

func (r *Runner) testObjC() error {
	r.printLangHeader("Objective-C", "FFI 动态库测试")

	objcDir := filepath.Join(r.testsDir, "objc")
	srcFile := filepath.Join(objcDir, "JiasineObjcTest.m")

	if _, err := os.Stat(srcFile); os.IsNotExist(err) {
		r.addResult("Objective-C", "源文件检查", false, "", "JiasineObjcTest.m 不存在")
		return nil
	}

	// 编译为动态库
	var dllPath string
	var compileCmd *exec.Cmd
	if runtime.GOOS == "windows" {
		dllPath = filepath.Join(objcDir, "jiasine_objc_test.dll")
		// Windows: 使用 GCC 编译纯 C 回退模式
		if checkCommand("gcc", "--version") {
			compileCmd = exec.Command("gcc", "-shared", "-DOBJC_FALLBACK_C",
				"-o", dllPath, srcFile)
		} else {
			r.addResult("Objective-C", "编译", false, "", "未找到 gcc 编译器")
			return nil
		}
	} else if runtime.GOOS == "darwin" {
		dllPath = filepath.Join(objcDir, "libjiasine_objc_test.dylib")
		// macOS: 使用 clang 编译完整 ObjC
		compileCmd = exec.Command("clang", "-shared", "-fPIC",
			"-framework", "Foundation",
			"-o", dllPath, srcFile)
	} else {
		dllPath = filepath.Join(objcDir, "libjiasine_objc_test.so")
		// Linux: 尝试 ObjC 编译，失败则回退
		if checkCommand("gnustep-config", "--version") {
			compileCmd = exec.Command("bash", "-c",
				fmt.Sprintf("gcc -shared -fPIC -o %s %s $(gnustep-config --objc-flags --base-libs)", dllPath, srcFile))
		} else {
			compileCmd = exec.Command("gcc", "-shared", "-fPIC", "-DOBJC_FALLBACK_C",
				"-o", dllPath, srcFile)
		}
	}

	compileCmd.Dir = objcDir
	output, err := compileCmd.CombinedOutput()
	if err != nil {
		r.addResult("Objective-C", "编译动态库", false, string(output), err.Error())
		return nil
	}
	r.addResult("Objective-C", "编译动态库", true, fmt.Sprintf("成功: %s", filepath.Base(dllPath)), "")

	// FFI 调用测试 (与 C 相同的 DLL 接口)
	r.testObjCFFI(dllPath)

	return nil
}

func (r *Runner) testObjCFFI(dllPath string) {
	if runtime.GOOS != "windows" {
		r.addResult("Objective-C", "FFI 调用", false, "", "当前仅支持 Windows DLL 直接测试，其他平台请使用 CGO")
		return
	}
	// 复用与 C 相同的 DLL 测试逻辑 (Objective-C 导出的是 C 兼容接口)
	testCallDLLWithLang(r, dllPath, "Objective-C")
}

// ═══════════════ 工具函数 ═══════════════

func (r *Runner) printLangHeader(lang, desc string) {
	fmt.Printf("── %s: %s ──\n", lang, desc)
}

func (r *Runner) addResult(lang, testName string, passed bool, output, errMsg string) {
	icon := "✗ FAIL"
	if passed {
		icon = "✓ PASS"
	}

	fmt.Printf("  [%s] %s/%s", icon, lang, testName)
	if output != "" && passed {
		// 截断长输出
		if len(output) > 80 {
			output = output[:80] + "..."
		}
		fmt.Printf("  → %s", output)
	}
	if errMsg != "" {
		fmt.Printf("  (%s)", errMsg)
	}
	fmt.Println()

	r.results = append(r.results, TestResult{
		Language: lang,
		TestName: testName,
		Passed:   passed,
		Output:   output,
		Error:    errMsg,
	})
}

func (r *Runner) printSummary() {
	passed := 0
	failed := 0
	for _, res := range r.results {
		if res.Passed {
			passed++
		} else {
			failed++
		}
	}

	fmt.Println("═══════════════════════════════════════════")
	fmt.Printf("  测试汇总: 共 %d 项, 通过 %d, 失败 %d\n", passed+failed, passed, failed)
	if failed == 0 {
		fmt.Println("  状态: 全部通过 ✓")
	} else {
		fmt.Println("  状态: 存在失败 ✗")
		fmt.Println("  失败项:")
		for _, res := range r.results {
			if !res.Passed {
				fmt.Printf("    - %s/%s: %s\n", res.Language, res.TestName, res.Error)
			}
		}
	}
	fmt.Println("═══════════════════════════════════════════")
}

func checkCommand(name string, arg string) bool {
	args := []string{}
	if arg != "" {
		args = append(args, arg)
	}
	cmd := exec.Command(name, args...)
	err := cmd.Run()
	return err == nil
}

func findPython() string {
	for _, name := range []string{"python", "python3"} {
		if checkCommand(name, "--version") {
			return name
		}
	}
	return ""
}

func findFreePort() int {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 9999
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	return port
}

func waitForService(url string, timeout time.Duration) bool {
	client := &http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode < 500 {
				return true
			}
		}
		time.Sleep(300 * time.Millisecond)
	}
	return false
}

// testCallDLL Windows DLL 调用测试 (在 ffi_test_windows.go / ffi_test_other.go 中实现)
// 这里使用 bridge 包的能力
func testCallDLL(r *Runner, dllPath string) {
	testCallDLLPlatform(r, dllPath)
}

// testCallDLLWithLang 带语言标签的 DLL 调用测试
func testCallDLLWithLang(r *Runner, dllPath string, lang string) {
	testCallDLLWithLangPlatform(r, dllPath, lang)
}

// jsonPretty 格式化 JSON 输出
func jsonPretty(s string) string {
	var out bytes.Buffer
	if err := json.Indent(&out, []byte(s), "", "  "); err != nil {
		return s
	}
	return out.String()
}
