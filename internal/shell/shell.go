// Package shell 提供交互式命令行 Shell
// 当用户双击 .exe 启动时，自动进入交互模式
package shell

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strings"

	"github.com/xiangjianhe-github/jiasinecli/internal/banner"
	"github.com/xiangjianhe-github/jiasinecli/internal/config"
	"github.com/xiangjianhe-github/jiasinecli/internal/theme"
	"github.com/xiangjianhe-github/jiasinecli/internal/tui"
)

// ── 命令分类定义 ─────────────────────────────────────────────────────────────

// cmdInfo 命令信息
type cmdInfo struct {
	Name string
	Desc string
}

// 全局命令
var globalCmds = []cmdInfo{
	{"/ai", "进入 AI 交互模式"},
	{"/model", "选择 AI 模型"},
	{"/theme", "查看或切换终端主题"},
	{"/clear", "清空屏幕"},
	{"/help", "显示帮助"},
	{"/exit", "退出"},
}

// Agent 环境命令
var agentCmds = []cmdInfo{
	{"/init", "初始化项目配置"},
	{"/agent", "浏览并选择可用 Agent"},
	{"/skills", "管理增强能力 Skills"},
	{"/mcp", "管理 MCP 服务器配置"},
	{"/plugin", "插件管理"},
}

// 模型与子代理命令
var modelCmds = []cmdInfo{
	{"/model", "选择 AI 模型"},
	{"/delegate", "委托任务给 Copilot 创建 PR"},
	{"/fleet", "启用 Fleet 模式并行子代理"},
	{"/tasks", "查看和管理后台任务"},
}

// 代码命令
var codeCmds = []cmdInfo{
	{"/ide", "连接 IDE 工作区"},
	{"/diff", "查看当前目录变更"},
	{"/review", "运行代码审查"},
	{"/lsp", "管理语言服务器配置"},
	{"/terminal-setup", "配置终端多行输入支持"},
}

// 权限命令
var permCmds = []cmdInfo{
	{"/allow-all", "启用所有权限"},
	{"/add-dir", "添加允许访问的目录"},
	{"/list-dirs", "列出所有允许目录"},
	{"/cwd", "切换/显示工作目录"},
	{"/reset-allowed-tools", "重置允许的工具列表"},
}

// 会话命令
var sessionCmds = []cmdInfo{
	{"/resume", "切换到其他会话"},
	{"/rename", "重命名当前会话"},
	{"/context", "显示上下文窗口使用情况"},
	{"/usage", "显示会话使用统计"},
	{"/session", "查看会话信息"},
	{"/compact", "压缩对话历史"},
	{"/share", "分享会话到文件"},
	{"/copy", "复制上次回复到剪贴板"},
}

// 帮助和反馈命令
var helpCmds = []cmdInfo{
	{"/help", "显示帮助"},
	{"/changelog", "查看版本更新日志"},
	{"/feedback", "提供反馈"},
	{"/theme", "查看或切换终端主题"},
	{"/update", "更新 CLI 到最新版本"},
	{"/experimental", "实验性功能"},
	{"/clear", "清空对话历史"},
	{"/instructions", "查看自定义指令文件"},
	{"/streamer-mode", "切换直播模式"},
}

// 其他命令
var otherCmds = []cmdInfo{
	{"/exit", "退出 CLI"},
	{"/quit", "退出 CLI"},
	{"/login", "登录"},
	{"/logout", "登出"},
	{"/plan", "创建实施计划"},
	{"/research", "深度研究调查"},
	{"/user", "管理用户列表"},
}

// 系统级 Cobra 命令（旧命令保留兼容）
var cobraCmds = []cmdInfo{
	{"/version", "查看版本信息"},
	{"/test", "运行集成测试"},
	{"/bridge", "桥接层管理"},
	{"/service", "服务管理"},
	{"/history", "查看对话历史"},
	{"/setup", "首次配置引导"},
}

// allSelectableCmds 合并所有可用于选择器的独立命令（去重）
func allSelectableCmds() []cmdInfo {
	seen := make(map[string]bool)
	var result []cmdInfo
	addGroup := func(cmds []cmdInfo) {
		for _, c := range cmds {
			if !seen[c.Name] {
				seen[c.Name] = true
				result = append(result, c)
			}
		}
	}
	addGroup(globalCmds)
	addGroup(agentCmds)
	addGroup(codeCmds)
	addGroup(sessionCmds)
	addGroup(cobraCmds)
	addGroup(otherCmds)
	return result
}

