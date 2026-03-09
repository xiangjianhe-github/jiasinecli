package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/mattn/go-runewidth"
	"github.com/xiangjianhe-github/jiasinecli/internal/ai"
	"github.com/xiangjianhe-github/jiasinecli/internal/banner"
	"github.com/xiangjianhe-github/jiasinecli/internal/config"
	historydb "github.com/xiangjianhe-github/jiasinecli/internal/history"
	"github.com/xiangjianhe-github/jiasinecli/internal/render"
	"github.com/xiangjianhe-github/jiasinecli/internal/shell"
	"github.com/xiangjianhe-github/jiasinecli/internal/theme"
	"github.com/xiangjianhe-github/jiasinecli/internal/tui"
	"github.com/spf13/cobra"
)

var (
	aiProvider  string // --provider 标志
	aiModel     string // --model 标志
	aiAgent     string // --agent 标志
	aiWebSearch bool   // --web 标志
)

// getAIManager 懒加载 AI 管理器，自动检测并生成配置
func getAIManager() (*ai.Manager, error) {
	// 检查是否有有效的 AI 配置（只要有一个已启用 + 有 API Key 即可）
	if !config.HasValidAIProviders() {
		// 尝试自动生成配置模板
		configPath, created, err := config.EnsureAIConfig()
		if err != nil {
			return nil, fmt.Errorf("生成 AI 配置失败: %w", err)
		}

		// 生成模板后重新加载配置，检查是否已有有效 key
		if !created {
			// 文件已存在 — 重新加载一次，确认最新状态
			_ = config.Reload()
			if config.HasValidAIProviders() {
				goto loadManager // 配置可用，继续
			}
		}

		// 确实没有配置 API Key
		if created {
			fmt.Printf("\n%s📄 已自动生成 AI 配置模板:%s\n", banner.BrightCyan, banner.Reset)
			fmt.Printf("   %s%s%s\n\n", banner.BrightGreen, configPath, banner.Reset)
		}
		fmt.Printf("\n%s⚠ 未检测到已配置的 AI API Key%s\n", banner.Yellow, banner.Reset)
		fmt.Printf("请编辑: %s%s%s\n", banner.BrightGreen, configPath, banner.Reset)
		fmt.Printf("%s只需配置一个服务商即可使用，无需全部填写%s\n\n", banner.Dim, banner.Reset)
		fmt.Printf("%s各服务商 API Key 获取地址:%s\n", banner.Dim, banner.Reset)
		fmt.Printf("  DeepSeek  (推荐)  https://platform.deepseek.com\n")
		fmt.Printf("  OpenAI           https://platform.openai.com/api-keys\n")
		fmt.Printf("  Anthropic        https://console.anthropic.com\n")
		fmt.Printf("  Google Gemini    https://aistudio.google.com/apikey\n")
		fmt.Printf("  通义千问          https://dashscope.console.aliyun.com\n\n")
		return nil, fmt.Errorf("请先配置至少一个 AI 服务商的 API Key")
	}

loadManager:

	cfg := config.Get()
	mgr := ai.NewManager(ai.AIConfig{
		Active:    cfg.AI.Active,
		WebSearch: cfg.AI.WebSearch,
		Providers: cfg.AI.Providers,
	})

	if !mgr.HasProviders() {
		return nil, fmt.Errorf("未能加载任何 AI 提供商，请检查配置")
	}

	return mgr, nil
}

// getSkillManager 懒加载 Skill 管理器
func getSkillManager() *ai.SkillManager {
	cfg := config.Get()
	return ai.NewSkillManager(cfg.AI.Skills)
}

// getAgentManager 懒加载 Agent 管理器
func getAgentManager() *ai.AgentManager {
	cfg := config.Get()
	aiMgr, _ := getAIManager()
	skillMgr := getSkillManager()
	return ai.NewAgentManager(aiMgr, skillMgr, cfg.AI.Agents)
}

// ===== ai 根命令 =====

var aiCmd = &cobra.Command{
	Use:   "ai",
	Short: "AI 大模型交互",
	Long: `AI 插件 — 支持主流大模型统一调用

支持的 AI 服务商:
  • OpenAI (ChatGPT)    — gpt-4o, gpt-4-turbo, o1, o3-mini
  • Anthropic (Claude)  — claude-sonnet-4, claude-opus-4
  • Google (Gemini)     — gemini-2.5-pro, gemini-2.5-flash
  • 阿里云 (Qwen/通义)   — qwen-max, qwen-plus, qwen-turbo
  • DeepSeek            — deepseek-chat, deepseek-coder

子命令:
  chat       与 AI 对话
  agent      Agent 智能体管理
  skill      Skills 技能管理
  provider   查看/切换 AI 提供商

直接运行 'ai' (不带子命令) 将进入 AI 交互模式。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 'ai' 不带子命令 → 进入交互式 AI 对话模式（默认使用 general agent）
		return enterAIInteractive("general")
	},
}

// ===== ai chat =====

var aiChatCmd = &cobra.Command{
	Use:   "chat [message]",
	Short: "与 AI 对话",
	Long: `发送消息给 AI 大模型，获取回复。
不带参数将进入交互式 AI 对话模式 (Ctrl+C 退出)。

示例:
  jiasinecli ai chat                                    # 进入 AI 交互模式
  jiasinecli ai chat "什么是 Go 语言?"                    # 单次对话
  jiasinecli ai chat --provider claude "解释递归"          # 指定服务商
  jiasinecli ai chat --agent coder "用 Go 写 HTTP 服务"   # 使用 Agent`,
	Args: cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			// 无参数 → 进入交互模式（默认使用 general agent）
			agent := aiAgent
			if agent == "" {
				agent = "general"
			}
			return enterAIInteractive(agent)
		}

		// 有参数 → 单次对话（默认使用 general agent）
		prompt := strings.Join(args, " ")

		// 走 Agent 流程（默认 general）
		agentName := aiAgent
		if agentName == "" {
			agentName = "general"
		}
		agentMgr := getAgentManager()

		// 设置联网搜索（来自 --web 标志）
		if aiWebSearch {
			agentMgr.AIManager().SetWebSearch(true)
		}

		resp, err := agentMgr.Run(agentName, prompt)
		if err != nil {
			return err
		}
		printAIResponse(resp)
		return nil
	},
}

// ===== ai provider =====

var aiProviderCmd = &cobra.Command{
	Use:   "provider",
	Short: "AI 服务商管理",
}

var aiProviderListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出已配置的 AI 提供商",
	RunE: func(cmd *cobra.Command, args []string) error {
		aiMgr, err := getAIManager()
		if err != nil {
			return err
		}
		providers := aiMgr.ListProviders()

		if len(providers) == 0 {
			fmt.Println("未配置任何 AI 提供商")
			fmt.Println()
			fmt.Println("请在 ~/.jiasine/config.yaml 中配置 ai.providers:")
			fmt.Println("  ai:")
			fmt.Println("    active: openai")
			fmt.Println("    providers:")
			fmt.Println("      openai:")
			fmt.Println("        name: openai")
			fmt.Println("        api_key: sk-xxx")
			fmt.Println("        enabled: true")
			return nil
		}

		fmt.Printf("%-15s %-10s %-25s %s\n", "提供商", "状态", "默认模型", "可用模型")
		fmt.Println(strings.Repeat("─", 80))
		for _, p := range providers {
			status := "  "
			if p.Active {
				status = fmt.Sprintf("%s✓%s", banner.BrightGreen, banner.Reset)
			}
			models := strings.Join(p.Models, ", ")
			if len(models) > 30 {
				models = models[:27] + "..."
			}
			fmt.Printf("%-15s %s %-9s %-25s %s\n", p.Name, status, "", p.DefaultModel, models)
		}
		return nil
	},
}

var aiProviderSwitchCmd = &cobra.Command{
	Use:   "switch [name]",
	Short: "切换当前 AI 提供商",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		aiMgr, err := getAIManager()
		if err != nil {
			return err
		}
		if err := aiMgr.SetActive(args[0]); err != nil {
			return err
		}
		// 持久化到配置文件
		if err := config.SetActiveProvider(args[0]); err != nil {
			fmt.Printf("%s⚠ 已在内存中切换，但持久化到配置文件失败: %s%s\n", banner.Yellow, err.Error(), banner.Reset)
		}
		fmt.Printf("已切换到 AI 提供商: %s%s%s\n", banner.BrightGreen, args[0], banner.Reset)
		return nil
	},
}

