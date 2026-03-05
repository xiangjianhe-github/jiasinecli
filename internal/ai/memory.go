// Package ai - AI 记忆系统
// 分层架构: 短期记忆(会话历史) + 长期记忆(关键事实/偏好)
// 所有数据存储在本地 ~/.jiasine/mem/ 目录，保障隐私安全
package ai

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/xiangjianhe-github/jiasinecli/internal/logger"
)

// ==================== 数据结构 ====================

// MemoryEntry 长期记忆条目
type MemoryEntry struct {
	ID         string    `json:"id"`                    // 唯一标识 (hash)
	Category   string    `json:"category"`              // 分类: user_info / preference / project / fact / instruction
	Content    string    `json:"content"`               // 记忆内容
	Source     string    `json:"source,omitempty"`       // 来源摘要 (触发提取的对话片段)
	Importance int       `json:"importance"`             // 重要性 1-10
	AccessCount int      `json:"access_count"`           // 被调用次数
	CreatedAt  time.Time `json:"created_at"`             // 创建时间
	UpdatedAt  time.Time `json:"updated_at"`             // 最后更新时间
	LastAccess time.Time `json:"last_access"`            // 最后访问时间
	Tags       []string  `json:"tags,omitempty"`         // 标签
	Expired    bool      `json:"expired,omitempty"`      // 是否已过期/遗忘
}

// SessionRecord 短期记忆 — 会话记录
type SessionRecord struct {
	ID        string    `json:"id"`                     // 会话 ID (时间戳)
	Agent     string    `json:"agent,omitempty"`        // 使用的 Agent
	Provider  string    `json:"provider,omitempty"`     // 使用的服务商
	Model     string    `json:"model,omitempty"`        // 使用的模型
	Messages  []Message `json:"messages"`               // 对话消息历史
	Summary   string    `json:"summary,omitempty"`      // 会话摘要 (由 AI 生成)
	CreatedAt time.Time `json:"created_at"`             // 会话开始时间
	UpdatedAt time.Time `json:"updated_at"`             // 最后更新时间
	TurnCount int       `json:"turn_count"`             // 对话轮次
}

// LongTermStore 长期记忆存储
type LongTermStore struct {
	Version  int            `json:"version"`           // 数据版本
	Entries  []*MemoryEntry `json:"entries"`           // 所有记忆条目
	Metadata struct {
		TotalSessions    int       `json:"total_sessions"`    // 历史会话总数
		TotalExtractions int       `json:"total_extractions"` // 提取记忆总次数
		LastCleanup      time.Time `json:"last_cleanup"`      // 最后清理时间
	} `json:"metadata"`
}

// ==================== 记忆管理器 ====================

// MemoryManager 记忆管理器
// 负责短期记忆(会话)和长期记忆的形成、存储、调用、更新、遗忘
type MemoryManager struct {
	memDir     string         // ~/.jiasine/mem/
	sessionDir string         // ~/.jiasine/mem/sessions/
	longTermFp string         // ~/.jiasine/mem/long_term.json
	longTerm   *LongTermStore // 长期记忆数据
	current    *SessionRecord // 当前活跃会话
	mu         sync.RWMutex
}

// NewMemoryManager 创建记忆管理器
func NewMemoryManager() (*MemoryManager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("获取用户目录失败: %w", err)
	}

	memDir := filepath.Join(home, ".jiasine", "mem")
	sessionDir := filepath.Join(memDir, "sessions")
	longTermFp := filepath.Join(memDir, "long_term.json")

	// 确保目录存在
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		return nil, fmt.Errorf("创建记忆目录失败: %w", err)
	}

	m := &MemoryManager{
		memDir:     memDir,
		sessionDir: sessionDir,
		longTermFp: longTermFp,
	}

	// 加载长期记忆
	m.loadLongTerm()

	return m, nil
}

// ==================== 短期记忆: 会话管理 ====================

