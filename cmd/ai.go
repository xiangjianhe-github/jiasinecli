package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strings"

	"github.com/xiangjianhe-github/jiasinecli/internal/ai"
	"github.com/xiangjianhe-github/jiasinecli/internal/banner"
	"github.com/xiangjianhe-github/jiasinecli/internal/config"
	"github.com/spf13/cobra"
)

var (
	aiProvider string // --provider 标志
	aiModel    string // --model 标志
	aiAgent    string // --agent 标志
)

// getAIManager 懒加载 AI 管理器，自动检测并生成配置
func getAIManager() (*ai.Manager, error) {
	// 检查是否有有效的 AI 配置
	if !config.HasValidAIProviders() {
		// 自动生成配置模板
		configPath, created, err := config.EnsureAIConfig()
		if err != nil {
			return nil, fmt.Errorf("生成 AI 配置失败: %w", err)
		}
		if created {
			fmt.Printf("\n%s📄 已自动生成 AI 配置模板:%s\n", banner.BrightCyan, banner.Reset)
			fmt.Printf("   %s%s%s\n\n", banner.BrightGreen, configPath, banner.Reset)
			fmt.Printf("请编辑配置文件，填入你的 API Key 后重新运行。\n\n")
			fmt.Printf("%s各服务商 API Key 获取地址:%s\n", banner.Dim, banner.Reset)
			fmt.Printf("  DeepSeek  (推荐)  https://platform.deepseek.com\n")
			fmt.Printf("  OpenAI           https://platform.openai.com/api-keys\n")
			fmt.Printf("  Anthropic        https://console.anthropic.com\n")
			fmt.Printf("  Google Gemini    https://aistudio.google.com/apikey\n")
			fmt.Printf("  通义千问          https://dashscope.console.aliyun.com\n\n")
			return nil, fmt.Errorf("请先配置 API Key")
		}
		// 文件存在但无有效 key
		fmt.Printf("\n%s⚠ AI 提供商未配置 API Key%s\n", banner.Yellow, banner.Reset)
		fmt.Printf("请编辑: %s%s%s\n\n", banner.BrightGreen, configPath, banner.Reset)
		return nil, fmt.Errorf("请先配置 API Key")
	}

	cfg := config.Get()
	mgr := ai.NewManager(ai.AIConfig{
		Active:    cfg.AI.Active,
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
		// 'ai' 不带子命令 → 进入交互式 AI 对话模式
		return enterAIInteractive("")
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
			// 无参数 → 进入交互模式
			return enterAIInteractive(aiAgent)
		}

		// 有参数 → 单次对话
		prompt := strings.Join(args, " ")

		// 如果指定了 Agent，走 Agent 流程
		if aiAgent != "" {
			agentMgr := getAgentManager()
			resp, err := agentMgr.Run(aiAgent, prompt)
			if err != nil {
				return err
			}
			printAIResponse(resp)
			return nil
		}

		// 直接调用 AI
		aiMgr, err := getAIManager()
		if err != nil {
			return err
		}

		if aiProvider != "" {
			if err := aiMgr.SetActive(aiProvider); err != nil {
				return err
			}
		}

		resp, err := aiMgr.ChatWith(aiProvider, prompt)
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
  jiasinecli ai agent run assistant "帮我总结这段文字"
  jiasinecli ai agent run coder "用 Rust 写快速排序"
  jiasinecli ai agent run translator "translate to japanese: hello world"`,
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
	Use:   "install [path]",
	Short: "安装 Skill (从 JSON 文件)",
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

func printAIResponse(resp *ai.ChatResponse) {
	fmt.Println()
	fmt.Println(resp.Content)
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

	// 3. 连接验证 — 发送一条极短的测试请求
	providerName, modelName := aiMgr.ActiveProviderInfo()
	fmt.Printf("\n%s正在连接 %s (%s) ...%s", banner.Dim, providerName, modelName, banner.Reset)

	testResp, testErr := aiMgr.Chat("ping")
	if testErr != nil {
		fmt.Printf(" %s失败%s\n", banner.Yellow, banner.Reset)
		return fmt.Errorf("模型连接失败: %w", testErr)
	}
	_ = testResp
	fmt.Printf(" %s✓ 已连接%s\n", banner.BrightGreen, banner.Reset)

	// 4. 显示欢迎信息
	fmt.Println()
	fmt.Printf("%s╭──────────────────────────────────────────╮%s\n", banner.Cyan, banner.Reset)
	fmt.Printf("%s│%s  🤖 AI 交互模式                           %s│%s\n", banner.Cyan, banner.Reset, banner.Cyan, banner.Reset)
	fmt.Printf("%s│%s  服务商: %-15s 模型: %-12s%s│%s\n", banner.Cyan, banner.BrightGreen, providerName, modelName, banner.Cyan, banner.Reset)
	if agentName != "" {
		fmt.Printf("%s│%s  Agent: %-33s %s│%s\n", banner.Cyan, banner.BrightCyan, agentName, banner.Cyan, banner.Reset)
	}
	fmt.Printf("%s│%s  输入问题开始对话，Ctrl+C 退出             %s│%s\n", banner.Cyan, banner.Dim, banner.Cyan, banner.Reset)
	fmt.Printf("%s╰──────────────────────────────────────────╯%s\n", banner.Cyan, banner.Reset)
	fmt.Println()

	// 5. 保存对话历史（上下文）
	history := []ai.Message{}

	// 如果使用 Agent，加入系统提示词
	var agentSystem string
	if agentName != "" {
		agentMgr := getAgentManager()
		system, err := agentMgr.GetSystemPrompt(agentName)
		if err != nil {
			fmt.Printf("%s⚠ Agent '%s' 未找到，将使用普通对话模式%s\n\n", banner.Yellow, agentName, banner.Reset)
		} else {
			agentSystem = system
		}
		if agentSystem != "" {
			history = append(history, ai.Message{Role: ai.RoleSystem, Content: agentSystem})
		}
	}

	// 6. REPL 循环
	scanner := bufio.NewScanner(os.Stdin)
	// 设置更大的缓冲区（1MB），允许较长的输入
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	// 捕获 Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	// 用 goroutine 监听 Ctrl+C
	done := make(chan bool, 1)
	go func() {
		<-sigChan
		fmt.Printf("\n\n%s👋 已退出 AI 模式%s\n\n", banner.Dim, banner.Reset)
		done <- true
	}()

	for {
		// 显示提示符
		fmt.Printf("%sAI> %s", banner.BrightCyan, banner.Reset)

		// 非阻塞检查是否收到退出信号
		select {
		case <-done:
			signal.Stop(sigChan)
			return nil
		default:
		}

		if !scanner.Scan() {
			// EOF 或错误
			break
		}

		input := strings.TrimSpace(scanner.Text())
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
			// 清空对话历史
			history = history[:0]
			if agentSystem != "" {
				history = append(history, ai.Message{Role: ai.RoleSystem, Content: agentSystem})
			}
			fmt.Printf("%s对话历史已清空%s\n\n", banner.Dim, banner.Reset)
			continue
		}
		if lower == "help" {
			printAIChatHelp()
			continue
		}

		// 加用户消息到历史
		history = append(history, ai.Message{Role: ai.RoleUser, Content: input})

		// 发送请求
		fmt.Printf("%s思考中...%s", banner.Dim, banner.Reset)
		resp, err := aiMgr.ChatMessages("", history)
		// 清除"思考中..."
		fmt.Printf("\r%s", strings.Repeat(" ", 20))
		fmt.Printf("\r")

		if err != nil {
			fmt.Printf("%s错误: %s%s\n\n", banner.Yellow, err.Error(), banner.Reset)
			// 移除失败的用户消息
			history = history[:len(history)-1]
			continue
		}

		// 显示回复
		fmt.Println(resp.Content)
		fmt.Printf("%s[tokens: %d]%s\n\n", banner.Dim, resp.TotalTokens, banner.Reset)

		// 加助手回复到历史
		history = append(history, ai.Message{Role: ai.RoleAssistant, Content: resp.Content})

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

	signal.Stop(sigChan)
	return nil
}

func printAIChatHelp() {
	fmt.Printf(`
%s AI 交互模式命令:%s
  直接输入文字       与 AI 对话 (支持多轮上下文)
  %sclear%s / %sreset%s     清空对话历史
  %sexit%s / %squit%s       退出 AI 模式
  %shelp%s               显示此帮助
  %sCtrl+C%s             退出 AI 模式

`,
		banner.BrightCyan, banner.Reset,
		banner.BrightGreen, banner.Reset, banner.BrightGreen, banner.Reset,
		banner.BrightGreen, banner.Reset, banner.BrightGreen, banner.Reset,
		banner.BrightGreen, banner.Reset,
		banner.BrightGreen, banner.Reset,
	)
}

func init() {
	// ai chat 标志
	aiChatCmd.Flags().StringVarP(&aiProvider, "provider", "p", "", "指定 AI 提供商 (openai/claude/gemini/qwen/deepseek)")
	aiChatCmd.Flags().StringVarP(&aiModel, "model", "m", "", "指定模型")
	aiChatCmd.Flags().StringVarP(&aiAgent, "agent", "a", "", "使用指定 Agent")

	// 组装子命令
	aiProviderCmd.AddCommand(aiProviderListCmd)
	aiProviderCmd.AddCommand(aiProviderSwitchCmd)

	aiAgentCmd.AddCommand(aiAgentListCmd)
	aiAgentCmd.AddCommand(aiAgentRunCmd)

	aiSkillCmd.AddCommand(aiSkillListCmd)
	aiSkillCmd.AddCommand(aiSkillInstallCmd)
	aiSkillCmd.AddCommand(aiSkillRemoveCmd)

	aiCmd.AddCommand(aiChatCmd)
	aiCmd.AddCommand(aiProviderCmd)
	aiCmd.AddCommand(aiAgentCmd)
	aiCmd.AddCommand(aiSkillCmd)

	rootCmd.AddCommand(aiCmd)
}