// ===== ai agent =====

var aiAgentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Agent 智能体管理",
}

var aiAgentListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出可用的 Agent",
	RunE: func(cmd *cobra.Command, args []string) error {
		agentMgr := getAgentManager()
		agents := agentMgr.List()

		if len(agents) == 0 {
			fmt.Println("暂无可用 Agent")
			return nil
		}

		fmt.Printf("%-15s %-40s %-15s %s\n", "名称", "描述", "提供商", "技能")
		fmt.Println(strings.Repeat("─", 90))
		for _, a := range agents {
			provider := a.Provider
			if provider == "" {
				provider = "(默认)"
			}
			skills := "-"
			if len(a.Skills) > 0 {
				skills = strings.Join(a.Skills, ", ")
			}
			fmt.Printf("%-15s %-40s %-15s %s\n", a.Name, a.Description, provider, skills)
		}
		return nil
	},
}

var aiAgentRunCmd = &cobra.Command{
	Use:   "run [agent] [message]",
	Short: "运行指定 Agent",
	Long: `使用指定的 Agent 处理消息。

示例:
  jiasinecli ai agent run general "帮我总结这段文字"
  jiasinecli ai agent run explore "分析项目架构"`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		agentName := args[0]
		prompt := strings.Join(args[1:], " ")

		agentMgr := getAgentManager()
		resp, err := agentMgr.Run(agentName, prompt)
		if err != nil {
			return err
		}

		printAIResponse(resp)
		return nil
	},
}

var aiAgentInstallCmd = &cobra.Command{
	Use:   "install [path]",
	Short: "安装 Agent (从 JSON 文件或目录)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agentMgr := getAgentManager()
		if err := agentMgr.Install(args[0]); err != nil {
			return err
		}
		fmt.Printf("Agent 安装成功\n")
		return nil
	},
}

var aiAgentRemoveCmd = &cobra.Command{
	Use:   "remove [name]",
	Short: "卸载 Agent",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agentMgr := getAgentManager()
		if err := agentMgr.Remove(args[0]); err != nil {
			return err
		}
		fmt.Printf("Agent '%s' 已卸载\n", args[0])
		return nil
	},
}

// ===== ai skill =====

var aiSkillCmd = &cobra.Command{
	Use:   "skill",
	Short: "Skills 技能管理",
}

var aiSkillListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出可用的 Skills",
	RunE: func(cmd *cobra.Command, args []string) error {
		skillMgr := getSkillManager()
		skills := skillMgr.List()

		if len(skills) == 0 {
			fmt.Println("暂无可用 Skill")
			return nil
		}

		fmt.Printf("%-18s %-45s %-10s %s\n", "名称", "描述", "版本", "标签")
		fmt.Println(strings.Repeat("─", 95))
		for _, s := range skills {
			tags := "-"
			if len(s.Tags) > 0 {
				tags = strings.Join(s.Tags, ", ")
			}
			fmt.Printf("%-18s %-45s %-10s %s\n", s.Name, s.Description, s.Version, tags)
		}
		return nil
	},
}

var aiSkillInstallCmd = &cobra.Command{
	Use:   "install [name|path]",
	Short: "安装 Skill (按名称、JSON/MD 文件路径或目录)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		skillMgr := getSkillManager()
		if err := skillMgr.Install(args[0]); err != nil {
			return err
		}
		fmt.Printf("Skill 安装成功\n")
		return nil
	},
}

var aiSkillRemoveCmd = &cobra.Command{
	Use:   "remove [name]",
	Short: "卸载 Skill",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		skillMgr := getSkillManager()
		if err := skillMgr.Remove(args[0]); err != nil {
			return err
		}
		fmt.Printf("Skill '%s' 已卸载\n", args[0])
		return nil
	},
}

// ===== 辅助函数 =====

// ===== ai list =====

