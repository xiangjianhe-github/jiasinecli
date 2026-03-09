package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xiangjianhe-github/jiasinecli/internal/banner"
	"github.com/xiangjianhe-github/jiasinecli/internal/history"
	"github.com/xiangjianhe-github/jiasinecli/internal/render"
	"github.com/spf13/cobra"
)

var (
	historyLimit   int
	historyAgent   string
	historyBefore  string
	historyKeyword string
)

// historyCmd 历史记录命令
var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "管理对话历史记录",
	Long: `管理 AI 对话历史记录

历史记录功能可以帮你:
  • 回顾过去的对话
  • 搜索特定内容
  • 恢复中断的会话
  • 管理存储空间

子命令:
  sessions    列出所有历史会话
  show        查看特定会话的详细内容
  search      搜索历史记录
  delete      删除指定会话
  clear       清理旧的历史记录
  stats       查看历史统计信息

示例:
  jiasinecli history sessions                     # 列出最近的会话
  jiasinecli history sessions --agent general     # 列出指定 agent 的会话
  jiasinecli history show a1b2c3d4                # 查看会话详情
  jiasinecli history search "Go 语言"              # 搜索相关内容
  jiasinecli history delete a1b2c3d4              # 删除指定会话
  jiasinecli history clear --before "2026-01-01"  # 清理指定日期前的记录
`,
}

// historySessionsCmd 列出会话
var historySessionsCmd = &cobra.Command{
	Use:   "sessions",
	Short: "列出历史会话",
	Long:  "列出所有 AI 对话历史会话，可按 agent、时间范围等筛选",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := getHistoryManager()
		if err != nil {
			return err
		}
		defer mgr.Close()

		query := history.Query{
			Limit:     historyLimit,
			AgentName: historyAgent,
		}

		sessions, err := mgr.ListSessions(query)
		if err != nil {
			return fmt.Errorf("获取会话列表失败: %w", err)
		}

		if len(sessions) == 0 {
			fmt.Printf("\n%s📭 没有找到历史会话%s\n\n", banner.BrightYellow, banner.Reset)
			return nil
		}

		fmt.Printf("\n%s📚 历史会话列表%s (共 %d 个)\n", banner.BrightCyan, banner.Reset, len(sessions))
		fmt.Printf("%s%s%s\n\n", banner.Gray, strings.Repeat("─", 80), banner.Reset)

		for i, session := range sessions {
			// 格式化时间
			elapsed := formatDuration(time.Since(session.StartedAt))
			duration := ""
			if session.EndedAt != nil {
				sessionDuration := session.EndedAt.Sub(session.StartedAt)
				duration = fmt.Sprintf(" (持续 %s)", formatDuration(sessionDuration))
			} else {
				duration = " (🟢 进行中)"
			}

			// 会话标题（如果有）
			titleDisplay := ""
			if session.Title != "" {
				// 限制标题显示长度
				title := session.Title
				if len(title) > 60 {
					title = title[:60] + "..."
				}
				titleDisplay = fmt.Sprintf("\n    %s💬 主题:%s %s", banner.Dim, banner.Reset, title)
			}

			// 会话信息
			fmt.Printf("%s%2d.%s %s会话 ID:%s %s%s\n",
				banner.Gray, i+1, banner.Reset,
				banner.BrightCyan, banner.Reset, session.ID[:8]+"...", titleDisplay)
			fmt.Printf("    %sAgent:%s %s  %s提供商:%s %s/%s\n",
				banner.BrightGreen, banner.Reset, session.AgentName,
				banner.BrightBlue, banner.Reset, session.Provider, session.Model)
			fmt.Printf("    %s开始时间:%s %s (%s前)%s\n",
				banner.BrightMagenta, banner.Reset,
				session.StartedAt.Format("2006-01-02 15:04:05"),
				elapsed, duration)
			fmt.Printf("    %s消息数:%s %d\n",
				banner.BrightYellow, banner.Reset, session.MessageCount)

			if len(session.Tags) > 0 {
				fmt.Printf("    %s标签:%s %s\n",
					banner.White, banner.Reset, strings.Join(session.Tags, ", "))
			}
			fmt.Println()
		}

		return nil
	},
}