// allCmdNames 所有命令名（用于 Tab 补全）
func allCmdNames() []string {
	all := allSelectableCmds()
	names := make([]string, len(all))
	for i, c := range all {
		names[i] = c.Name
	}
	return names
}

// ── 帮助文本 ─────────────────────────────────────────────────────────────────

func interactiveHelp() string {
	g := banner.BrightGreen
	c := banner.BrightCyan
	d := banner.Dim
	b := banner.Bold
	y := banner.BrightYellow
	r := banner.Reset

	var sb strings.Builder

	// ── 快捷键 ──
	sb.WriteString("\n" + b + c + "全局快捷键" + r + "\n")
	keys := [][2]string{
		{"Ctrl+C", "取消操作 / 清空输入 / 退出"},
		{"Ctrl+D", "关闭程序"},
		{"Ctrl+L", "清屏"},
		{"↑ ↓", "浏览命令历史"},
		{"Tab", "命令自动补全"},
		{"Esc", "取消当前操作"},
	}
	for _, k := range keys {
		sb.WriteString(fmt.Sprintf("  %s%-16s%s %s\n", y, k[0], r, k[1]))
	}

	sb.WriteString("\n" + b + c + "编辑快捷键" + r + "\n")
	editKeys := [][2]string{
		{"Ctrl+A", "移到行首"},
		{"Ctrl+E", "移到行尾"},
		{"Ctrl+H", "删除前一个字符"},
		{"Ctrl+W", "删除前一个单词"},
		{"Ctrl+U", "删除到行首"},
		{"Ctrl+K", "删除到行尾"},
	}
	for _, k := range editKeys {
		sb.WriteString(fmt.Sprintf("  %s%-16s%s %s\n", y, k[0], r, k[1]))
	}

	// ── 命令分区 ──
	printSection := func(title string, cmds []cmdInfo) {
		sb.WriteString("\n" + b + c + title + r + "\n")
		for _, cmd := range cmds {
			name := cmd.Name
			sb.WriteString(fmt.Sprintf("  %s%-22s%s %s%s%s\n", g, name, r, d, cmd.Desc, r))
		}
	}

	printSection("Agent 环境", agentCmds)
	printSection("模型与子代理", modelCmds)
	printSection("代码", codeCmds)
	printSection("权限", permCmds)
	printSection("会话", sessionCmds)
	printSection("帮助和反馈", helpCmds)
	printSection("系统", cobraCmds)
	printSection("其他", otherCmds)

	sb.WriteString("\n" + d + "提示: 输入 / 后按 Tab 补全 · 单独输入 / 按 Enter 弹出命令选择器" + r + "\n\n")
	return sb.String()
}

// ── 主题 ─────────────────────────────────────────────────────────────────────

// resolveTheme 解析主题设置（处理 "auto" → 检测系统主题）
func resolveTheme() {
	themeName := config.GetTheme()
	if themeName == "auto" {
		themeName = DetectSystemTheme()
	}
	theme.Set(theme.ThemeName(themeName))
}

// ── 终端初始化 ────────────────────────────────────────────────────────────────

// InitTerminal 初始化终端环境（VT处理、配置、主题、颜色）
// 供外部调用，在进入 AI 模式前初始化终端显示
func InitTerminal() {
	enableVirtualTerminal()
	_ = config.Init("")
	resolveTheme()
	banner.RefreshColors()
}

// ── 交互式 Shell ─────────────────────────────────────────────────────────────

