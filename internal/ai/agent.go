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
type Agent struct {
	Name        string   `json:"name" yaml:"name"`
	Description string   `json:"description" yaml:"description"`
	Provider    string   `json:"provider" yaml:"provider"`       // 使用的 AI 提供商
	Model       string   `json:"model" yaml:"model"`             // 使用的模型
	System      string   `json:"system" yaml:"system"`           // 系统提示词
	Skills      []string `json:"skills" yaml:"skills"`           // 关联的 Skills
	MaxTurns    int      `json:"max_turns" yaml:"max_turns"`     // 最大对话轮次
	Temperature float64  `json:"temperature" yaml:"temperature"` // 温度参数
}

// AgentConfig Agent 在配置文件中的结构
type AgentConfig struct {
	Dir    string           `yaml:"dir" mapstructure:"dir"`       // Agent 配置目录
	Agents map[string]Agent `yaml:"agents" mapstructure:"agents"` // 内置 Agent
}

// AgentManager Agent 管理器
type AgentManager struct {
	agents  map[string]*Agent
	aiMgr   *Manager
	skillMgr *SkillManager
}

// NewAgentManager 创建 Agent 管理器
func NewAgentManager(aiMgr *Manager, skillMgr *SkillManager, cfg AgentConfig) *AgentManager {
	mgr := &AgentManager{
		agents:   make(map[string]*Agent),
		aiMgr:    aiMgr,
		skillMgr: skillMgr,
	}

	// 加载配置文件中的 Agent
	for name, agent := range cfg.Agents {
		a := agent // copy
		a.Name = name
		mgr.agents[name] = &a
		logger.Debug("Agent 已加载", "name", name)
	}

	// 从 Agent 目录加载
	if cfg.Dir != "" {
		mgr.loadFromDir(cfg.Dir)
	}

	// 注册内置 Agent
	mgr.registerBuiltinAgents()

	return mgr
}

// Run 运行指定 Agent（单轮对话）
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

	// 选择提供商
	providerName := agent.Provider
	if providerName == "" {
		providerName = m.aiMgr.GetActive()
	}

	provider, err := m.aiMgr.getProvider(providerName)
	if err != nil {
		return nil, err
	}

	req := &ChatRequest{
		Model: agent.Model,
		Messages: []Message{
			{Role: RoleSystem, Content: system},
			{Role: RoleUser, Content: prompt},
		},
		Temperature: agent.Temperature,
	}

	return provider.Chat(req)
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
		"assistant": {
			Name:        "assistant",
			Description: "通用 AI 助手 — 回答问题、写作、翻译",
			System:      "你是 Jiasine CLI 的内置 AI 助手。请简洁、准确地回答用户问题。如果用户使用中文提问，请用中文回答。",
			MaxTurns:    20,
			Temperature: 0.7,
		},
		"coder": {
			Name:        "coder",
			Description: "编程助手 — 代码生成、调试、重构",
			System: `你是一个专业的编程助手。请遵循以下原则：
1. 代码简洁清晰，有必要的注释
2. 遵循语言最佳实践
3. 先分析问题再给出方案
4. 给出可直接运行的代码`,
			MaxTurns:    30,
			Temperature: 0.3,
		},
		"translator": {
			Name:        "translator",
			Description: "翻译助手 — 多语言互译",
			System:      "你是一个专业的翻译助手。保持原文的语气和风格，翻译结果自然流畅。如果无法确定目标语言，中文翻译为英文，其他语言翻译为中文。",
			MaxTurns:    20,
			Temperature: 0.3,
		},
		"devops": {
			Name:        "devops",
			Description: "运维助手 — 部署、监控、故障排查",
			System: `你是一个 DevOps/SRE 专家助手。擅长：
- 容器化 (Docker/Kubernetes)
- CI/CD 管道
- 基础设施即代码 (Terraform/Ansible)
- 监控告警 (Prometheus/Grafana)
- 故障诊断与性能优化
请给出安全、可靠的运维建议。`,
			MaxTurns:    20,
			Temperature: 0.5,
		},
	}

	for name, agent := range builtins {
		if _, exists := m.agents[name]; !exists {
			m.agents[name] = agent
		}
	}
}

// loadFromDir 从目录加载 Agent JSON 配置
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
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
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