// historyShowCmd 查看会话详情
var historyShowCmd = &cobra.Command{
	Use:   "show <session-id>",
	Short: "查看会话详细内容",
	Long:  "查看指定会话的所有消息记录",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sessionID := args[0]
		mgr, err := getHistoryManager()
		if err != nil {
			return err
		}
		defer mgr.Close()

		// 获取会话信息
		query := history.Query{Limit: 1000}
		sessions, err := mgr.ListSessions(query)
		if err != nil {
			return fmt.Errorf("获取会话信息失败: %w", err)
		}

		var targetSession *history.Session
		for i := range sessions {
			if strings.HasPrefix(sessions[i].ID, sessionID) {
				targetSession = &sessions[i]
				break
			}
		}

		if targetSession == nil {
			return fmt.Errorf("未找到会话 ID: %s", sessionID)
		}

		// 获取消息列表
		messages, err := mgr.GetMessages(targetSession.ID)
		if err != nil {
			return fmt.Errorf("获取消息列表失败: %w", err)
		}

		// 显示会话信息
		fmt.Printf("\n%s╔═══════════════════════════════════════════════════════════════════════════╗%s\n",
			banner.BrightCyan, banner.Reset)
		fmt.Printf("%s║%s  会话详情                                                                   %s║%s\n",
			banner.BrightCyan, banner.Reset, banner.BrightCyan, banner.Reset)
		fmt.Printf("%s╠═══════════════════════════════════════════════════════════════════════════╣%s\n",
			banner.BrightCyan, banner.Reset)
		fmt.Printf("%s║%s  ID:       %s%-62s %s║%s\n",
			banner.BrightCyan, banner.Reset, banner.White, targetSession.ID, banner.BrightCyan, banner.Reset)
		fmt.Printf("%s║%s  Agent:    %s%-62s %s║%s\n",
			banner.BrightCyan, banner.Reset, banner.BrightGreen, targetSession.AgentName, banner.BrightCyan, banner.Reset)
		fmt.Printf("%s║%s  提供商:   %s%-62s %s║%s\n",
			banner.BrightCyan, banner.Reset, banner.BrightBlue, targetSession.Provider+"/"+targetSession.Model, banner.BrightCyan, banner.Reset)
		fmt.Printf("%s║%s  开始时间: %s%-62s %s║%s\n",
			banner.BrightCyan, banner.Reset, banner.BrightMagenta, targetSession.StartedAt.Format("2006-01-02 15:04:05"), banner.BrightCyan, banner.Reset)
		fmt.Printf("%s║%s  消息数:   %s%-62d %s║%s\n",
			banner.BrightCyan, banner.Reset, banner.BrightYellow, targetSession.MessageCount, banner.BrightCyan, banner.Reset)
		fmt.Printf("%s╚═══════════════════════════════════════════════════════════════════════════╝%s\n\n",
			banner.BrightCyan, banner.Reset)

		// 显示消息列表
		for i, msg := range messages {
			roleColor := banner.BrightGreen
			roleIcon := "👤"
			if msg.Role == "assistant" {
				roleColor = banner.BrightBlue
				roleIcon = "🤖"
			}

			fmt.Printf("%s%s [%s] %s%s%s\n",
				roleIcon, roleColor, msg.Role, banner.Reset,
				msg.Timestamp.Format("15:04:05"), banner.Reset)
			fmt.Printf("%s%s%s\n", banner.Gray, strings.Repeat("─", 80), banner.Reset)

			// 使用 Markdown 渲染输出
			rendered := render.Markdown(msg.Content)
			fmt.Println(rendered)

			if i < len(messages)-1 {
				fmt.Println()
			}
		}

		return nil
	},
}

// historySearchCmd 搜索历史记录
var historySearchCmd = &cobra.Command{
	Use:   "search <keyword>",
	Short: "搜索历史记录",
	Long:  "在历史消息中搜索包含指定关键词的内容",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		keyword := args[0]
		mgr, err := getHistoryManager()
		if err != nil {
			return err
		}
		defer mgr.Close()

		query := history.Query{
			Keyword: keyword,
			Limit:   historyLimit,
		}

		messages, err := mgr.SearchMessages(query)
		if err != nil {
			return fmt.Errorf("搜索失败: %w", err)
		}

		if len(messages) == 0 {
			fmt.Printf("\n%s🔍 未找到包含 \"%s\" 的消息%s\n\n", banner.BrightYellow, keyword, banner.Reset)
			return nil
		}

		fmt.Printf("\n%s🔍 搜索结果%s (找到 %d 条相关消息)\n", banner.BrightCyan, banner.Reset, len(messages))
		fmt.Printf("%s%s%s\n\n", banner.Gray, strings.Repeat("─", 80), banner.Reset)

		for i, msg := range messages {
			roleColor := banner.BrightGreen
			roleIcon := "👤"
			if msg.Role == "assistant" {
				roleColor = banner.BrightBlue
				roleIcon = "🤖"
			}

			fmt.Printf("%s%2d. %s%s [%s]%s %s\n",
				banner.Gray, i+1, roleIcon, roleColor, msg.Role, banner.Reset,
				msg.Timestamp.Format("2006-01-02 15:04:05"))
			fmt.Printf("    %s会话:%s %s\n", banner.BrightCyan, banner.Reset, msg.SessionID[:8]+"...")

			// 截取内容预览（最多显示 200 字符）
			content := msg.Content
			if len(content) > 200 {
				content = content[:200] + "..."
			}
			// 高亮关键词（简单替换）
			highlighted := strings.ReplaceAll(content, keyword,
				banner.BrightYellow+keyword+banner.Reset)

			fmt.Printf("    %s\n\n", highlighted)
		}

		return nil
	},
}

