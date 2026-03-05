package ai

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xiangjianhe-github/jiasinecli/internal/logger"
)

// Agent AI 智能体
// 封装了系统提示词、技能调度、上下文管理
// 支持 MCP (Model Context Protocol) 协议
type Agent struct {
	Name        string   `json:"name" yaml:"name"`
	Description string   `json:"description" yaml:"description"`
	Provider    string   `json:"provider" yaml:"provider"`       // 使用的 AI 提供商
	Model       string   `json:"model" yaml:"model"`             // 使用的模型
	System      string   `json:"system" yaml:"system"`           // 系统提示词
	Skills      []string `json:"skills" yaml:"skills"`           // 关联的 Skills
	MaxTurns    int      `json:"max_turns" yaml:"max_turns"`     // 最大对话轮次
	Temperature float64  `json:"temperature" yaml:"temperature"` // 温度参数
	// MCP 协议支持
	MCP *MCPConfig `json:"mcp,omitempty" yaml:"mcp,omitempty"` // MCP 工具/资源配置
}

// AgentConfig Agent 在配置文件中的结构
type AgentConfig struct {
	Dir    string           `yaml:"dir" mapstructure:"dir"`       // Agent 配置目录
	Agents map[string]Agent `yaml:"agents" mapstructure:"agents"` // 内置 Agent
}

// AgentManager Agent 管理器
type AgentManager struct {
	agents   map[string]*Agent
	aiMgr    *Manager
	skillMgr *SkillManager
	agentDir string
}

// NewAgentManager 创建 Agent 管理器
func NewAgentManager(aiMgr *Manager, skillMgr *SkillManager, cfg AgentConfig) *AgentManager {
	agentDir := cfg.Dir
	if agentDir == "" {
		if home, err := os.UserHomeDir(); err == nil {
			agentDir = filepath.Join(home, ".jiasine", "agents")
		}
	}

	mgr := &AgentManager{
		agents:   make(map[string]*Agent),
		aiMgr:    aiMgr,
		skillMgr: skillMgr,
		agentDir: agentDir,
	}

	// 加载配置文件中的 Agent
	for name, agent := range cfg.Agents {
		a := agent // copy
		a.Name = name
		mgr.agents[name] = &a
		logger.Debug("Agent 已加载", "name", name)
	}

	// 从 Agent 目录加载
	if mgr.agentDir != "" {
		mgr.loadFromDir(mgr.agentDir)
	}

	// 注册内置 Agent
	mgr.registerBuiltinAgents()

	// 将内置 Agent 写入磁盘（如目录下不存在）
	mgr.ensureDefaults(mgr.agentDir)

	return mgr
}

// AIManager 返回内部持有的 AI 管理器
func (m *AgentManager) AIManager() *Manager {
	return m.aiMgr
}