var aiListCmd = &cobra.Command{
	Use:   "list",
	Short: "查看可用 AI 模型列表",
	Long: `列出所有已配置且可用的 AI 模型。

显示每个服务商的名称、状态、默认模型等信息。
标记为 ✓ 的是当前激活的提供商。

示例:
  jiasinecli ai list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		aiMgr, err := getAIManager()
		if err != nil {
			return err
		}

		providers := aiMgr.ListProviders()
		if len(providers) == 0 {
			fmt.Println("未配置任何 AI 提供商")
			fmt.Printf("请在 %s~/.jiasine/config.yaml%s 中配置\n", banner.BrightGreen, banner.Reset)
			return nil
		}

		fmt.Printf("\n%s🤖 可用 AI 模型列表%s\n", banner.Bold+banner.BrightCyan, banner.Reset)
		fmt.Println(strings.Repeat("─", 70))
		fmt.Printf("  %-3s %-15s %-25s %s\n", "", "服务商", "默认模型", "可用模型")
		fmt.Println(strings.Repeat("─", 70))

		for i, p := range providers {
			status := fmt.Sprintf("  %s%d%s", banner.Dim, i+1, banner.Reset)
			if p.Active {
				status = fmt.Sprintf("%s✓ %d%s", banner.BrightGreen, i+1, banner.Reset)
			}
			models := strings.Join(p.Models, ", ")
			if len(models) > 25 {
				models = models[:22] + "..."
			}
			fmt.Printf("  %s %-15s %-25s %s\n", status, p.Name, p.DefaultModel, models)
		}

		fmt.Printf("\n%s提示%s: 使用 %sai connect%s 交互选择并连接模型\n\n",
			banner.Dim, banner.Reset,
			banner.BrightGreen, banner.Reset)
		return nil
	},
}

// ===== ai connect =====

var aiConnectCmd = &cobra.Command{
	Use:   "connect",
	Short: "选择 AI 模型并连接对话",
	Long: `交互式选择要连接的 AI 服务商，然后进入对话模式。

使用 ↑/↓ 箭头键选择，Enter 确认，Esc 取消。
选择后自动进入 AI 交互对话模式。

示例:
  jiasinecli ai connect`,
	RunE: func(cmd *cobra.Command, args []string) error {
		aiMgr, err := getAIManager()
		if err != nil {
			return err
		}

		providers := aiMgr.ListProviders()
		if len(providers) == 0 {
			return fmt.Errorf("未配置任何可用的 AI 提供商")
		}

		// 构建选择项
		options := make([]tui.SelectOption, len(providers))
		defaultIdx := 0
		for i, p := range providers {
			options[i] = tui.SelectOption{
				Label:       p.Name,
				Description: p.DefaultModel,
				Active:      p.Active,
			}
			if p.Active {
				defaultIdx = i
			}
		}

		// 交互式选择 (↑/↓ 箭头键)
		fmt.Println()
		idx, err := tui.Select("选择要连接的 AI 模型", options, defaultIdx)
		if err != nil {
			return nil // 用户取消
		}

		selected := providers[idx]
		if err := aiMgr.SetActive(selected.Key); err != nil {
			return fmt.Errorf("切换提供商失败: %w", err)
		}

		// 持久化到配置文件
		_ = config.SetActiveProvider(selected.Key)

		fmt.Printf("\n%s✓ 已连接 %s (%s)%s\n",
			banner.BrightGreen, selected.Name, selected.DefaultModel, banner.Reset)

		// 进入交互模式（默认使用 general agent）
		aiProvider = selected.Key
		return enterAIInteractive("general")
	},
}

// ===== 辅助函数 =====

// ── Box 绘制工具 ─────────────────────────────────────────────────────────────

// boxLine 表示 box 中的一行内容
type boxLine struct {
	text  string // 纯文本内容（用于计算显示宽度）
	ansi  string // 带 ANSI 颜色的完整内容（用于实际输出）
}

// rwCond 用于终端显示宽度计算的 runewidth 条件
// EastAsianWidth=true: 正确处理中文/日文/韩文环境下的全角/半角字符宽度
//   → U+00B7 (·) 等 Ambiguous 字符按 2 列计算，与 CJK 终端实际渲染一致
// StrictEmojiNeutral=false: Emoji 统一按宽字符 (2列) 计算
//   → ⚡🌐🧠🤖 等 Emoji 都按 2 列，与现代终端渲染一致
var rwCond = &runewidth.Condition{EastAsianWidth: true, StrictEmojiNeutral: false}

// displayWidth 计算字符串在 CJK 终端中的实际显示宽度
func displayWidth(s string) int {
	return rwCond.StringWidth(s)
}

// drawBox 绘制对齐的 Unicode Box，自动计算显示宽度
// borderColor: 边框颜色 ANSI 码
// lines: 内容行（纯文本 + ANSI 版本）
// separator: 在哪些行号后插入分隔线（从 0 开始）
func drawBox(borderColor string, lines []boxLine, separators []int) {
	r := banner.Reset

	// 计算最大显示宽度
	maxWidth := 0
	for _, l := range lines {
		w := displayWidth(l.text)
		if w > maxWidth {
			maxWidth = w
		}
	}

	// 两侧各留 2 个空格的 padding
	innerWidth := maxWidth + 4

	// 顶部边框
	fmt.Printf("%s╭%s╮%s\n", borderColor, strings.Repeat("─", innerWidth), r)

	sepSet := make(map[int]bool)
	for _, s := range separators {
		sepSet[s] = true
	}

	for i, l := range lines {
		textWidth := displayWidth(l.text)
		padRight := innerWidth - 2 - textWidth // 2 = left padding spaces
		if padRight < 0 {
			padRight = 0
		}
		fmt.Printf("%s│%s  %s%s%s│%s\n",
			borderColor, r,
			l.ansi, strings.Repeat(" ", padRight),
			borderColor, r)

		if sepSet[i] {
			fmt.Printf("%s├%s┤%s\n", borderColor, strings.Repeat("─", innerWidth), r)
		}
	}

	// 底部边框
	fmt.Printf("%s╰%s╯%s\n", borderColor, strings.Repeat("─", innerWidth), r)
}

func printAIResponse(resp *ai.ChatResponse) {
	// 顶部分隔线（类似 Copilot CLI）
	fmt.Printf("\n%s%s%s\n", banner.Gray, strings.Repeat("─", 80), banner.Reset)

	// AI 回复内容（Markdown 渲染）
	fmt.Println(render.Markdown(resp.Content))

	// 底部元信息（更优雅的展示）
	fmt.Printf("\n%s%s━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━%s\n",
		banner.Dim, banner.Gray, banner.Reset)
	fmt.Printf("%s  %s%s%s  •  %s%s%s  •  %stokens: %d%s\n",
		banner.Dim,
		banner.BrightCyan, resp.Provider, banner.Dim,
		banner.BrightBlue, resp.Model, banner.Dim,
		banner.LightGray, resp.TotalTokens, banner.Reset)
}

// EnterDefaultAIMode 默认 AI 模式入口（无参数启动时调用）
// 初始化终端 → 显示欢迎屏幕 → 尝试连接模型 → 进入 AI 交互对话
// 如果未配置模型，提示用户配置
func EnterDefaultAIMode() {
	// 初始化应用（配置、主题、日志、目录）
	if err := initializeApp(); err != nil {
		fmt.Fprintf(os.Stderr, "初始化失败: %v\n", err)
		return
	}

	// 初始化终端显示（Windows VT 处理）
	shell.InitTerminal()

	// 显示欢迎屏幕
	fmt.Print(banner.WelcomeScreen())

	// 进入 AI 交互模式
	if err := enterAIInteractive("general"); err != nil {
		// getAIManager() 已经打印了详细的配置引导信息
		// 这里只输出一个操作提示
		fmt.Printf("\n%s💡 配置完成后重新启动即可使用 AI 功能%s\n", banner.Dim, banner.Reset)
		fmt.Printf("%s   或运行 'jiasinecli ai connect' 交互选择模型%s\n\n", banner.Dim, banner.Reset)
	}
}

// enterAIInteractive 进入 AI 交互式对话模式
// 连接模型 → 显示欢迎 → REPL 循环 → Ctrl+C 退出
func enterAIInteractive(agentName string) error {
	// 1. 获取 AI 管理器（含自动配置生成）
	aiMgr, err := getAIManager()
	if err != nil {
		return err
	}

	// 1.5 初始化历史记录管理器
	historyMgr, historyErr := getHistoryManager()
	var sessionID string
	if historyErr != nil {
		fmt.Printf("%s⚠ 历史记录初始化失败: %s%s\n", banner.Yellow, historyErr.Error(), banner.Reset)
	}
	defer func() {
		if historyMgr != nil {
			// 退出时结束会话
			if sessionID != "" {
				_ = historyMgr.EndSession(sessionID)
			}
			historyMgr.Close()
		}
	}()

	// 2. 如果指定了 provider，切换
	if aiProvider != "" {
		if err := aiMgr.SetActive(aiProvider); err != nil {
			return err
		}
	}

	// 2.5 设置联网搜索（来自 --web 标志）
	if aiWebSearch {
		aiMgr.SetWebSearch(true)
	}

	// 3. 连接验证 — 发送一条极短的测试请求（不带 web search / tools，保持轻量）
	providerName, modelName := aiMgr.ActiveProviderInfo()
	fmt.Printf("\n%s正在连接 %s (%s) ...%s", banner.Dim, providerName, modelName, banner.Reset)

	testResp, testErr := aiMgr.TestConnection()
	if testErr != nil {
		fmt.Printf(" %s失败%s\n", banner.Yellow, banner.Reset)
		return fmt.Errorf("模型连接失败: %w", testErr)
	}
	_ = testResp
	fmt.Printf(" %s✓ 已连接%s\n", banner.BrightGreen, banner.Reset)

	// 3.5 创建历史会话
	if historyMgr != nil {
		session := &historydb.Session{
			AgentName: agentName,
			Provider:  providerName,
			Model:     modelName,
		}
		if err := historyMgr.CreateSession(session); err != nil {
			fmt.Printf("%s⚠ 创建历史会话失败: %s%s\n", banner.Yellow, err.Error(), banner.Reset)
		} else {
			sessionID = session.ID
		}
	}

	// 4. 显示欢迎信息
	webStatus := "关闭"
	if aiMgr.IsWebSearch() {
		webStatus = "🌐 开启"
	}

	// 4.1 初始化记忆系统
	memMgr, memErr := ai.NewMemoryManager()
	if memErr != nil {
		fmt.Printf("%s⚠ 记忆系统初始化失败: %s%s\n", banner.Yellow, memErr.Error(), banner.Reset)
	}

	// 记忆统计
	memStatus := "关闭"
	if memMgr != nil {
		sessions, activeMem, _ := memMgr.Stats()
		if activeMem > 0 || sessions > 0 {
			memStatus = fmt.Sprintf("🧠 %d 条记忆 · %d 次会话", activeMem, sessions)
		} else {
			memStatus = "🧠 就绪"
		}
	}

	fmt.Println()
	// 构建 welcome box 内容
	welcomeLines := []boxLine{
		{
			text: "🤖 AI 交互模式              ⚡ 流式输出",
			ansi: fmt.Sprintf("🤖 AI 交互模式              %s⚡ 流式输出%s", banner.Dim, banner.Reset),
		},
		{
			text: "服务商: " + providerName,
			ansi: fmt.Sprintf("%s服务商: %s%s", banner.Reset, banner.BrightGreen+providerName, banner.Reset),
		},
		{
			text: "模型:   " + modelName,
			ansi: fmt.Sprintf("%s模型:   %s%s", banner.Reset, banner.BrightGreen+modelName, banner.Reset),
		},
	}
	if agentName != "" {
		welcomeLines = append(welcomeLines, boxLine{
			text: "Agent:  " + agentName,
			ansi: fmt.Sprintf("%sAgent:  %s%s", banner.Reset, banner.BrightCyan+agentName, banner.Reset),
		})
	}
	welcomeLines = append(welcomeLines,
		boxLine{
			text: "联网:   " + webStatus,
			ansi: fmt.Sprintf("%s联网:   %s%s", banner.Reset, banner.BrightCyan+webStatus, banner.Reset),
		},
		boxLine{
			text: "记忆:   " + memStatus,
			ansi: fmt.Sprintf("%s记忆:   %s%s", banner.Reset, banner.BrightCyan+memStatus, banner.Reset),
		},
		boxLine{
			text: "输入 /help 查看命令，Ctrl+C 退出",
			ansi: fmt.Sprintf("%s输入 /help 查看命令，Ctrl+C 退出%s", banner.Dim, banner.Reset),
		},
	)
	drawBox(banner.Cyan, welcomeLines, nil)
	fmt.Println()

	// 5. 保存对话历史（上下文）
	history := []ai.Message{}

	// 如果使用 Agent，加入系统提示词 + 收集 MCP 工具
	var agentSystem string
	var mcpTools []map[string]interface{}
	if agentName != "" {
		agentMgr := getAgentManager()
		system, err := agentMgr.GetSystemPrompt(agentName)
		if err != nil {
			fmt.Printf("%s⚠ Agent '%s' 未找到，将使用普通对话模式%s\n\n", banner.Yellow, agentName, banner.Reset)
		} else {
			agentSystem = system
		}

		// 注入长期记忆 + 上次会话摘要到系统提示词
		if memMgr != nil {
			memContext := memMgr.BuildMemoryContext()
			if memContext != "" {
				agentSystem += memContext
			}
			sessionSummary := memMgr.BuildSessionSummary()
			if sessionSummary != "" {
				agentSystem += sessionSummary
			}
		}

		if agentSystem != "" {
			history = append(history, ai.Message{Role: ai.RoleSystem, Content: agentSystem})
		}

		// 收集 Agent 关联的 MCP 工具定义
		skillMgr := getSkillManager()
		agents := agentMgr.List()
		for _, a := range agents {
			if a.Name == agentName && len(a.Skills) > 0 {
				var skills []*ai.Skill
				for _, sn := range a.Skills {
					if s, err := skillMgr.Get(sn); err == nil {
						skills = append(skills, s)
					}
				}
				mcpTools = ai.MCPToolDefs(skills)
				break
			}
		}
	}

	// MCP 工具执行器
	executor := ai.NewToolExecutor()

	// 5.5 记忆系统: 启动会话 + 恢复上次对话
	if memMgr != nil {
		memMgr.StartSession(agentName, providerName, modelName)

		// 尝试恢复最近 24h 内的会话消息
		restored := memMgr.RestoreLastMessages()
		if len(restored) > 0 {
			// 还原对话消息到 history (追加在 system prompt 之后)
			for _, msg := range restored {
				history = append(history, msg)
			}
			turns := 0
			for _, m := range restored {
				if m.Role == ai.RoleUser {
					turns++
				}
			}
			fmt.Printf("%s🧠 已恢复上次对话 (%d 轮)%s\n\n", banner.Dim, turns, banner.Reset)

			// 显示最近的对话内容（最多显示最近 5 条 user/assistant 消息）
			printRestoredMessages(restored)
		}

		// 运行自动衰减
		memMgr.AutoDecay()
	}

	// 记忆开关（可通过 /mem on  /mem off 切换）
	memEnabled := (memMgr != nil)

	// 6. REPL 循环
	// 每轮使用 tui.ReadLine 读取一行（支持 Tab 补全、方向键编辑、/ 命令智能提示）
	// ReadLine 返回后不再占用 stdin，TUI 组件可以安全使用 raw 模式
	type inputLine struct {
		text string
		ok   bool
	}

	// 捕获 Ctrl+C — 防止进程直接终止
	sigChan := make(chan os.Signal, 2)
	signal.Notify(sigChan, os.Interrupt)

	for {
		// ReadLine 自带提示符 + Tab 补全 + / 命令智能提示
		fmt.Print("\n")
		inputCh := make(chan inputLine, 1)
		go func() {
			text, ok := tui.ReadLine(aiMgr.IsWebSearch())
			inputCh <- inputLine{text: strings.TrimSpace(text), ok: ok}
		}()

		// 同时等待用户输入和 Ctrl+C 信号
		var input string
		exit := false
		select {
		case <-sigChan:
			fmt.Printf("\n\n%s👋 已退出 AI 模式%s\n\n", banner.Dim, banner.Reset)
			exit = true
		case res := <-inputCh:
			if !res.ok {
				// EOF — 退出
				exit = true
			} else {
				input = res.text
			}
		}
		if exit {
			break
		}

		if input == "" {
			continue
		}

		// 斜杠命令 — 以 / 开头的输入
		if strings.HasPrefix(input, "/") {
			slashCmd := strings.ToLower(strings.TrimSpace(input[1:]))

			if slashCmd == "exit" || slashCmd == "quit" || slashCmd == "bye" {
				fmt.Printf("\n%s👋 已退出 AI 模式%s\n\n", banner.Dim, banner.Reset)
				break
			}
			if slashCmd == "clear" || slashCmd == "reset" {
				history = history[:0]
				if agentSystem != "" {
					history = append(history, ai.Message{Role: ai.RoleSystem, Content: agentSystem})
				}
				if memMgr != nil {
					_ = memMgr.SaveSession()
					memMgr.StartSession(agentName, providerName, modelName)
				}
				fmt.Printf("%s对话历史已清空 (新会话)%s\n\n", banner.Dim, banner.Reset)
				continue
			}
			if slashCmd == "memory on" || slashCmd == "mem on" {
				if memMgr == nil {
					fmt.Printf("%s记忆系统未初始化%s\n\n", banner.Yellow, banner.Reset)
				} else {
					memEnabled = true
					fmt.Printf("%s🧠 记忆系统已开启%s\n\n", banner.BrightGreen, banner.Reset)
				}
				continue
			}
			if slashCmd == "memory off" || slashCmd == "mem off" {
				memEnabled = false
				fmt.Printf("%s📴 记忆系统已关闭%s\n\n", banner.Dim, banner.Reset)
				continue
			}
			if slashCmd == "memory" || slashCmd == "mem" {
				if memMgr == nil {
					fmt.Printf("%s记忆系统未初始化%s\n\n", banner.Yellow, banner.Reset)
				} else if !memEnabled {
					fmt.Printf("%s📴 记忆系统已关闭 (使用 /mem on 开启)%s\n\n", banner.Dim, banner.Reset)
				} else {
					printMemoryStatus(memMgr)
				}
				continue
			}
			if slashCmd == "memory clear" || slashCmd == "mem clear" {
				if memMgr != nil {
					_ = memMgr.ClearAllMemories()
					fmt.Printf("%s🗑 已清空所有记忆%s\n\n", banner.Dim, banner.Reset)
				}
				continue
			}
			if slashCmd == "web" || slashCmd == "search" {
				enabled := aiMgr.ToggleWebSearch()
				_ = config.SetWebSearch(enabled)
				if enabled {
					fmt.Printf("%s🌐 联网搜索已开启 (已保存到配置)%s\n\n", banner.BrightGreen, banner.Reset)
				} else {
					fmt.Printf("%s📴 联网搜索已关闭 (已保存到配置)%s\n\n", banner.Dim, banner.Reset)
				}
				continue
			}
			if slashCmd == "theme" {
				handleThemeCommand()
				continue
			}
			if slashCmd == "skills" {
				handleSkillsCommand()
				continue
			}
			if slashCmd == "model" || slashCmd == "connect" {
				oldProvider := providerName
				handleModelCommand(aiMgr)
				newProvider, newModel := aiMgr.ActiveProviderInfo()
				if newProvider != oldProvider {
					providerName, modelName = newProvider, newModel
					// 切换模型后清空历史（避免跨模型上下文混淆）
					history = history[:0]
					if agentSystem != "" {
						history = append(history, ai.Message{Role: ai.RoleSystem, Content: agentSystem})
					}
					if memMgr != nil && memEnabled {
						_ = memMgr.SaveSession()
						memMgr.StartSession(agentName, providerName, modelName)
					}
					if historyMgr != nil && sessionID != "" {
						_ = historyMgr.EndSession(sessionID)
						session := &historydb.Session{AgentName: agentName, Provider: providerName, Model: modelName}
						if err := historyMgr.CreateSession(session); err == nil {
							sessionID = session.ID
						}
					}
					fmt.Printf("%s📝 对话历史已清空 (新模型新会话)%s\n", banner.Dim, banner.Reset)
				} else {
					providerName, modelName = newProvider, newModel
				}
				continue
			}
			if slashCmd == "status" {
				handleStatusCommand(aiMgr, agentName, memMgr, history)
				continue
			}
			if slashCmd == "new" {
				history = history[:0]
				if agentSystem != "" {
					history = append(history, ai.Message{Role: ai.RoleSystem, Content: agentSystem})
				}
				if memMgr != nil {
					_ = memMgr.SaveSession()
					memMgr.StartSession(agentName, providerName, modelName)
				}
				if historyMgr != nil && sessionID != "" {
					_ = historyMgr.EndSession(sessionID)
					session := &historydb.Session{AgentName: agentName, Provider: providerName, Model: modelName}
					if err := historyMgr.CreateSession(session); err == nil {
						sessionID = session.ID
					}
				}
				fmt.Printf("%s✨ 新会话已开始%s\n\n", banner.BrightGreen, banner.Reset)
				continue
			}
			if slashCmd == "history" {
				handleHistoryInChat(historyMgr)
				continue
			}
			if slashCmd == "help" {
				printAIChatHelp()
				continue
			}
			if slashCmd == "compact" {
				// 压缩上下文: 保留 system prompt + 最近 10 条消息
				before := len(history)
				keepN := 10
				if agentSystem != "" {
					if len(history) > keepN+1 {
						history = append(history[:1], history[len(history)-keepN:]...)
					}
				} else {
					if len(history) > keepN {
						history = history[len(history)-keepN:]
					}
				}
				after := len(history)
				fmt.Printf("%s📦 已压缩上下文: %d → %d 条消息%s\n\n", banner.BrightGreen, before, after, banner.Reset)
				continue
			}
			if slashCmd == "context" {
				// 显示上下文使用情况
				totalMsgs := len(history)
				userMsgs := 0
				assistantMsgs := 0
				systemMsgs := 0
				totalChars := 0
				for _, m := range history {
					totalChars += len(m.Content)
					switch m.Role {
					case ai.RoleUser:
						userMsgs++
					case ai.RoleAssistant:
						assistantMsgs++
					case ai.RoleSystem:
						systemMsgs++
					}
				}
				fmt.Println()
				fmt.Printf("%s📊 上下文使用情况%s\n", banner.BrightCyan, banner.Reset)
				fmt.Println(strings.Repeat("─", 40))
				fmt.Printf("  总消息数:   %d\n", totalMsgs)
				fmt.Printf("  系统提示:   %d\n", systemMsgs)
				fmt.Printf("  用户消息:   %d\n", userMsgs)
				fmt.Printf("  AI 回复:    %d\n", assistantMsgs)
				fmt.Printf("  总字符数:   %d\n", totalChars)
				fmt.Printf("  预估 tokens: ~%d\n", totalChars/3)
				fmt.Println()
				continue
			}
			if slashCmd == "agent" {
				handleAgentCommand()
				continue
			}

			// 尝试作为 CLI 命令执行（如 /version, /setup, /update, /plugin 等）
			cliArgs := strings.Fields(slashCmd)
			if err := ExecuteArgs(cliArgs); err == nil {
				fmt.Println()
				continue
			}
			// CLI 命令也无法识别
			fmt.Printf("  %s未知命令: /%s，输入 /help 查看可用命令%s\n\n",
				banner.Bold+banner.BrightCyan, slashCmd, banner.Reset)
			continue
		}

		// 非斜杠输入 → 发送给 AI 对话
		history = append(history, ai.Message{Role: ai.RoleUser, Content: input})

		// 保存用户消息到历史数据库
		if historyMgr != nil && sessionID != "" {
			userMsg := &historydb.Message{
				SessionID: sessionID,
				Role:      "user",
				Content:   input,
				Timestamp: time.Now(),
			}
			if err := historyMgr.SaveMessage(userMsg); err != nil {
				// 静默失败，不打断对话
				_ = err
			}

			// 如果这是第一条用户消息，更新会话标题
			session, _ := historyMgr.GetSession(sessionID)
			if session != nil && session.MessageCount == 1 && session.Title == "" {
				// 使用第一条用户消息作为会话标题
				_ = historyMgr.UpdateSessionTitle(sessionID, input)
			}
		}

		// 显示 AI 回复开始标记（类似 Copilot CLI）
		fmt.Printf("\n%s%s━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━%s\n",
			banner.Gray, banner.Dim, banner.Reset)
		fmt.Printf("%s  🤖 %s%sAI 回复%s\n",
			banner.Dim, banner.BrightCyan, banner.Bold, banner.Reset)
		fmt.Printf("%s%s━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━%s\n\n",
			banner.Gray, banner.Dim, banner.Reset)

		// ===== 流式输出 + 工具调用循环（最多 10 轮）=====
		var fullContent string    // 累积的完整回复文本
		var totalUsage ai.TokenUsage
		toolLoopErr := false
		maxToolLoops := 10

		for toolIter := 0; toolIter < maxToolLoops; toolIter++ {
			streamCh, streamErr := aiMgr.ChatMessagesWithToolsStream("", history, mcpTools)
			if streamErr != nil {
				fmt.Printf("\r\033[K") // 清除加载指示器
				fmt.Printf("%s错误: %s%s\n\n", banner.Yellow, streamErr.Error(), banner.Reset)
				history = history[:len(history)-1]
				toolLoopErr = true
				break
			}

			// 消费流式响应
			var iterContent string     // 本轮累积文本
			var iterThinking string    // 本轮思考内容
			var iterToolCalls []ai.ToolCall
			var iterStopReason string
			isThinking := false        // 当前是否在输出思考内容
			hasContent := false        // 是否已开始输出正文
			firstChunk := true         // 是否是第一个有效 chunk

			for chunk := range streamCh {
				if chunk.Error != nil {
					fmt.Printf("%s错误: %s%s\n\n", banner.Yellow, chunk.Error.Error(), banner.Reset)
					if toolIter == 0 && fullContent == "" {
						history = history[:len(history)-1]
					}
					toolLoopErr = true
					break
				}

				// 收到第一个 chunk 时显示流式输出标记
				if firstChunk && (chunk.Type == "thinking" || chunk.Type == "content" || chunk.Type == "tool_use") {
					firstChunk = false
				}

				switch chunk.Type {
				case "thinking":
					if !isThinking {
						// 开始思考 — 显示思考区域头部
						isThinking = true
						fmt.Printf("%s%s💭 思考中...%s\n", banner.Dim, banner.Italic, banner.Reset)
						fmt.Printf("%s", banner.Dim+banner.Italic)
					}
					fmt.Print(chunk.Thinking)
					iterThinking += chunk.Thinking

				case "content":
					if isThinking {
						isThinking = false
						fmt.Printf("%s\n", banner.Reset)
						fmt.Printf("%s%s─ 思考完毕 ─%s\n\n", banner.Dim, banner.Italic, banner.Reset)
					}
					if !hasContent {
						hasContent = true
						fmt.Printf("%s✈ 生成中...%s", banner.Dim, banner.Reset)
					}
					iterContent += chunk.Content

				case "tool_use":
					if isThinking {
						isThinking = false
						fmt.Printf("%s\n", banner.Reset)
					}
					iterToolCalls = append(iterToolCalls, chunk.ToolCalls...)

				case "usage":
					iterStopReason = chunk.StopReason
					if chunk.Usage != nil {
						totalUsage.PromptTokens += chunk.Usage.PromptTokens
						totalUsage.OutputTokens += chunk.Usage.OutputTokens
						totalUsage.TotalTokens += chunk.Usage.TotalTokens
					}
				}
			}

			if toolLoopErr {
				break
			}

			// 如果思考标记未关闭，关闭它
			if isThinking {
				fmt.Printf("%s\n", banner.Reset)
			}

			fullContent = iterContent

			// 清除“生成中”提示，输出 Markdown 渲染结果
			if hasContent && iterContent != "" {
				fmt.Print("\r\033[2K")
				formatted := render.Markdown(iterContent)
				fmt.Println(formatted)
			}

			// 如果没有工具调用 → 完成
			if len(iterToolCalls) == 0 || iterStopReason != "tool_use" {
				break
			}

			// ---- 执行工具调用 ----
			if hasContent {
				fmt.Println()
			}

			// 将助手的 tool_use 响应存入历史
			assistantContent := ai.BuildAssistantToolUseBlocks(iterContent, iterToolCalls)
			history = append(history, ai.Message{
				Role:    ai.RoleAssistantToolUse,
				Content: assistantContent,
			})

			// 逐个执行工具
			var toolResults []map[string]interface{}
			for _, call := range iterToolCalls {
				fmt.Printf("%s🔧 执行工具: %s%s%s\n", banner.Dim, banner.BrightCyan, call.Name, banner.Reset)
				result := executor.Execute(call)
				if result.IsError {
					fmt.Printf("%s   ⚠ 工具执行出错%s\n", banner.Yellow, banner.Reset)
				} else {
					preview := result.Content
					if len(preview) > 200 {
						preview = preview[:200] + "..."
					}
					fmt.Printf("%s   ✓ 结果: %s%s\n", banner.Dim, preview, banner.Reset)
				}
				toolResults = append(toolResults, map[string]interface{}{
					"type":        "tool_result",
					"tool_use_id": result.ToolUseID,
					"content":     result.Content,
				})
			}

			// 将工具结果作为 user 消息发回
			toolResultJSON, _ := json.Marshal(toolResults)
			history = append(history, ai.Message{
				Role:    ai.RoleToolResult,
				Content: string(toolResultJSON),
			})

			// 继续循环，让 AI 基于工具结果生成最终回复 (下一轮会再次流式输出)
		}

		if toolLoopErr {
			continue
		}

		// 显示底部分隔线和元信息（类似 Copilot CLI）
		webLabel := ""
		if aiMgr.IsWebSearch() {
			webLabel = " · 🌐 联网"
		}
		fmt.Printf("\n%s%s━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━%s\n",
			banner.Gray, banner.Dim, banner.Reset)
		fmt.Printf("%s  %s%s%s  •  %s%s%s  •  %stokens: %d%s%s\n",
			banner.Dim,
			banner.BrightCyan, providerName, banner.Dim,
			banner.BrightBlue, modelName, banner.Dim,
			banner.LightGray, totalUsage.TotalTokens, webLabel, banner.Reset)

		// 加助手回复到历史
		history = append(history, ai.Message{Role: ai.RoleAssistant, Content: fullContent})

		// 保存助手消息到历史数据库
		if historyMgr != nil && sessionID != "" {
			assistantMsg := &historydb.Message{
				SessionID: sessionID,
				Role:      "assistant",
				Content:   fullContent,
				Timestamp: time.Now(),
				Tokens:    totalUsage.TotalTokens,
			}
			if err := historyMgr.SaveMessage(assistantMsg); err != nil {
				// 静默失败，不打断对话
				_ = err
			}
		}

		// 记忆系统: 追踪每轮对话的 user + assistant 消息
		if memMgr != nil && memEnabled {
			// 追加用户消息（前面已加到 history，这里追加到记忆会话）
			memMgr.AppendMessage(ai.Message{Role: ai.RoleUser, Content: input})
			memMgr.AppendMessage(ai.Message{Role: ai.RoleAssistant, Content: fullContent})
			// 每轮自动保存一次会话
			if memMgr != nil {
				_ = memMgr.SaveSession()
			}
		}

		// 防止历史过长（保留最近 50 条对话）
		maxHistory := 50
		if agentSystem != "" {
			maxHistory = 51 // system prompt + 50 条
		}
		if len(history) > maxHistory {
			if agentSystem != "" {
				// 保留 system prompt + 最近 N 条
				history = append(history[:1], history[len(history)-50:]...)
			} else {
				history = history[len(history)-50:]
			}
		}
	}

	// ===== 退出清理: 保存会话 + 提取长期记忆 =====
	if memMgr != nil && memEnabled {
		// 保存当前会话
		_ = memMgr.SaveSession()

		// 提取长期记忆: 用 AI 分析本次对话提取值得记住的信息
		// 只在有足够对话量时触发 (至少 2 轮)
		userMsgCount := 0
		for _, msg := range history {
			if msg.Role == ai.RoleUser {
				userMsgCount++
			}
		}
		if userMsgCount >= 2 {
			fmt.Printf("%s🧠 正在提取记忆...%s", banner.Dim, banner.Reset)
			done := make(chan struct{})
			go func() {
				extractAndSaveMemories(aiMgr, memMgr, history)
				close(done)
			}()
			// 最多等 15 秒让提取完成
			select {
			case <-done:
				fmt.Printf(" %s✓ 完成%s\n", banner.BrightGreen, banner.Reset)
			case <-time.After(15 * time.Second):
				fmt.Printf(" %s(超时，后台继续)%s\n", banner.Dim, banner.Reset)
			}
		}

		// 清理旧会话 (保留最近 30 个)
		memMgr.CleanOldSessions(30)
	}

	// 清理: 排空残留信号后再恢复默认行为，防止 Windows 上
	// 残留的 CTRL_C_EVENT 在 signal.Stop 后触发默认终止
	for {
		select {
		case <-sigChan:
			continue
		default:
		}
		break
	}
	signal.Stop(sigChan)
	return nil
}

// printRestoredMessages 显示恢复的会话历史（最多显示最近 5 条 user/assistant 消息）
func printRestoredMessages(messages []ai.Message) {
	// 过滤出 user/assistant 消息（跳过 system, tool_result, assistant_tool_use）
	type displayMsg struct {
		role    ai.Role
		content string
	}
	var visible []displayMsg
	for _, m := range messages {
		if m.Role == ai.RoleUser || m.Role == ai.RoleAssistant {
			visible = append(visible, displayMsg{role: m.Role, content: m.Content})
		}
	}
	if len(visible) == 0 {
		return
	}

	// 如果超过 5 条，只显示最后 5 条
	showCount := len(visible)
	startIdx := 0
	if showCount > 5 {
		startIdx = showCount - 5
		fmt.Printf("%s  ... 省略了 %d 条更早的对话 ...%s\n\n", banner.Dim, startIdx, banner.Reset)
	}

	for i := startIdx; i < showCount; i++ {
		msg := visible[i]
		content := msg.content
		// 截断过长消息
		lines := strings.Split(content, "\n")
		if len(lines) > 6 {
			content = strings.Join(lines[:5], "\n") + fmt.Sprintf("\n%s  ... (%d 行已省略)%s", banner.Dim, len(lines)-5, banner.Reset)
		} else if len(content) > 300 {
			content = content[:297] + "..."
		}

		if msg.role == ai.RoleUser {
			fmt.Printf("  %s👤 你:%s %s\n", banner.BrightCyan, banner.Reset, content)
		} else {
			fmt.Printf("  %s🤖 AI:%s %s\n", banner.BrightGreen, banner.Reset, content)
		}
	}
	fmt.Println()
}

func printAIChatHelp() {
	g := banner.BrightGreen
	c := banner.BrightCyan
	d := banner.Dim
	r := banner.Reset

	fmt.Printf(`
%s AI 交互模式命令:%s

  %s对话%s
  直接输入文字         与 AI 对话 (流式输出 · 支持多轮上下文)
  %s/new%s                 开始新会话 (清空上下文)
  %s/clear%s / %s/reset%s       清空对话历史

  %s设置%s
  %s/theme%s               切换主题 (暗色/亮色)
  %s/model%s / %s/connect%s     切换 AI 模型/提供商
  %s/agent%s               选择 Agent 智能体
  %s/skills%s              管理 Skills (启用/禁用)
  %s/web%s / %s/search%s        切换联网搜索 (开/关)

  %s上下文 & 记忆%s
  %s/context%s              查看上下文使用情况
  %s/compact%s              压缩对话上下文
  %s/memory%s / %s/mem%s        查看记忆状态
  %s/mem on%s / %s/mem off%s    开/关记忆追踪
  %s/mem clear%s            清空所有记忆
  %s/history%s              查看最近会话

  %s其他%s
  %s/status%s               查看当前状态
  %s/help%s                 显示此帮助
  %s/exit%s / %s/quit%s         退出 AI 模式
  %sCtrl+C%s               退出 AI 模式

  %sCLI 命令 (直接用 / 前缀调用)%s
  %s/version%s              查看版本信息
  %s/setup%s                配置系统环境 (PATH)
  %s/update%s               检查更新
  %s/plugin list%s          查看已安装插件
  %s/test status%s          查看测试工具链状态
  %s/bridge list%s          查看桥接层列表
  %s/service health%s       服务健康检查

%s提示:%s AI 回复采用流式输出 (打字机效果)
      支持思考模型的推理过程可视化 (灰色斜体显示)
      输入 %s-i%s 或 %s--interactive%s 可进入传统命令 Shell

`,
		c, r,
		c, r,
		g, r,
		g, r, g, r,
		c, r,
		g, r,
		g, r, g, r,
		g, r,
		g, r,
		g, r, g, r,
		c, r,
		g, r,
		g, r,
		g, r, g, r,
		g, r, g, r,
		g, r,
		g, r,
		c, r,
		g, r,
		g, r,
		g, r, g, r,
		g, r,
		c, r,
		g, r,
		g, r,
		g, r,
		g, r,
		g, r,
		g, r,
		g, r,
		d, r,
		g, r, g, r,
	)
}

// handleAgentCommand 处理 /agent 命令 — 交互式 Agent 选择
func handleAgentCommand() {
	agentMgr := getAgentManager()
	agents := agentMgr.List()
	if len(agents) == 0 {
		fmt.Printf("%s暂无可用 Agent%s\n\n", banner.Dim, banner.Reset)
		return
	}

	options := make([]tui.SelectOption, len(agents))
	for i, a := range agents {
		options[i] = tui.SelectOption{
			Label:       a.Name,
			Description: a.Description,
		}
	}

	idx, err := tui.Select("选择 Agent", options, 0)
	if err != nil || idx < 0 {
		return
	}

	selected := agents[idx]
	fmt.Printf("%s✓ 已选择 Agent: %s%s\n\n", banner.BrightGreen, selected.Name, banner.Reset)
}

// handleThemeCommand 处理 /theme 命令 — 用 bubbletea 交互式主题切换
func handleThemeCommand() {
	currentKey := string(theme.CurrentName())

	// 使用 bubbletea 丰富的主题选择器
	selectedKey := tui.PickTheme(currentKey)
	if selectedKey == "" {
		return // 用户取消
	}

	theme.Set(theme.ThemeName(selectedKey))
	banner.RefreshColors()
	_ = config.SetTheme(selectedKey)

	label := "🌙 暗色主题"
	if selectedKey == "light" {
		label = "☀️ 亮色主题"
	}
	fmt.Printf("\n%s✓ 已切换为 %s (已保存)%s\n\n", banner.BrightGreen, label, banner.Reset)
}

// handleSkillsCommand 处理 /skills 命令 — 交互式 Skill 管理
func handleSkillsCommand() {
	skillMgr := getSkillManager()
	skills := skillMgr.List()
	if len(skills) == 0 {
		fmt.Printf("%s暂无可用 Skills%s\n\n", banner.Dim, banner.Reset)
		return
	}

	options := make([]tui.MultiSelectOption, len(skills))
	for i, s := range skills {
		desc := s.Description
		if s.Version != "" {
			desc += " v" + s.Version
		}
		options[i] = tui.MultiSelectOption{
			Label:       s.Name,
			Description: desc,
			Checked:     true, // 默认全部启用
		}
	}

	checked, err := tui.MultiSelect("选择要启用的 Skills", options)
	if err != nil {
		return
	}

	enabled := 0
	disabled := 0
	for i, c := range checked {
		if c {
			enabled++
		} else {
			disabled++
		}
		_ = i
	}

	fmt.Printf("%s✓ 已启用 %d 个 Skills, 已禁用 %d 个%s\n\n",
		banner.BrightGreen, enabled, disabled, banner.Reset)
}

// handleModelCommand 处理 /model 命令 — 交互式模型切换
func handleModelCommand(aiMgr *ai.Manager) {
	providers := aiMgr.ListProviders()
	if len(providers) == 0 {
		fmt.Printf("%s暂无可用模型%s\n\n", banner.Dim, banner.Reset)
		return
	}

	options := make([]tui.SelectOption, len(providers))
	activeIdx := 0
	for i, p := range providers {
		options[i] = tui.SelectOption{
			Label:       p.Name,
			Description: p.DefaultModel,
			Active:      p.Active,
		}
		if p.Active {
			activeIdx = i
		}
	}

	idx, err := tui.Select("切换 AI 模型", options, activeIdx)
	if err != nil || idx < 0 {
		return
	}

	selected := providers[idx]
	if err := aiMgr.SetActive(selected.Key); err != nil {
		fmt.Printf("%s✗ 切换失败: %s%s\n\n", banner.BrightRed, err, banner.Reset)
		return
	}

	_ = config.SetActiveProvider(selected.Key)
	fmt.Printf("%s✓ 已切换到 %s (%s)%s\n\n", banner.BrightGreen, selected.Name, selected.DefaultModel, banner.Reset)
}

// handleStatusCommand 处理 /status 命令 — 显示当前状态
func handleStatusCommand(aiMgr *ai.Manager, agentName string, memMgr *ai.MemoryManager, history []ai.Message) {
	providerName, modelName := aiMgr.ActiveProviderInfo()

	webStatus := "关闭"
	if aiMgr.IsWebSearch() {
		webStatus = "🌐 开启"
	}

	memStatus := "关闭"
	if memMgr != nil {
		_, activeMem, _ := memMgr.Stats()
		memStatus = fmt.Sprintf("🧠 %d 条", activeMem)
	}

	turns := 0
	for _, m := range history {
		if m.Role == ai.RoleUser {
			turns++
		}
	}

	themeName := theme.CurrentName()
	turnsStr := fmt.Sprintf("%d", turns)

	statusLines := []boxLine{
		{
			text: "📊 当前状态",
			ansi: fmt.Sprintf("%s📊 当前状态%s", banner.BrightCyan, banner.Reset),
		},
		{
			text: "服务商:   " + providerName,
			ansi: fmt.Sprintf("服务商:   %s", providerName),
		},
		{
			text: "模型:     " + modelName,
			ansi: fmt.Sprintf("模型:     %s", modelName),
		},
	}
	if agentName != "" {
		statusLines = append(statusLines, boxLine{
			text: "Agent:    " + agentName,
			ansi: fmt.Sprintf("Agent:    %s", agentName),
		})
	}
	statusLines = append(statusLines,
		boxLine{text: "联网:     " + webStatus, ansi: "联网:     " + webStatus},
		boxLine{text: "记忆:     " + memStatus, ansi: "记忆:     " + memStatus},
		boxLine{text: "对话轮次: " + turnsStr, ansi: "对话轮次: " + turnsStr},
		boxLine{text: "主题:     " + string(themeName), ansi: "主题:     " + string(themeName)},
	)

	fmt.Println()
	drawBox(banner.Cyan, statusLines, []int{0})
	fmt.Println()
}

// handleHistoryInChat 在聊天模式内查看最近会话
func handleHistoryInChat(historyMgr *historydb.Manager) {
	if historyMgr == nil {
		fmt.Printf("%s历史记录未初始化%s\n\n", banner.Yellow, banner.Reset)
		return
	}

	sessions, err := historyMgr.ListSessions(historydb.Query{Limit: 10})
	if err != nil || len(sessions) == 0 {
		fmt.Printf("%s暂无历史会话%s\n\n", banner.Dim, banner.Reset)
		return
	}

	fmt.Printf("\n%s📋 最近会话:%s\n", banner.BrightCyan, banner.Reset)
	fmt.Println(strings.Repeat("─", 55))
	for _, s := range sessions {
		title := s.Title
		if title == "" {
			title = "(无标题)"
		}
		if len(title) > 35 {
			title = title[:32] + "..."
		}
		fmt.Printf("  %s%-8s%s  %s%-35s%s  %s%d 条%s\n",
			banner.Dim, s.StartedAt.Format("01-02 15:04"), banner.Reset,
			banner.Reset, title, banner.Reset,
			banner.Dim, s.MessageCount, banner.Reset,
		)
	}
	fmt.Println()
}

// printMemoryStatus 显示记忆系统状态
func printMemoryStatus(memMgr *ai.MemoryManager) {
	sessions, activeMem, expiredMem := memMgr.Stats()
	fmt.Printf("\n%s🧠 记忆系统状态%s\n", banner.BrightCyan, banner.Reset)
	fmt.Println(strings.Repeat("─", 45))
	fmt.Printf("  会话记录:   %d 次\n", sessions)
	fmt.Printf("  活跃记忆:   %d 条\n", activeMem)
	fmt.Printf("  已遗忘:     %d 条\n", expiredMem)
	fmt.Printf("  存储位置:   %s\n", memMgr.GetMemDir())

	// 列出活跃记忆
	memories := memMgr.GetAllMemories()
	if len(memories) > 0 {
		fmt.Printf("\n%s活跃记忆:%s\n", banner.Dim, banner.Reset)
		categories := map[string]string{
			"user_info":   "👤",
			"preference":  "⚙️",
			"project":     "📁",
			"fact":        "📌",
			"instruction": "📝",
		}
		for i, e := range memories {
			icon := categories[e.Category]
			if icon == "" {
				icon = "💡"
			}
			content := e.Content
			if len(content) > 60 {
				content = content[:60] + "..."
			}
			fmt.Printf("  %s %s[%d] %s%s\n", icon, banner.Dim, i+1, banner.Reset, content)
		}
	}
	fmt.Println()
}

// extractAndSaveMemories 异步从对话中提取长期记忆
func extractAndSaveMemories(aiMgr *ai.Manager, memMgr *ai.MemoryManager, history []ai.Message) {
	// 构建提取请求
	prompt := ai.ExtractMemoryPrompt(history)
	resp, err := aiMgr.Chat(prompt)
	if err != nil {
		return
	}

	// 解析提取结果
	extracted := ai.ParseExtractedMemories(resp.Content)
	for _, item := range extracted {
		// 构建来源摘要
		source := ""
		for _, msg := range history {
			if msg.Role == ai.RoleUser {
				if len(msg.Content) > 100 {
					source = msg.Content[:100] + "..."
				} else {
					source = msg.Content
				}
			}
		}
		memMgr.AddMemory(item.Category, item.Content, source, item.Importance, item.Tags)
	}
}

func init() {
	// ai chat 标志
	aiChatCmd.Flags().StringVarP(&aiProvider, "provider", "p", "", "指定 AI 提供商 (openai/claude/gemini/qwen/deepseek)")
	aiChatCmd.Flags().StringVarP(&aiModel, "model", "m", "", "指定模型")
	aiChatCmd.Flags().StringVarP(&aiAgent, "agent", "a", "", "使用指定 Agent")
	aiChatCmd.Flags().BoolVarP(&aiWebSearch, "web", "w", false, "启用联网搜索")

	// 组装子命令
	aiProviderCmd.AddCommand(aiProviderListCmd)
	aiProviderCmd.AddCommand(aiProviderSwitchCmd)

	aiAgentCmd.AddCommand(aiAgentListCmd)
	aiAgentCmd.AddCommand(aiAgentRunCmd)
	aiAgentCmd.AddCommand(aiAgentInstallCmd)
	aiAgentCmd.AddCommand(aiAgentRemoveCmd)

	aiSkillCmd.AddCommand(aiSkillListCmd)
	aiSkillCmd.AddCommand(aiSkillInstallCmd)
	aiSkillCmd.AddCommand(aiSkillRemoveCmd)

	aiMemoryCmd.AddCommand(aiMemoryListCmd)
	aiMemoryCmd.AddCommand(aiMemoryClearCmd)
	aiMemoryCmd.AddCommand(aiMemorySessionsCmd)

	aiCmd.AddCommand(aiChatCmd)
	aiCmd.AddCommand(aiListCmd)
	aiCmd.AddCommand(aiConnectCmd)
	aiCmd.AddCommand(aiProviderCmd)
	aiCmd.AddCommand(aiAgentCmd)
	aiCmd.AddCommand(aiSkillCmd)
	aiCmd.AddCommand(aiMemoryCmd)

	rootCmd.AddCommand(aiCmd)
}

// ===== ai memory =====

var aiMemoryCmd = &cobra.Command{
	Use:   "memory",
	Short: "AI 记忆管理",
	Long: `管理 AI 的记忆系统 — 查看、清理长期记忆和会话历史。

记忆数据安全存储在本地: ~/.jiasine/mem/
  sessions/     会话记录 (短期记忆)
  long_term.json 长期记忆 (关键事实/偏好)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		memMgr, err := ai.NewMemoryManager()
		if err != nil {
			return err
		}
		printMemoryStatus(memMgr)
		return nil
	},
}