// StartSession 开始新会话
func (m *MemoryManager) StartSession(agent, provider, model string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.current = &SessionRecord{
		ID:        time.Now().Format("20060102-150405"),
		Agent:     agent,
		Provider:  provider,
		Model:     model,
		Messages:  []Message{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// AppendMessage 追加消息到当前会话
func (m *MemoryManager) AppendMessage(msg Message) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.current == nil {
		return
	}

	m.current.Messages = append(m.current.Messages, msg)
	m.current.UpdatedAt = time.Now()

	// 统计对话轮次 (user+assistant = 1 轮)
	userCount := 0
	for _, ms := range m.current.Messages {
		if ms.Role == RoleUser {
			userCount++
		}
	}
	m.current.TurnCount = userCount
}

// SaveSession 保存当前会话到磁盘
func (m *MemoryManager) SaveSession() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.current == nil || len(m.current.Messages) == 0 {
		return nil
	}

	// 过滤掉 system 消息 (不保存到记录中，每次重新注入)
	var saveMessages []Message
	for _, msg := range m.current.Messages {
		if msg.Role != RoleSystem {
			saveMessages = append(saveMessages, msg)
		}
	}

	if len(saveMessages) == 0 {
		return nil
	}

	record := *m.current
	record.Messages = saveMessages

	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化会话失败: %w", err)
	}

	fp := filepath.Join(m.sessionDir, record.ID+".json")
	if err := os.WriteFile(fp, data, 0644); err != nil {
		return fmt.Errorf("写入会话文件失败: %w", err)
	}

	logger.Info("会话已保存", "id", record.ID, "turns", record.TurnCount)
	return nil
}

// LoadLastSession 加载最近一次会话记录
// 返回消息历史(不含 system), 如果没有则返回 nil
func (m *MemoryManager) LoadLastSession() *SessionRecord {
	sessions := m.listSessionFiles()
	if len(sessions) == 0 {
		return nil
	}

	// 按文件名倒序 (最近的在前)
	sort.Sort(sort.Reverse(sort.StringSlice(sessions)))

	for _, fp := range sessions {
		record, err := m.loadSession(fp)
		if err != nil {
			logger.Info("加载会话失败", "file", fp, "error", err)
			continue
		}
		// 只返回有实际对话的记录
		if record.TurnCount > 0 {
			return record
		}
	}

	return nil
}

// ListSessions 列出所有会话记录 (最近的在前)
func (m *MemoryManager) ListSessions(limit int) []*SessionRecord {
	files := m.listSessionFiles()
	sort.Sort(sort.Reverse(sort.StringSlice(files)))

	var sessions []*SessionRecord
	for _, fp := range files {
		if limit > 0 && len(sessions) >= limit {
			break
		}
		record, err := m.loadSession(fp)
		if err != nil {
			continue
		}
		sessions = append(sessions, record)
	}

	return sessions
}

// CleanOldSessions 清理旧会话, 只保留最近 N 个
func (m *MemoryManager) CleanOldSessions(keep int) int {
	files := m.listSessionFiles()
	if len(files) <= keep {
		return 0
	}

	sort.Sort(sort.Reverse(sort.StringSlice(files)))

	removed := 0
	for i := keep; i < len(files); i++ {
		if err := os.Remove(files[i]); err == nil {
			removed++
		}
	}

	logger.Info("清理旧会话", "removed", removed, "kept", keep)
	return removed
}

func (m *MemoryManager) listSessionFiles() []string {
	entries, err := os.ReadDir(m.sessionDir)
	if err != nil {
		return nil
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			files = append(files, filepath.Join(m.sessionDir, e.Name()))
		}
	}
	return files
}

func (m *MemoryManager) loadSession(fp string) (*SessionRecord, error) {
	data, err := os.ReadFile(fp)
	if err != nil {
		return nil, err
	}

	var record SessionRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, err
	}

	return &record, nil
}

// ==================== 长期记忆: 知识提取与管理 ====================

// loadLongTerm 从磁盘加载长期记忆
func (m *MemoryManager) loadLongTerm() {
	m.longTerm = &LongTermStore{
		Version: 1,
		Entries: []*MemoryEntry{},
	}

	data, err := os.ReadFile(m.longTermFp)
	if err != nil {
		// 文件不存在，使用空存储
		return
	}

	if err := json.Unmarshal(data, m.longTerm); err != nil {
		logger.Info("解析长期记忆失败，使用空存储", "error", err)
		m.longTerm = &LongTermStore{
			Version: 1,
			Entries: []*MemoryEntry{},
		}
	}
}