// RunInteractive 启动交互式 Shell 模式
func RunInteractive(executeFunc func(args []string) error) {
	// 内部 panic 保护：打印错误而不是闪退
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "\n⚠ Shell 内部错误: %v\n", r)
		}
	}()

	// 在 Windows 上启用 ANSI 转义序列
	enableVirtualTerminal()

	// 初始化配置和主题（静默失败）
	_ = config.Init("")
	resolveTheme()
	banner.RefreshColors()

	// 显示欢迎屏幕
	fmt.Print(banner.WelcomeScreen())

	// 拦截 Ctrl+C 信号（不退出，只提示）
	sigChan := make(chan os.Signal, 2)
	signal.Notify(sigChan, os.Interrupt)
	defer signal.Stop(sigChan)
	go func() {
		for range sigChan {
			fmt.Printf("\n%s输入 /exit 退出，/help 查看帮助%s\n", banner.Dim, banner.Reset)
			printPrompt()
		}
	}()

	// 使用 bufio.NewReader 逐行读取 — 不会提前缓冲后续行
	// 这样当 /ai 等子命令接管 os.Stdin 时不会冲突
	reader := bufio.NewReader(os.Stdin)
	for {
		printPrompt()
		line, err := reader.ReadString('\n')
		if err != nil {
			// EOF 或读取错误 → 退出
			fmt.Print(banner.Farewell())
			return
		}

		input := strings.TrimSpace(line)
		input = strings.TrimLeft(input, "\uFEFF\u200B\u00A0")
		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		// 所有命令必须以 / 开头
		if !strings.HasPrefix(input, "/") {
			fmt.Printf("  %s提示: 使用 / 开头输入命令，如 /help 查看帮助%s\n\n",
				banner.Dim, banner.Reset)
			continue
		}

		// 单独输入 / → 显示命令选择器
		if input == "/" {
			cmds := allSelectableCmds()
			options := make([]tui.SelectOption, len(cmds))
			for i, c := range cmds {
				options[i] = tui.SelectOption{Label: c.Name, Description: c.Desc}
			}
			idx, _ := tui.Select("选择命令", options, 0)
			if idx < 0 {
				continue
			}
			input = cmds[idx].Name
		}

		// 解析命令
		cmd := strings.TrimPrefix(input, "/")
		parts := strings.Fields(cmd)
		if len(parts) == 0 {
			continue
		}
		cmdName := strings.ToLower(parts[0])

		// ── 内置命令 ──
		switch cmdName {
		case "exit", "quit", "q":
			fmt.Print(banner.Farewell())
			return
		case "clear", "cls":
			clearScreen()
			fmt.Print(banner.ShortBanner())
			fmt.Println()
			continue
		case "banner":
			fmt.Println(banner.Logo())
			continue
		case "help", "?":
			fmt.Print(interactiveHelp())
			continue
		case "theme":
			selectedKey := tui.PickTheme(string(theme.CurrentName()))
			if selectedKey != "" {
				theme.Set(theme.ThemeName(selectedKey))
				banner.RefreshColors()
				_ = config.SetTheme(selectedKey)
				label := "🌙 暗色主题"
				if selectedKey == "light" {
					label = "☀️ 亮色主题"
				}
				fmt.Printf("  %s✓ 已切换为 %s%s\n", banner.BrightGreen, label, banner.Reset)
			}
			continue
		case "jiasine", "jiasinecli":
			clearScreen()
			fmt.Print(banner.WelcomeScreen())
			continue
		case "copy":
			fmt.Printf("  %s📋 (暂未实现) 剪贴板功能%s\n", banner.Dim, banner.Reset)
			continue
		case "context", "usage":
			fmt.Printf("  %s📊 (暂未实现) 使用统计%s\n", banner.Dim, banner.Reset)
			continue
		case "compact":
			fmt.Printf("  %s📦 (暂未实现) 对话压缩%s\n", banner.Dim, banner.Reset)
			continue
		case "share":
			fmt.Printf("  %s📤 (暂未实现) 分享功能%s\n", banner.Dim, banner.Reset)
			continue
		case "streamer-mode":
			fmt.Printf("  %s🎬 (暂未实现) 直播模式%s\n", banner.Dim, banner.Reset)
			continue
		case "experimental":
			fmt.Printf("  %s🧪 (暂未实现) 实验性功能%s\n", banner.Dim, banner.Reset)
			continue
		case "instructions":
			fmt.Printf("  %s📝 (暂未实现) 自定义指令%s\n", banner.Dim, banner.Reset)
			continue
		case "login":
			fmt.Printf("  %s🔑 (暂未实现) 登录%s\n", banner.Dim, banner.Reset)
			continue
		case "logout":
			fmt.Printf("  %s🔓 (暂未实现) 登出%s\n", banner.Dim, banner.Reset)
			continue
		case "plan":
			fmt.Printf("  %s📋 (暂未实现) 实施计划%s\n", banner.Dim, banner.Reset)
			continue
		case "research":
			fmt.Printf("  %s🔍 (暂未实现) 深度研究%s\n", banner.Dim, banner.Reset)
			continue
		case "delegate":
			fmt.Printf("  %s🤖 (暂未实现) 委托任务%s\n", banner.Dim, banner.Reset)
			continue
		case "fleet":
			fmt.Printf("  %s⚡ (暂未实现) Fleet 模式%s\n", banner.Dim, banner.Reset)
			continue
		case "tasks":
			fmt.Printf("  %s📋 (暂未实现) 后台任务%s\n", banner.Dim, banner.Reset)
			continue
		case "ide":
			fmt.Printf("  %s💻 (暂未实现) IDE 连接%s\n", banner.Dim, banner.Reset)
			continue
		case "diff":
			fmt.Printf("  %s📝 (暂未实现) 变更对比%s\n", banner.Dim, banner.Reset)
			continue
		case "review":
			fmt.Printf("  %s🔍 (暂未实现) 代码审查%s\n", banner.Dim, banner.Reset)
			continue
		case "lsp":
			fmt.Printf("  %s🔧 (暂未实现) LSP 配置%s\n", banner.Dim, banner.Reset)
			continue
		case "terminal-setup":
			fmt.Printf("  %s⚙️  (暂未实现) 终端配置%s\n", banner.Dim, banner.Reset)
			continue
		case "allow-all":
			fmt.Printf("  %s✅ (暂未实现) 全部权限%s\n", banner.Dim, banner.Reset)
			continue
		case "add-dir":
			fmt.Printf("  %s📁 (暂未实现) 添加目录%s\n", banner.Dim, banner.Reset)
			continue
		case "list-dirs":
			fmt.Printf("  %s📂 (暂未实现) 目录列表%s\n", banner.Dim, banner.Reset)
			continue
		case "cwd":
			cwd, _ := os.Getwd()
			fmt.Printf("  %s📂 %s%s\n", banner.BrightCyan, cwd, banner.Reset)
			continue
		case "reset-allowed-tools":
			fmt.Printf("  %s🔄 (暂未实现) 重置工具权限%s\n", banner.Dim, banner.Reset)
			continue
		case "resume":
			fmt.Printf("  %s🔄 (暂未实现) 切换会话%s\n", banner.Dim, banner.Reset)
			continue
		case "rename":
			fmt.Printf("  %s✏️  (暂未实现) 重命名会话%s\n", banner.Dim, banner.Reset)
			continue
		case "session":
			fmt.Printf("  %s📋 (暂未实现) 会话信息%s\n", banner.Dim, banner.Reset)
			continue
		case "changelog":
			fmt.Printf("  %s📄 (暂未实现) 版本日志%s\n", banner.Dim, banner.Reset)
			continue
		case "feedback":
			fmt.Printf("  %s💬 (暂未实现) 反馈%s\n", banner.Dim, banner.Reset)
			continue
		case "update":
			// 转交给 Cobra 处理
		case "user":
			fmt.Printf("  %s👤 (暂未实现) 用户管理%s\n", banner.Dim, banner.Reset)
			continue
		case "init":
			fmt.Printf("  %s📋 (暂未实现) 项目初始化%s\n", banner.Dim, banner.Reset)
			continue
		case "mcp":
			fmt.Printf("  %s🔌 (暂未实现) MCP 配置%s\n", banner.Dim, banner.Reset)
			continue
		case "model":
			// 转交给 Cobra（ai provider）或内置处理
		}

		// 其余命令交给 Cobra 执行
		args := parseArgs(cmd)
		if err := executeFunc(args); err != nil {
			errMsg := err.Error()
			if strings.Contains(errMsg, "unknown command") {
				fmt.Printf("  %s未知命令: /%s，输入 /help 查看可用命令%s\n",
					banner.Bold+banner.BrightCyan, args[0], banner.Reset)
			} else {
				fmt.Printf("  %s错误: %v%s\n",
					banner.Bold+banner.BrightCyan, err, banner.Reset)
			}
		}
		fmt.Println()
	}
}

// printPrompt 输出命令提示符
func printPrompt() {
	fmt.Printf("%s%sjiasinecli%s%s > %s",
		banner.Bold, banner.BrightCyan,
		banner.Reset, banner.Dim, banner.Reset,
	)
}

// parseArgs 简单解析命令行参数（支持引号）
func parseArgs(input string) []string {
	var args []string
	var current strings.Builder
	inQuote := false
	quoteChar := byte(0)

	for i := 0; i < len(input); i++ {
		ch := input[i]
		switch {
		case !inQuote && (ch == '"' || ch == '\''):
			inQuote = true
			quoteChar = ch
		case inQuote && ch == quoteChar:
			inQuote = false
		case !inQuote && ch == ' ':
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteByte(ch)
		}
	}
	if current.Len() > 0 {
		args = append(args, current.String())
	}
	return args
}

// clearScreen 清空终端屏幕
func clearScreen() {
	fmt.Print("\033[2J\033[H")
}