// historyDeleteCmd 删除会话
var historyDeleteCmd = &cobra.Command{
	Use:   "delete <session-id>",
	Short: "删除指定会话",
	Long:  "删除指定的历史会话及其所有消息",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sessionID := args[0]
		mgr, err := getHistoryManager()
		if err != nil {
			return err
		}
		defer mgr.Close()

		// 查找完整的会话 ID
		query := history.Query{Limit: 1000}
		sessions, err := mgr.ListSessions(query)
		if err != nil {
			return fmt.Errorf("获取会话列表失败: %w", err)
		}

		var fullID string
		for _, session := range sessions {
			if strings.HasPrefix(session.ID, sessionID) {
				fullID = session.ID
				break
			}
		}

		if fullID == "" {
			return fmt.Errorf("未找到会话 ID: %s", sessionID)
		}

		// 删除会话
		if err := mgr.DeleteSession(fullID); err != nil {
			return fmt.Errorf("删除会话失败: %w", err)
		}

		fmt.Printf("\n%s✅ 已删除会话:%s %s\n\n", banner.BrightGreen, banner.Reset, fullID[:8]+"...")
		return nil
	},
}

// historyClearCmd 清理旧记录
var historyClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "清理旧的历史记录",
	Long:  "删除指定日期之前的所有历史会话",
	RunE: func(cmd *cobra.Command, args []string) error {
		if historyBefore == "" {
			return fmt.Errorf("请使用 --before 参数指定日期 (格式: 2006-01-02)")
		}

		beforeTime, err := time.Parse("2006-01-02", historyBefore)
		if err != nil {
			return fmt.Errorf("日期格式错误: %w", err)
		}

		mgr, err := getHistoryManager()
		if err != nil {
			return err
		}
		defer mgr.Close()

		count, err := mgr.DeleteOldSessions(beforeTime)
		if err != nil {
			return fmt.Errorf("清理失败: %w", err)
		}

		fmt.Printf("\n%s✅ 已清理 %d 个会话%s (%s 之前的记录)\n\n",
			banner.BrightGreen, count, banner.Reset, historyBefore)
		return nil
	},
}

// historyResumeCmd 恢复历史会话
var historyResumeCmd = &cobra.Command{
	Use:   "resume <session-id>",
	Short: "恢复历史会话，继续对话",
	Long: `恢复指定的历史会话，加载对话历史后继续对话。

这个功能允许你:
  • 继续上次未完成的对话
  • 在中断后恢复对话上下文
  • 基于历史对话内容继续提问

示例:
  jiasinecli history resume a1b2c3d4        # 恢复指定会话ID的对话
  jiasinecli history sessions               # 先查看会话列表获取ID`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sessionID := args[0]

		// 获取历史管理器
		historyMgr, err := getHistoryManager()
		if err != nil {
			return err
		}
		defer historyMgr.Close()

		// 获取会话信息
		session, err := historyMgr.GetSession(sessionID)
		if err != nil {
			return fmt.Errorf("会话未找到: %w", err)
		}

		// 获取会话所有消息
		messages, err := historyMgr.GetMessages(sessionID)
		if err != nil {
			return fmt.Errorf("获取会话消息失败: %w", err)
		}

		if len(messages) == 0 {
			return fmt.Errorf("会话没有消息记录")
		}

		// 显示会话信息
		fmt.Printf("\n%s📚 正在恢复会话...%s\n", banner.BrightCyan, banner.Reset)
		fmt.Printf("%s会话 ID:%s %s\n", banner.Dim, banner.Reset, session.ID[:8]+"...")
		if session.Title != "" {
			fmt.Printf("%s主题:%s %s\n", banner.Dim, banner.Reset, session.Title)
		}
		fmt.Printf("%sAgent:%s %s  %s提供商:%s %s/%s\n",
			banner.Dim, banner.Reset, session.AgentName,
			banner.Dim, banner.Reset, session.Provider, session.Model)
		fmt.Printf("%s消息数:%s %d 条\n", banner.Dim, banner.Reset, len(messages))
		fmt.Printf("%s%s%s\n", banner.Gray, strings.Repeat("─", 60), banner.Reset)

		// 显示最近几条消息预览
		previewCount := 3
		if len(messages) < previewCount {
			previewCount = len(messages)
		}
		fmt.Printf("\n%s最近 %d 条消息：%s\n", banner.Dim, previewCount, banner.Reset)
		for i := len(messages) - previewCount; i < len(messages); i++ {
			msg := messages[i]
			roleIcon := "👤"
			roleColor := banner.BrightGreen
			if msg.Role == "assistant" {
				roleIcon = "🤖"
				roleColor = banner.BrightCyan
			}
			preview := msg.Content
			if len(preview) > 80 {
				preview = preview[:80] + "..."
			}
			fmt.Printf("  %s%s %s:%s %s\n", roleColor, roleIcon, msg.Role, banner.Reset, preview)
		}

		fmt.Printf("\n%s✨ 按 Enter 继续对话，Ctrl+C 取消...%s ", banner.BrightYellow, banner.Reset)
		fmt.Scanln()

		// 提示用户如何继续对话
		fmt.Printf("\n%s✓ 会话已恢复！%s\n", banner.BrightGreen, banner.Reset)
		fmt.Printf("%s提示: 使用以下命令继续对话:%s\n\n", banner.Dim, banner.Reset)
		fmt.Printf("  %sjiasinecli ai chat%s\n", banner.BrightCyan, banner.Reset)
		if session.AgentName != "" && session.AgentName != "general" {
			fmt.Printf("  %s或指定 Agent: jiasinecli ai chat --agent %s%s\n\n", banner.Dim, session.AgentName, banner.Reset)
		}
		fmt.Printf("%s注意: 新对话将自动使用相同的 Agent 和 Provider 设置%s\n", banner.Dim, banner.Reset)

		return nil
	},
}