// saveLongTerm 将长期记忆写入磁盘
func (m *MemoryManager) saveLongTerm() error {
	data, err := json.MarshalIndent(m.longTerm, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化长期记忆失败: %w", err)
	}

	if err := os.WriteFile(m.longTermFp, data, 0644); err != nil {
		return fmt.Errorf("写入长期记忆失败: %w", err)
	}

	return nil
}

// AddMemory 添加一条长期记忆
func (m *MemoryManager) AddMemory(category, content, source string, importance int, tags []string) *MemoryEntry {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 生成 ID
	hash := sha256.Sum256([]byte(category + ":" + content))
	id := fmt.Sprintf("%x", hash[:8])

	// 检查是否已存在相似记忆 (同类别 + 相同 ID)
	for _, e := range m.longTerm.Entries {
		if e.ID == id && !e.Expired {
			// 更新已有记忆
			e.Content = content
			e.UpdatedAt = time.Now()
			e.AccessCount++
			if importance > e.Importance {
				e.Importance = importance
			}
			_ = m.saveLongTerm()
			return e
		}
	}

	entry := &MemoryEntry{
		ID:          id,
		Category:    category,
		Content:     content,
		Source:      source,
		Importance:  importance,
		AccessCount: 0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		LastAccess:  time.Now(),
		Tags:        tags,
	}

	m.longTerm.Entries = append(m.longTerm.Entries, entry)
	m.longTerm.Metadata.TotalExtractions++
	_ = m.saveLongTerm()

	logger.Info("添加长期记忆", "id", id, "category", category, "importance", importance)
	return entry
}

// RecallMemories 检索相关的长期记忆
// 返回活跃的、按重要性排序的记忆条目
func (m *MemoryManager) RecallMemories(limit int) []*MemoryEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 收集所有活跃记忆
	var active []*MemoryEntry
	for _, e := range m.longTerm.Entries {
		if !e.Expired {
			active = append(active, e)
		}
	}

	// 按综合评分排序: importance * 2 + accessCount + 时间衰减
	now := time.Now()
	sort.Slice(active, func(i, j int) bool {
		scoreI := m.calcScore(active[i], now)
		scoreJ := m.calcScore(active[j], now)
		return scoreI > scoreJ
	})

	// 更新访问时间
	if limit > 0 && len(active) > limit {
		active = active[:limit]
	}

	for _, e := range active {
		e.LastAccess = now
		e.AccessCount++
	}

	return active
}

// ForgetMemory 遗忘一条记忆 (标记为过期)
func (m *MemoryManager) ForgetMemory(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, e := range m.longTerm.Entries {
		if e.ID == id {
			e.Expired = true
			e.UpdatedAt = time.Now()
			_ = m.saveLongTerm()
			logger.Info("遗忘记忆", "id", id, "content", e.Content[:min(30, len(e.Content))])
			return true
		}
	}
	return false
}

// AutoDecay 自动衰减: 对长期未访问的低重要性记忆进行遗忘
// 返回被遗忘的记忆数量
func (m *MemoryManager) AutoDecay() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	decayed := 0

	for _, e := range m.longTerm.Entries {
		if e.Expired {
			continue
		}

		daysSinceAccess := now.Sub(e.LastAccess).Hours() / 24

		// 遗忘规则:
		// - 重要性 <= 3 且超过 30 天未访问
		// - 重要性 <= 5 且超过 90 天未访问
		// - 任何记忆超过 365 天未访问且访问次数 < 3
		shouldDecay := false
		if e.Importance <= 3 && daysSinceAccess > 30 {
			shouldDecay = true
		} else if e.Importance <= 5 && daysSinceAccess > 90 {
			shouldDecay = true
		} else if daysSinceAccess > 365 && e.AccessCount < 3 {
			shouldDecay = true
		}

		if shouldDecay {
			e.Expired = true
			e.UpdatedAt = now
			decayed++
		}
	}

	if decayed > 0 {
		m.longTerm.Metadata.LastCleanup = now
		_ = m.saveLongTerm()
		logger.Info("自动衰减记忆", "decayed", decayed)
	}

	return decayed
}