// Run 运行指定 Agent（单轮对话，支持 MCP 工具调用循环）
func (m *AgentManager) Run(agentName, prompt string) (*ChatResponse, error) {
	agent, ok := m.agents[agentName]
	if !ok {
		return nil, fmt.Errorf("Agent '%s' 不存在 (可用: %s)", agentName, m.availableNames())
	}

	// 构建系统提示词：Agent 自身 + 关联的 Skills
	system := agent.System
	if len(agent.Skills) > 0 && m.skillMgr != nil {
		skillContext := m.skillMgr.BuildContext(agent.Skills)
		if skillContext != "" {
			system += "\n\n## 可用技能\n" + skillContext
		}
	}

	// 收集 MCP 工具定义
	var mcpTools []map[string]interface{}
	if m.skillMgr != nil && len(agent.Skills) > 0 {
		var skills []*Skill
		for _, name := range agent.Skills {
			if s, err := m.skillMgr.Get(name); err == nil {
				skills = append(skills, s)
			}
		}
		mcpTools = MCPToolDefs(skills)
	}

	// 选择提供商
	providerName := agent.Provider
	if providerName == "" {
		providerName = m.aiMgr.GetActive()
	}

	provider, err := m.aiMgr.getProvider(providerName)
	if err != nil {
		return nil, err
	}

	// 构建初始请求
	messages := []Message{
		{Role: RoleSystem, Content: system},
		{Role: RoleUser, Content: prompt},
	}

	executor := NewToolExecutor()
	maxToolLoops := 10
	var totalResponse *ChatResponse

	for i := 0; i < maxToolLoops; i++ {
		req := &ChatRequest{
			Model:       agent.Model,
			Messages:    messages,
			Temperature: agent.Temperature,
			WebSearch:   m.aiMgr.IsWebSearch(),
			Tools:       mcpTools,
		}

		resp, err := provider.Chat(req)
		if err != nil {
			return nil, err
		}

		// 累加 tokens
		if totalResponse == nil {
			totalResponse = resp
		} else {
			totalResponse.Content += resp.Content
			totalResponse.PromptTokens += resp.PromptTokens
			totalResponse.OutputTokens += resp.OutputTokens
			totalResponse.TotalTokens += resp.TotalTokens
		}

		// 如果没有工具调用，直接返回
		if len(resp.ToolCalls) == 0 || resp.StopReason != "tool_use" {
			totalResponse.Content = resp.Content
			return totalResponse, nil
		}

		// 将助手的 tool_use 响应原始内容存入历史
		assistantContent := BuildAssistantToolUseContent(resp)
		messages = append(messages, Message{
			Role:    RoleAssistantToolUse,
			Content: assistantContent,
		})

		// 执行工具调用
		var toolResults []map[string]interface{}
		for _, call := range resp.ToolCalls {
			logger.Info("🔧 执行工具", "tool", call.Name, "id", call.ID)
			result := executor.Execute(call)
			toolResults = append(toolResults, map[string]interface{}{
				"type":        "tool_result",
				"tool_use_id": result.ToolUseID,
				"content":     result.Content,
			})
		}

		// 将工具结果作为 user 消息发回
		toolResultJSON, _ := json.Marshal(toolResults)
		messages = append(messages, Message{
			Role:    RoleToolResult,
			Content: string(toolResultJSON),
		})
	}

	return totalResponse, nil
}

// BuildAssistantToolUseContent 将 AI 响应中的 text + tool_use 块序列化
// 用于回传给 API 保持对话一致性
func BuildAssistantToolUseContent(resp *ChatResponse) string {
	return BuildAssistantToolUseBlocks(resp.Content, resp.ToolCalls)
}

// BuildAssistantToolUseBlocks 从文本内容 + 工具调用列表构建 assistant 内容块 JSON
// 用于流式模式下将累积结果回传给 API
func BuildAssistantToolUseBlocks(content string, toolCalls []ToolCall) string {
	var blocks []interface{}
	if content != "" {
		blocks = append(blocks, map[string]interface{}{
			"type": "text",
			"text": content,
		})
	}
	for _, tc := range toolCalls {
		blocks = append(blocks, map[string]interface{}{
			"type":  "tool_use",
			"id":    tc.ID,
			"name":  tc.Name,
			"input": tc.Input,
		})
	}
	data, _ := json.Marshal(blocks)
	return string(data)
}

// GetSystemPrompt 获取指定 Agent 的完整系统提示词（含 Skills 上下文）
func (m *AgentManager) GetSystemPrompt(name string) (string, error) {
	key := strings.ToLower(name)
	agent, ok := m.agents[key]
	if !ok {
		return "", fmt.Errorf("Agent '%s' 不存在", name)
	}
	system := agent.System
	if m.skillMgr != nil && len(agent.Skills) > 0 {
		ctx := m.skillMgr.BuildContext(agent.Skills)
		if ctx != "" {
			system += "\n\n" + ctx
		}
	}
	return system, nil
}

// List 列出所有 Agent
func (m *AgentManager) List() []AgentInfo {
	var result []AgentInfo
	for name, a := range m.agents {
		result = append(result, AgentInfo{
			Name:        name,
			Description: a.Description,
			Provider:    a.Provider,
			Model:       a.Model,
			Skills:      a.Skills,
		})
	}
	return result
}

// AgentInfo Agent 展示信息
type AgentInfo struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Provider    string   `json:"provider"`
	Model       string   `json:"model"`
	Skills      []string `json:"skills"`
}