// historyStatsCmd 统计信息
var historyStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "查看历史统计信息",
	Long:  "显示历史记录的统计数据",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := getHistoryManager()
		if err != nil {
			return err
		}
		defer mgr.Close()

		stats, err := mgr.GetStats()
		if err != nil {
			return fmt.Errorf("获取统计信息失败: %w", err)
		}

		fmt.Printf("\n%s📊 历史记录统计%s\n", banner.BrightCyan, banner.Reset)
		fmt.Printf("%s%s%s\n\n", banner.Gray, strings.Repeat("─", 40), banner.Reset)
		fmt.Printf("  %s总会话数:%s %d\n", banner.BrightGreen, banner.Reset, stats["total_sessions"])
		fmt.Printf("  %s总消息数:%s %d\n", banner.BrightBlue, banner.Reset, stats["total_messages"])

		avgMessages := 0
		if sessions, ok := stats["total_sessions"].(int); ok && sessions > 0 {
			if messages, ok := stats["total_messages"].(int); ok {
				avgMessages = messages / sessions
			}
		}
		fmt.Printf("  %s平均消息数:%s %d/会话\n\n", banner.BrightYellow, banner.Reset, avgMessages)

		return nil
	},
}

// getHistoryManager 获取历史记录管理器
func getHistoryManager() (*history.Manager, error) {
	// 获取用户目录
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("获取用户目录失败: %w", err)
	}

	// 配置目录路径
	configDir := filepath.Join(home, ".jiasine")
	dbPath := filepath.Join(configDir, "history.db")

	mgr, err := history.NewManager(dbPath)
	if err != nil {
		return nil, fmt.Errorf("初始化历史管理器失败: %w", err)
	}

	return mgr, nil
}

// formatDuration 格式化时间间隔
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0f秒", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.0f分钟", d.Minutes())
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%.1f小时", d.Hours())
	}
	days := int(d.Hours() / 24)
	return fmt.Sprintf("%d天", days)
}

func init() {
	// 添加到根命令
	rootCmd.AddCommand(historyCmd)

	// 添加子命令
	historyCmd.AddCommand(historySessionsCmd)
	historyCmd.AddCommand(historyShowCmd)
	historyCmd.AddCommand(historySearchCmd)
	historyCmd.AddCommand(historyDeleteCmd)
	historyCmd.AddCommand(historyClearCmd)
	historyCmd.AddCommand(historyResumeCmd)  // 恢复会话
	historyCmd.AddCommand(historyStatsCmd)

	// sessions 命令的标志
	historySessionsCmd.Flags().IntVarP(&historyLimit, "limit", "l", 10, "显示的会话数量")
	historySessionsCmd.Flags().StringVarP(&historyAgent, "agent", "a", "", "筛选指定 agent 的会话")

	// search 命令的标志
	historySearchCmd.Flags().IntVarP(&historyLimit, "limit", "l", 20, "显示的结果数量")

	// clear 命令的标志
	historyClearCmd.Flags().StringVarP(&historyBefore, "before", "b", "", "清理指定日期之前的记录 (格式: 2006-01-02)")
}