// GetAllMemories 获取所有活跃记忆 (用于 CLI 展示)
func (m *MemoryManager) GetAllMemories() []*MemoryEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var active []*MemoryEntry
	for _, e := range m.longTerm.Entries {
		if !e.Expired {
			active = append(active, e)
		}
	}
	return active
}

// ClearAllMemories 清空所有记忆 (长期 + 短期)
func (m *MemoryManager) ClearAllMemories() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 清空长期记忆
	m.longTerm = &LongTermStore{
		Version: 1,
		Entries: []*MemoryEntry{},
	}
	if err := m.saveLongTerm(); err != nil {
		return err
	}

	// 清空所有会话文件
	files := m.listSessionFiles()
	for _, fp := range files {
		os.Remove(fp)
	}

	logger.Info("已清空所有记忆")
	return nil
}

// calcScore 计算记忆的综合评分
func (m *MemoryManager) calcScore(e *MemoryEntry, now time.Time) float64 {
	// 基础分 = 重要性 * 2
	score := float64(e.Importance) * 2.0

	// 访问频率加成
	score += float64(e.AccessCount) * 0.5

	// 时间衰减: 越久未访问分数越低
	daysSince := now.Sub(e.LastAccess).Hours() / 24
	if daysSince > 0 {
		score -= daysSince * 0.02 // 每天衰减 0.02
	}

	// 新鲜度加成: 最近创建的记忆有额外分数
	daysSinceCreated := now.Sub(e.CreatedAt).Hours() / 24
	if daysSinceCreated < 7 {
		score += 2.0 // 一周内的新记忆加 2 分
	}

	return score
}

// ==================== 记忆提取 (从对话中提取关键信息) ====================

// ExtractMemoryPrompt 生成让 AI 从对话中提取记忆的提示词
// 在会话结束时发送给 AI，要求它提取值得记住的信息
func ExtractMemoryPrompt(messages []Message) string {
	// 构建对话摘要
	var conversation strings.Builder
	for _, msg := range messages {
		if msg.Role == RoleSystem || msg.Role == RoleToolResult || msg.Role == RoleAssistantToolUse {
			continue
		}
		role := "用户"
		if msg.Role == RoleAssistant {
			role = "助手"
		}
		content := msg.Content
		if len(content) > 500 {
			content = content[:500] + "..."
		}
		conversation.WriteString(fmt.Sprintf("[%s]: %s\n", role, content))
	}

	return fmt.Sprintf(`请分析以下对话，提取值得长期记住的关键信息。

对话内容:
---
%s
---

请以 JSON 数组格式输出需要记忆的条目，每个条目包含:
- category: 分类 (user_info=用户信息, preference=偏好, project=项目知识, fact=事实, instruction=指令习惯)
- content: 要记住的内容 (简洁明了，一句话)
- importance: 重要性 1-10
- tags: 相关标签数组

规则:
1. 只提取有长期价值的信息，忽略临时性对话
2. 用户的个人信息、偏好、常用项目路径等优先级高
3. 如果没有值得记忆的内容，返回空数组 []
4. 每次最多提取 5 条
5. 直接输出 JSON，不要其他文字

示例输出:
[
  {"category": "user_info", "content": "用户名叫小明，是 Go 后端开发者", "importance": 8, "tags": ["用户", "Go"]},
  {"category": "preference", "content": "用户偏好使用 Claude 模型", "importance": 6, "tags": ["偏好", "模型"]}
]`, conversation.String())
}

// ParseExtractedMemories 解析 AI 返回的记忆提取结果
func ParseExtractedMemories(response string) []struct {
	Category   string   `json:"category"`
	Content    string   `json:"content"`
	Importance int      `json:"importance"`
	Tags       []string `json:"tags"`
} {
	// 尝试从响应中提取 JSON 数组
	content := strings.TrimSpace(response)

	// 如果被 markdown 代码块包裹，去掉
	if strings.HasPrefix(content, "```") {
		lines := strings.Split(content, "\n")
		var jsonLines []string
		inBlock := false
		for _, line := range lines {
			if strings.HasPrefix(line, "```") {
				inBlock = !inBlock
				continue
			}
			if inBlock {
				jsonLines = append(jsonLines, line)
			}
		}
		content = strings.Join(jsonLines, "\n")
	}

	// 找到 JSON 数组的起止位置
	start := strings.Index(content, "[")
	end := strings.LastIndex(content, "]")
	if start >= 0 && end > start {
		content = content[start : end+1]
	}

	var results []struct {
		Category   string   `json:"category"`
		Content    string   `json:"content"`
		Importance int      `json:"importance"`
		Tags       []string `json:"tags"`
	}

	if err := json.Unmarshal([]byte(content), &results); err != nil {
		logger.Info("解析记忆提取结果失败", "error", err, "content", content[:min(100, len(content))])
		return nil
	}

	return results
}

