package ai

import (
	"fmt"
	"strings"
	"sync"

	"github.com/xiangjianhe-github/jiasinecli/internal/logger"
)

// Manager AI 统一管理器
// 管理所有配置的 AI 提供商，提供统一调用入口
type Manager struct {
	providers map[string]Provider   // name -> provider instance
	configs   map[string]ProviderConfig
	active    string                // 当前激活的提供商
	webSearch bool                  // 是否启用联网搜索
	mu        sync.RWMutex
}

// AIConfig AI 总配置
type AIConfig struct {
	Active    string                    `yaml:"active" mapstructure:"active"`           // 当前使用的提供商
	WebSearch bool                      `yaml:"web_search" mapstructure:"web_search"` // 默认启用联网搜索
	Providers map[string]ProviderConfig `yaml:"providers" mapstructure:"providers"`     // 各提供商配置
}

// NewManager 创建 AI 管理器
func NewManager(cfg AIConfig) *Manager {
	m := &Manager{
		providers: make(map[string]Provider),
		configs:   cfg.Providers,
		active:    cfg.Active,
		webSearch: cfg.WebSearch,
	}

	// 初始化所有已启用且配置了 API Key 的提供商
	for name, pcfg := range cfg.Providers {
		if !pcfg.Enabled {
			continue // 未启用，静默跳过
		}
		if pcfg.APIKey == "" {
			// 已启用但未配置 API Key — 仅记录 Debug（用户可能只配了其中一个）
			logger.Debug("AI 提供商已启用但未配置 API Key，跳过", "provider", name)
			continue
		}
		pcfg.Name = name // 确保 name 字段一致
		provider, err := NewProvider(pcfg)
		if err != nil {
			logger.Warn("初始化 AI 提供商失败", "provider", name, "error", err)
			continue
		}
		m.providers[strings.ToLower(name)] = provider
		logger.Debug("AI 提供商已加载", "provider", name, "model", provider.DefaultModel())
	}

	// 如果未指定 active，使用第一个可用的
	if m.active == "" {
		for name := range m.providers {
			m.active = name
			break
		}
	}

	return m
}

// TestConnection 发送最小化测试请求验证连接（不带 web search / tools）
func (m *Manager) TestConnection() (*ChatResponse, error) {
	provider, err := m.getProvider("")
	if err != nil {
		return nil, err
	}
	return provider.Chat(&ChatRequest{
		Messages:  []Message{{Role: RoleUser, Content: "ping"}},
		MaxTokens: 16,
	})
}

// Chat 使用当前激活的提供商发送聊天请求
func (m *Manager) Chat(prompt string) (*ChatResponse, error) {
	return m.ChatWith("", prompt)
}

// ChatWith 使用指定提供商发送聊天请求
func (m *Manager) ChatWith(providerName, prompt string) (*ChatResponse, error) {
	provider, err := m.getProvider(providerName)
	if err != nil {
		return nil, err
	}

	req := &ChatRequest{
		Messages: []Message{
			{Role: RoleUser, Content: prompt},
		},
		WebSearch: m.webSearch,
	}

	return provider.Chat(req)
}

// ChatWithSystem 使用系统提示词 + 用户消息进行聊天
func (m *Manager) ChatWithSystem(providerName, system, prompt string) (*ChatResponse, error) {
	provider, err := m.getProvider(providerName)
	if err != nil {
		return nil, err
	}

	req := &ChatRequest{
		Messages: []Message{
			{Role: RoleSystem, Content: system},
			{Role: RoleUser, Content: prompt},
		},
		WebSearch: m.webSearch,
	}

	return provider.Chat(req)
}

// ChatMessages 使用完整消息列表进行聊天
func (m *Manager) ChatMessages(providerName string, messages []Message) (*ChatResponse, error) {
	provider, err := m.getProvider(providerName)
	if err != nil {
		return nil, err
	}

	req := &ChatRequest{
		Messages:  messages,
		WebSearch: m.webSearch,
	}

	return provider.Chat(req)
}