// registerBuiltinAgents 注册内置 Agent
func (m *AgentManager) registerBuiltinAgents() {
	builtins := map[string]*Agent{
		"general": {
			Name:        "general",
			Description: "通用 AI 智能体 — 问答、编码、翻译、写作",
			System: `你是 Jiasine CLI 的通用 AI 智能体，具备广泛的能力：
- 回答各类问题，提供专业建议
- 编写、调试、重构代码
- 多语言翻译
- 文档写作与格式化
- DevOps 运维指导

请根据用户需求灵活应对，保持简洁准确。如果用户使用中文提问，请用中文回答。
你可以调用可用的 Skills 来增强回答质量。`,
			Skills:      []string{"prompt-analysis", "ask", "git-ai-search"},
			MaxTurns:    30,
			Temperature: 0.7,
			MCP: &MCPConfig{
				Tools: []MCPTool{
					{
						Name:        "run_skill",
						Description: "调用已安装的 Skill 来处理特定任务",
						InputSchema: map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"skill_name": map[string]interface{}{"type": "string", "description": "要调用的 Skill 名称"},
								"input":      map[string]interface{}{"type": "string", "description": "传递给 Skill 的输入"},
							},
							"required": []string{"skill_name", "input"},
						},
					},
				},
				Transport: "stdio",
			},
		},
		"explore": {
			Name:        "explore",
			Description: "代码探索智能体 — 代码库分析、架构理解、变更追踪",
			System: `你是 Jiasine CLI 的代码探索智能体，专注于帮助用户理解代码库：

核心能力：
1. 代码库导航 — 搜索、阅读、分析代码结构
2. 架构理解 — 解释模块依赖、设计模式、数据流
3. 变更追踪 — 通过 git 历史理解代码演进
4. 上下文恢复 — 从 git 提交中恢复 AI 对话上下文

工作原则：
- 先搜索、后分析、再总结
- 引用具体文件路径和行号
- 解释 "为什么" 而非仅仅 "是什么"
- 优先使用 ask 和 git-ai-search 技能`,
			Skills:      []string{"ask", "git-ai-search"},
			MaxTurns:    50,
			Temperature: 0.2,
			MCP: &MCPConfig{
				Tools: []MCPTool{
					{
						Name:        "read_code",
						Description: "读取指定文件的代码内容",
						InputSchema: map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"file_path":  map[string]interface{}{"type": "string", "description": "文件路径"},
								"start_line": map[string]interface{}{"type": "integer", "description": "起始行号（可选）"},
								"end_line":   map[string]interface{}{"type": "integer", "description": "结束行号（可选）"},
							},
							"required": []string{"file_path"},
						},
					},
					{
						Name:        "search_code",
						Description: "在代码库中搜索关键字或正则模式",
						InputSchema: map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"query":    map[string]interface{}{"type": "string", "description": "搜索关键字或正则"},
								"is_regex": map[string]interface{}{"type": "boolean", "description": "是否为正则表达式"},
								"path":     map[string]interface{}{"type": "string", "description": "限定搜索路径（可选）"},
							},
							"required": []string{"query"},
						},
					},
					{
						Name:        "git_log",
						Description: "查看 git 提交历史",
						InputSchema: map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"path":   map[string]interface{}{"type": "string", "description": "文件/目录路径（可选）"},
								"author": map[string]interface{}{"type": "string", "description": "作者（可选）"},
								"limit":  map[string]interface{}{"type": "integer", "description": "条数限制"},
							},
						},
					},
				},
				Resources: []MCPResource{
					{
						URI:         "workspace://",
						Name:        "workspace",
						Description: "当前工作区的代码文件",
						MimeType:    "text/plain",
					},
					{
						URI:         "git://history",
						Name:        "git-history",
						Description: "Git 提交历史",
						MimeType:    "application/json",
					},
				},
				Transport: "stdio",
			},
		},
	}

	for name, agent := range builtins {
		if _, exists := m.agents[name]; !exists {
			m.agents[name] = agent
		}
	}
}

// ensureDefaults 将内置 Agent 定义写入磁盘（仅当对应目录/文件不存在时）
func (m *AgentManager) ensureDefaults(dir string) {
	if dir == "" {
		return
	}
	os.MkdirAll(dir, 0755)

	for name, agent := range m.agents {
		dirPath := filepath.Join(dir, name)
		jsonPath := filepath.Join(dirPath, "agent.json")
		legacyPath := filepath.Join(dir, name+".json")

		if _, err := os.Stat(dirPath); err == nil {
			continue
		}
		if _, err := os.Stat(legacyPath); err == nil {
			continue
		}

		os.MkdirAll(dirPath, 0755)
		data, err := json.MarshalIndent(agent, "", "  ")
		if err != nil {
			continue
		}
		os.WriteFile(jsonPath, data, 0644)
		logger.Debug("写入默认 Agent 定义", "name", name, "path", jsonPath)
	}
}

