package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/xiangjianhe-github/jiasinecli/internal/ai"
	"github.com/xiangjianhe-github/jiasinecli/internal/banner"
	"github.com/xiangjianhe-github/jiasinecli/internal/config"
	"github.com/xiangjianhe-github/jiasinecli/internal/render"
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

func printAIResponse(resp *ai.ChatResponse) {
	fmt.Println()
	fmt.Println(render.Markdown(resp.Content))
	fmt.Println()
	fmt.Printf("%s[%s · %s · tokens: %d]%s\n",
		banner.Dim, resp.Provider, resp.Model, resp.TotalTokens, banner.Reset)
}

// enterAIInteractive 进入 AI 交互式对话模式
// 连接模型 → 显示欢迎 → REPL 循环 → Ctrl+C 退出
func enterAIInteractive(agentName string) error {
	// 1. 获取 AI 管理器（含自动配置生成）
	aiMgr, err := getAIManager()
	if err != nil {
		return err
	}

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
	fmt.Printf("%s╭──────────────────────────────────────────╮%s\n", banner.Cyan, banner.Reset)
	fmt.Printf("%s│%s  🤖 AI 交互模式               %s⚡ 流式输出%s %s│%s\n", banner.Cyan, banner.Reset, banner.Dim, banner.Reset, banner.Cyan, banner.Reset)
	fmt.Printf("%s│%s  服务商: %-33s %s│%s\n", banner.Cyan, banner.BrightGreen, providerName, banner.Cyan, banner.Reset)
	fmt.Printf("%s│%s  模型:   %-33s %s│%s\n", banner.Cyan, banner.BrightGreen, modelName, banner.Cyan, banner.Reset)
	if agentName != "" {
		fmt.Printf("%s│%s  Agent:  %-33s %s│%s\n", banner.Cyan, banner.BrightCyan, agentName, banner.Cyan, banner.Reset)
	}
	fmt.Printf("%s│%s  联网:   %-33s %s│%s\n", banner.Cyan, banner.BrightCyan, webStatus, banner.Cyan, banner.Reset)
	fmt.Printf("%s│%s  记忆:   %-33s %s│%s\n", banner.Cyan, banner.BrightCyan, memStatus, banner.Cyan, banner.Reset)
	fmt.Printf("%s│%s  输入 help 查看命令，Ctrl+C 退出          %s│%s\n", banner.Cyan, banner.Dim, banner.Cyan, banner.Reset)
	fmt.Printf("%s╰──────────────────────────────────────────╯%s\n", banner.Cyan, banner.Reset)
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

	// 6. REPL 循环
	// 使用 goroutine 读取 stdin，让 select 可以同时监听 Ctrl+C 信号
	// 在 Windows 上，Go 的 signal handler 返回 TRUE(已处理)，
	// 不会中断 scanner.Scan() 的阻塞读取，因此必须用 goroutine + select 模式
	type inputLine struct {
		text string
		ok   bool
	}
	inputCh := make(chan inputLine, 1)
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
		for scanner.Scan() {
			inputCh <- inputLine{text: scanner.Text(), ok: true}
		}
		// EOF 或读取错误
		inputCh <- inputLine{ok: false}
	}()

	// 捕获 Ctrl+C — 防止进程直接终止
	sigChan := make(chan os.Signal, 2)
	signal.Notify(sigChan, os.Interrupt)

	for {
		// 显示提示符（带联网标记）
		if aiMgr.IsWebSearch() {
			fmt.Printf("%s🌐 AI> %s", banner.BrightCyan, banner.Reset)
		} else {
			fmt.Printf("%sAI> %s", banner.BrightCyan, banner.Reset)
		}

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
				input = strings.TrimSpace(res.text)
			}
		}
		if exit {
			break
		}

		if input == "" {
			continue
		}

		// 内置命令
		lower := strings.ToLower(input)
		if lower == "exit" || lower == "quit" || lower == "bye" {
			fmt.Printf("\n%s👋 已退出 AI 模式%s\n\n", banner.Dim, banner.Reset)
			break
		}
		if lower == "clear" || lower == "reset" {
			// 清空对话历史 (保留 system prompt)
			history = history[:0]
			if agentSystem != "" {
				history = append(history, ai.Message{Role: ai.RoleSystem, Content: agentSystem})
			}
			// 同时开始新会话
			if memMgr != nil {
				_ = memMgr.SaveSession()
				memMgr.StartSession(agentName, providerName, modelName)
			}
			fmt.Printf("%s对话历史已清空 (新会话)%s\n\n", banner.Dim, banner.Reset)
			continue
		}
		if lower == "memory" || lower == "mem" {
			if memMgr == nil {
				fmt.Printf("%s记忆系统未初始化%s\n\n", banner.Yellow, banner.Reset)
			} else {
				printMemoryStatus(memMgr)
			}
			continue
		}
		if lower == "memory clear" || lower == "mem clear" {
			if memMgr != nil {
				_ = memMgr.ClearAllMemories()
				fmt.Printf("%s🗑 已清空所有记忆%s\n\n", banner.Dim, banner.Reset)
			}
			continue
		}
		if lower == "web" || lower == "search" {
			enabled := aiMgr.ToggleWebSearch()
			// 持久化到配置文件
			_ = config.SetWebSearch(enabled)
			if enabled {
				fmt.Printf("%s🌐 联网搜索已开启 (已保存到配置)%s\n\n", banner.BrightGreen, banner.Reset)
			} else {
				fmt.Printf("%s📴 联网搜索已关闭 (已保存到配置)%s\n\n", banner.Dim, banner.Reset)
			}
			continue
		}
		if lower == "help" {
			printAIChatHelp()
			continue
		}

		// 加用户消息到历史
		history = append(history, ai.Message{Role: ai.RoleUser, Content: input})

		// 立即显示加载指示器
		fmt.Printf("%s⏳ ...%s", banner.Dim, banner.Reset)

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
					fmt.Printf("\r\033[K") // 清除加载指示器
					fmt.Printf("%s错误: %s%s\n\n", banner.Yellow, chunk.Error.Error(), banner.Reset)
					if toolIter == 0 && fullContent == "" {
						history = history[:len(history)-1]
					}
					toolLoopErr = true
					break
				}

				// 收到第一个 chunk 时清除加载指示器
				if firstChunk && (chunk.Type == "thinking" || chunk.Type == "content" || chunk.Type == "tool_use") {
					fmt.Printf("\r\033[K") // 回到行首并清除该行
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
						// 思考结束，切换到正文
						isThinking = false
						fmt.Printf("%s\n", banner.Reset)
						fmt.Printf("%s%s─ 思考完毕 ─%s\n\n", banner.Dim, banner.Italic, banner.Reset)
					}
					if !hasContent {
						hasContent = true
						fmt.Println() // 正文前换行
					}
					fmt.Print(chunk.Content)
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

			// 如果没有工具调用 → 完成
			if len(iterToolCalls) == 0 || iterStopReason != "tool_use" {
				if hasContent {
					fmt.Println() // 正文后换行
				}
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

		// 显示 token 统计
		webLabel := ""
		if aiMgr.IsWebSearch() {
			webLabel = " · 🌐联网"
		}
		fmt.Printf("%s[tokens: %d%s]%s\n\n", banner.Dim, totalUsage.TotalTokens, webLabel, banner.Reset)

		// 加助手回复到历史
		history = append(history, ai.Message{Role: ai.RoleAssistant, Content: fullContent})

		// 记忆系统: 追踪每轮对话的 user + assistant 消息
		if memMgr != nil {
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
	if memMgr != nil {
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
	fmt.Printf(`
%s AI 交互模式命令:%s
  直接输入文字       与 AI 对话 (流式输出 · 支持多轮上下文)
  %sweb%s / %ssearch%s      切换联网搜索 (开/关)
  %smemory%s / %smem%s      查看记忆状态
  %smem clear%s          清空所有记忆
  %sclear%s / %sreset%s     清空对话历史 (开始新会话)
  %sexit%s / %squit%s       退出 AI 模式
  %shelp%s               显示此帮助
  %sCtrl+C%s             退出 AI 模式

%s提示:%s AI 回复采用流式输出 (打字机效果)
      支持思考模型的推理过程可视化 (灰色斜体显示)

`,
		banner.BrightCyan, banner.Reset,
		banner.BrightGreen, banner.Reset, banner.BrightGreen, banner.Reset,
		banner.BrightGreen, banner.Reset, banner.BrightGreen, banner.Reset,
		banner.BrightGreen, banner.Reset,
		banner.BrightGreen, banner.Reset, banner.BrightGreen, banner.Reset,
		banner.BrightGreen, banner.Reset, banner.BrightGreen, banner.Reset,
		banner.BrightGreen, banner.Reset,
		banner.BrightGreen, banner.Reset,
		banner.Dim, banner.Reset,
	)
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