// ==================== 记忆注入 (构建系统提示词) ====================

// BuildMemoryContext 构建记忆上下文，注入到系统提示词中
func (m *MemoryManager) BuildMemoryContext() string {
	memories := m.RecallMemories(20) // 最多注入 20 条记忆
	if len(memories) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n## 🧠 长期记忆\n")
	sb.WriteString("以下是你对用户的了解（来自之前的交互），请自然地运用这些信息：\n\n")

	// 按类别分组
	categories := map[string]string{
		"user_info":   "👤 用户信息",
		"preference":  "⚙️ 偏好设置",
		"project":     "📁 项目知识",
		"fact":        "📌 重要事实",
		"instruction": "📝 习惯指令",
	}

	grouped := make(map[string][]*MemoryEntry)
	for _, e := range memories {
		grouped[e.Category] = append(grouped[e.Category], e)
	}

	for cat, label := range categories {
		entries, ok := grouped[cat]
		if !ok {
			continue
		}
		sb.WriteString(fmt.Sprintf("### %s\n", label))
		for _, e := range entries {
			sb.WriteString(fmt.Sprintf("- %s\n", e.Content))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// BuildSessionSummary 构建最近会话的摘要上下文
func (m *MemoryManager) BuildSessionSummary() string {
	last := m.LoadLastSession()
	if last == nil {
		return ""
	}

	// 只显示最近会话中最后几轮对话作为"上次聊到哪"
	var sb strings.Builder
	sb.WriteString("\n## 📋 上次对话摘要\n")
	sb.WriteString(fmt.Sprintf("时间: %s | Agent: %s | 共 %d 轮对话\n\n",
		last.UpdatedAt.Format("2006-01-02 15:04"),
		last.Agent,
		last.TurnCount,
	))

	if last.Summary != "" {
		sb.WriteString(fmt.Sprintf("摘要: %s\n\n", last.Summary))
	}

	// 取最后 3 轮对话 (6条消息: user+assistant)
	msgs := last.Messages
	maxMsgs := 6
	if len(msgs) > maxMsgs {
		msgs = msgs[len(msgs)-maxMsgs:]
		sb.WriteString("(最近几轮对话)\n")
	}
	for _, msg := range msgs {
		if msg.Role == RoleUser {
			content := msg.Content
			if len(content) > 200 {
				content = content[:200] + "..."
			}
			sb.WriteString(fmt.Sprintf("用户: %s\n", content))
		} else if msg.Role == RoleAssistant {
			content := msg.Content
			if len(content) > 200 {
				content = content[:200] + "..."
			}
			sb.WriteString(fmt.Sprintf("助手: %s\n", content))
		}
	}

	return sb.String()
}

// GetSessionDir 获取会话存储目录
func (m *MemoryManager) GetSessionDir() string {
	return m.sessionDir
}

// GetMemDir 获取记忆根目录
func (m *MemoryManager) GetMemDir() string {
	return m.memDir
}

// Stats 返回记忆统计信息
func (m *MemoryManager) Stats() (sessions int, activeMemories int, expiredMemories int) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessions = len(m.listSessionFiles())
	for _, e := range m.longTerm.Entries {
		if e.Expired {
			expiredMemories++
		} else {
			activeMemories++
		}
	}
	return
}

// RestoreLastMessages 返回最近一次会话的可恢复消息列表 (不含 system prompt)
// 供 REPL 启动时注入
func (m *MemoryManager) RestoreLastMessages() []Message {
	last := m.LoadLastSession()
	if last == nil {
		return nil
	}

	// 检查是否是今天的会话，或者最近 24 小时内
	if time.Since(last.UpdatedAt) > 24*time.Hour {
		return nil // 超过 24 小时的会话不自动恢复
	}

	return last.Messages
}