var aiMemoryListCmd = &cobra.Command{
	Use:   "list",
	Short: "查看所有长期记忆",
	RunE: func(cmd *cobra.Command, args []string) error {
		memMgr, err := ai.NewMemoryManager()
		if err != nil {
			return err
		}

		memories := memMgr.GetAllMemories()
		if len(memories) == 0 {
			fmt.Printf("%s暂无长期记忆%s\n", banner.Dim, banner.Reset)
			fmt.Printf("AI 会在对话结束时自动提取有价值的信息存为长期记忆\n")
			return nil
		}

		categories := map[string]string{
			"user_info":   "👤 用户信息",
			"preference":  "⚙️ 偏好设置",
			"project":     "📁 项目知识",
			"fact":        "📌 重要事实",
			"instruction": "📝 习惯指令",
		}

		fmt.Printf("\n%s🧠 长期记忆 (%d 条)%s\n", banner.BrightCyan, len(memories), banner.Reset)
		fmt.Println(strings.Repeat("─", 70))

		for i, e := range memories {
			catLabel := categories[e.Category]
			if catLabel == "" {
				catLabel = "💡 " + e.Category
			}
			fmt.Printf("\n  %s[%d]%s %s\n", banner.Dim, i+1, banner.Reset, catLabel)
			fmt.Printf("  内容: %s\n", e.Content)
			fmt.Printf("  %s重要性: %d | 访问: %d 次 | 创建: %s%s\n",
				banner.Dim, e.Importance, e.AccessCount,
				e.CreatedAt.Format("2006-01-02 15:04"), banner.Reset)
		}
		fmt.Println()
		return nil
	},
}

var aiMemoryClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "清空所有记忆 (长期 + 会话)",
	RunE: func(cmd *cobra.Command, args []string) error {
		memMgr, err := ai.NewMemoryManager()
		if err != nil {
			return err
		}

		sessions, activeMem, _ := memMgr.Stats()
		if sessions == 0 && activeMem == 0 {
			fmt.Printf("%s记忆已经是空的%s\n", banner.Dim, banner.Reset)
			return nil
		}

		fmt.Printf("即将清空: %d 条长期记忆 + %d 个会话记录\n", activeMem, sessions)
		fmt.Printf("确认清空? (y/N): ")
		var confirm string
		fmt.Scanln(&confirm)
		if strings.ToLower(confirm) != "y" {
			fmt.Println("已取消")
			return nil
		}

		if err := memMgr.ClearAllMemories(); err != nil {
			return err
		}
		fmt.Printf("%s🗑 已清空所有记忆%s\n", banner.BrightGreen, banner.Reset)
		return nil
	},
}

var aiMemorySessionsCmd = &cobra.Command{
	Use:   "sessions",
	Short: "查看最近的会话记录",
	RunE: func(cmd *cobra.Command, args []string) error {
		memMgr, err := ai.NewMemoryManager()
		if err != nil {
			return err
		}

		sessions := memMgr.ListSessions(10)
		if len(sessions) == 0 {
			fmt.Printf("%s暂无会话记录%s\n", banner.Dim, banner.Reset)
			return nil
		}

		fmt.Printf("\n%s📋 最近会话记录 (%d 条)%s\n", banner.BrightCyan, len(sessions), banner.Reset)
		fmt.Println(strings.Repeat("─", 70))

		for i, s := range sessions {
			agent := s.Agent
			if agent == "" {
				agent = "(无)"
			}
			fmt.Printf("  %s[%d]%s %s | Agent: %s | %d 轮 | %s · %s\n",
				banner.Dim, i+1, banner.Reset,
				s.CreatedAt.Format("2006-01-02 15:04"),
				agent, s.TurnCount,
				s.Provider, s.Model,
			)
			// 显示第一条用户消息预览
			for _, msg := range s.Messages {
				if msg.Role == ai.RoleUser {
					preview := msg.Content
					if len(preview) > 50 {
						preview = preview[:50] + "..."
					}
					fmt.Printf("       %s\"%s\"%s\n", banner.Dim, preview, banner.Reset)
					break
				}
			}
		}
		fmt.Println()
		return nil
	},
}