// ChatMessagesWithTools 使用完整消息列表 + MCP 工具进行聊天
func (m *Manager) ChatMessagesWithTools(providerName string, messages []Message, tools []map[string]interface{}) (*ChatResponse, error) {
	provider, err := m.getProvider(providerName)
	if err != nil {
		return nil, err
	}

	req := &ChatRequest{
		Messages:  messages,
		WebSearch: m.webSearch,
		Tools:     tools,
	}

	return provider.Chat(req)
}

// ChatMessagesStream 流式聊天（无工具）
func (m *Manager) ChatMessagesStream(providerName string, messages []Message) (<-chan StreamChunk, error) {
	provider, err := m.getProvider(providerName)
	if err != nil {
		return nil, err
	}

	req := &ChatRequest{
		Messages:  messages,
		WebSearch: m.webSearch,
		Stream:    true,
	}

	return provider.ChatStream(req)
}

// ChatMessagesWithToolsStream 流式聊天 + MCP 工具
func (m *Manager) ChatMessagesWithToolsStream(providerName string, messages []Message, tools []map[string]interface{}) (<-chan StreamChunk, error) {
	provider, err := m.getProvider(providerName)
	if err != nil {
		return nil, err
	}

	req := &ChatRequest{
		Messages:  messages,
		WebSearch: m.webSearch,
		Tools:     tools,
		Stream:    true,
	}

	return provider.ChatStream(req)
}

// SetWebSearch 设置是否启用联网搜索
func (m *Manager) SetWebSearch(enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.webSearch = enabled
}

// IsWebSearch 返回是否启用联网搜索
func (m *Manager) IsWebSearch() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.webSearch
}

// ToggleWebSearch 切换联网搜索状态，返回切换后的状态
func (m *Manager) ToggleWebSearch() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.webSearch = !m.webSearch
	return m.webSearch
}

// ListProviders 列出所有已加载的提供商
func (m *Manager) ListProviders() []ProviderInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []ProviderInfo
	for name, p := range m.providers {
		result = append(result, ProviderInfo{
			Name:         p.Name(),
			Key:          name,
			Active:       name == strings.ToLower(m.active),
			DefaultModel: p.DefaultModel(),
			Models:       p.Models(),
		})
	}
	return result
}

// SetActive 切换当前激活的提供商
func (m *Manager) SetActive(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := strings.ToLower(name)
	if _, ok := m.providers[key]; !ok {
		return fmt.Errorf("提供商 '%s' 未加载 (可用: %s)", name, m.availableNames())
	}
	m.active = key
	return nil
}

// GetActive 返回当前激活的提供商名称
func (m *Manager) GetActive() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.active
}

// HasProviders 检查是否有已加载的提供商
func (m *Manager) HasProviders() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.providers) > 0
}

// ActiveProviderName 返回当前激活提供商的显示名称和模型
func (m *Manager) ActiveProviderInfo() (name, model string) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if p, ok := m.providers[m.active]; ok {
		return p.Name(), p.DefaultModel()
	}
	return m.active, ""
}

// ProviderInfo 提供商信息（用于展示）
type ProviderInfo struct {
	Name         string   `json:"name"`
	Key          string   `json:"key"`
	Active       bool     `json:"active"`
	DefaultModel string   `json:"default_model"`
	Models       []string `json:"models"`
}

func (m *Manager) getProvider(name string) (Provider, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if name == "" {
		name = m.active
	}
	key := strings.ToLower(name)

	provider, ok := m.providers[key]
	if !ok {
		if len(m.providers) == 0 {
			return nil, fmt.Errorf("未配置任何 AI 提供商，请在 config.yaml 的 ai.providers 中配置")
		}
		return nil, fmt.Errorf("提供商 '%s' 未加载 (可用: %s)", name, m.availableNames())
	}

	return provider, nil
}

func (m *Manager) availableNames() string {
	names := make([]string, 0, len(m.providers))
	for name := range m.providers {
		names = append(names, name)
	}
	return strings.Join(names, ", ")
}