// Install 安装 Agent（从 JSON 文件或目录）
func (m *AgentManager) Install(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("路径不存在: %s", path)
	}

	var agent Agent
	if info.IsDir() {
		// 目录格式 — 查找 agent.json
		jsonPath := filepath.Join(path, "agent.json")
		data, err := os.ReadFile(jsonPath)
		if err != nil {
			return fmt.Errorf("目录中未找到 agent.json: %s", path)
		}
		if err := json.Unmarshal(data, &agent); err != nil {
			return fmt.Errorf("解析 agent.json 失败: %w", err)
		}
		if agent.Name == "" {
			agent.Name = filepath.Base(path)
		}
		// 复制整个目录到 agents/<name>/
		homeDir, _ := os.UserHomeDir()
		destDir := filepath.Join(homeDir, ".jiasine", "agents", agent.Name)
		if err := copyDir(path, destDir); err != nil {
			return fmt.Errorf("复制 Agent 目录失败: %w", err)
		}
	} else {
		// 单文件 JSON 格式
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("读取文件失败: %w", err)
		}
		if err := json.Unmarshal(data, &agent); err != nil {
			return fmt.Errorf("JSON 解析失败: %w", err)
		}

		if agent.Name == "" {
			agent.Name = strings.TrimSuffix(filepath.Base(path), ".json")
		}

		// 保存到 agents/<name>/agent.json 目录格式
		homeDir, _ := os.UserHomeDir()
		destDir := filepath.Join(homeDir, ".jiasine", "agents", agent.Name)
		os.MkdirAll(destDir, 0755)
		destPath := filepath.Join(destDir, "agent.json")
		if err := os.WriteFile(destPath, data, 0644); err != nil {
			return fmt.Errorf("写入失败: %w", err)
		}
	}

	m.agents[agent.Name] = &agent
	logger.Info("Agent 已安装", "name", agent.Name)
	return nil
}

// Remove 卸载 Agent
func (m *AgentManager) Remove(name string) error {
	if _, ok := m.agents[name]; !ok {
		return fmt.Errorf("Agent '%s' 不存在", name)
	}

	delete(m.agents, name)

	homeDir, _ := os.UserHomeDir()

	// 移除目录格式
	dirPath := filepath.Join(homeDir, ".jiasine", "agents", name)
	if _, err := os.Stat(dirPath); err == nil {
		os.RemoveAll(dirPath)
	}

	// 移除旧的单文件格式
	filePath := filepath.Join(homeDir, ".jiasine", "agents", name+".json")
	os.Remove(filePath)

	logger.Info("Agent 已卸载", "name", name)
	return nil
}

// loadFromDir 从目录加载 Agent 配置
// 支持两种格式：
//  1. <dir>/<name>.json — 单文件 JSON 格式
//  2. <dir>/<name>/agent.json — 子目录格式
func (m *AgentManager) loadFromDir(dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		logger.Warn("读取 Agent 目录失败", "dir", dir, "error", err)
		return
	}

	for _, entry := range entries {
		// 子目录格式 — 查找 agent.json
		if entry.IsDir() {
			name := entry.Name()
			jsonPath := filepath.Join(dir, name, "agent.json")
			if data, err := os.ReadFile(jsonPath); err == nil {
				var agent Agent
				if err := json.Unmarshal(data, &agent); err == nil {
					if agent.Name == "" {
						agent.Name = name
					}
					m.agents[name] = &agent
					logger.Debug("从目录加载 Agent", "name", name)
				}
			}
			continue
		}

		// 单文件 JSON 格式
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			logger.Warn("读取 Agent 文件失败", "path", path, "error", err)
			continue
		}

		var agent Agent
		if err := json.Unmarshal(data, &agent); err != nil {
			logger.Warn("解析 Agent 文件失败", "path", path, "error", err)
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".json")
		if agent.Name == "" {
			agent.Name = name
		}
		m.agents[name] = &agent
		logger.Debug("从文件加载 Agent", "name", name, "path", path)
	}
}

func (m *AgentManager) availableNames() string {
	names := make([]string, 0, len(m.agents))
	for name := range m.agents {
		names = append(names, name)
	}
	return strings.Join(names, ", ")
}
