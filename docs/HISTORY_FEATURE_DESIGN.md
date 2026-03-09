# 历史对话查看功能 - 设计文档

## 需求分析

### 用户期望
> "期望可以点击打开之前的对话内容，能看到过往详细的对话记录，点击一次向上回滚一次聊天记录内容"

### 功能拆解
1. **历史记录存储**: 持久化保存所有对话
2. **快速检索**: 按时间、主题、关键词查找
3. **交互浏览**: 类似 shell history，使用快捷键滚动
4. **上下文恢复**: 点击历史对话可恢复上下文继续对话

---

## 架构设计

### 数据存储

#### SQLite 数据库结构
```sql
-- 会话表
CREATE TABLE sessions (
    id TEXT PRIMARY KEY,
    agent_name TEXT NOT NULL,
    provider TEXT NOT NULL,
    model TEXT NOT NULL,
    started_at DATETIME NOT NULL,
    ended_at DATETIME,
    message_count INTEGER DEFAULT 0,
    tags TEXT -- JSON array
);

-- 消息表
CREATE TABLE messages (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    role TEXT NOT NULL, -- user, assistant, system
    content TEXT NOT NULL,
    timestamp DATETIME NOT NULL,
    tokens INTEGER,
    metadata TEXT, -- JSON object
    FOREIGN KEY (session_id) REFERENCES sessions(id)
);

-- 附件表 (用于存储上下文文件、图片等)
CREATE TABLE attachments (
    id TEXT PRIMARY KEY,
    message_id TEXT NOT NULL,
    type TEXT NOT NULL, -- file, image, code
    path TEXT NOT NULL,
    content TEXT,
    FOREIGN KEY (message_id) REFERENCES messages(id)
);

-- 索引
CREATE INDEX idx_messages_session ON messages(session_id, timestamp);
CREATE INDEX idx_messages_content ON messages(content); -- FTS5 全文索引
CREATE INDEX idx_sessions_time ON sessions(started_at DESC);
```

### Go 数据结构
```go
// internal/history/history.go
package history

type HistoryManager struct {
    db *sql.DB
}

type Session struct {
    ID           string    `json:"id"`
    AgentName    string    `json:"agent_name"`
    Provider     string    `json:"provider"`
    Model        string    `json:"model"`
    StartedAt    time.Time `json:"started_at"`
    EndedAt      *time.Time `json:"ended_at"`
    MessageCount int       `json:"message_count"`
    Tags         []string  `json:"tags"`
}

type Message struct {
    ID        string                 `json:"id"`
    SessionID string                 `json:"session_id"`
    Role      string                 `json:"role"`
    Content   string                 `json:"content"`
    Timestamp time.Time              `json:"timestamp"`
    Tokens    int                    `json:"tokens"`
    Metadata  map[string]interface{} `json:"metadata"`
}

type HistoryQuery struct {
    SessionID string
    Limit     int
    Offset    int
    Keyword   string
    StartTime *time.Time
    EndTime   *time.Time
    Role      string
}
```

---

## 交互设计

### CLI 命令

#### 1. 查看会话列表
```bash
jiasinecli history sessions [--limit 10] [--agent <name>]

# 输出示例
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Recent Sessions
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
 ID       Started At           Messages  Tags
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
 abc123   2026-03-06 10:30     15        [code, debug]
 def456   2026-03-05 16:20     8         [api, design]
 ghi789   2026-03-04 09:15     23        [refactor]
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

#### 2. 查看会话详情
```bash
jiasinecli history show <session-id>

# 输出示例（使用 Markdown 渲染）
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Session abc123
  Started: 2026-03-06 10:30  •  Messages: 15
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

[10:30:15] 👤 User
如何实现单例模式？

[10:30:18] 🤖 Assistant
在 Go 中实现单例模式有几种方式：

**1. 使用 sync.Once**
```go
var instance *Singleton
var once sync.Once

func GetInstance() *Singleton {
    once.Do(func() {
        instance = &Singleton{}
    })
    return instance
}
```
...
```

#### 3. 搜索历史对话
```bash
jiasinecli history search "单例模式" [--limit 5]

# 输出示例
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Search Results for "单例模式"
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Session abc123 • 2026-03-06 10:30
  User: 如何实现单例模式？
  Assistant: 在 Go 中实现单例模式有几种方式...

  Session xyz999 • 2026-02-28 14:20
  User: 单例模式和工厂模式的区别
  Assistant: 两者都是创建型模式，主要区别在于...
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

#### 4. 恢复会话上下文
```bash
jiasinecli history resume <session-id>

# 进入 AI 对话模式，加载历史上下文
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Resuming Session abc123
  Loaded 15 messages from history
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

┃ ❯ [继续之前的对话...]
```

#### 5. 删除历史记录
```bash
jiasinecli history delete <session-id>
jiasinecli history clear --before "2026-01-01"
```

---

### 交互式浏览（类似 shell history）

#### 在 AI 对话模式中使用快捷键

```
进入 AI 模式:
jiasinecli ai

快捷键：
  ↑ (Up Arrow)     - 向上滚动，查看更早的消息
  ↓ (Down Arrow)   - 向下滚动，查看更新的消息
  Ctrl+R           - 搜索历史对话
  Ctrl+L           - 清屏
  /history         - 显示当前会话历史
  /sessions        - 列出所有会话
```

#### 实现方案

使用 `github.com/chzyer/readline` 库实现类似 bash 的 history 功能：

```go
import "github.com/chzyer/readline"

func enterAIInteractiveWithHistory(agentName string) error {
    // 配置 readline
    rl, err := readline.NewEx(&readline.Config{
        Prompt:          prompt,
        HistoryFile:     filepath.Join(configDir, ".ai_history"),
        InterruptPrompt: "^C",
        EOFPrompt:       "exit",

        // 历史搜索配置
        HistorySearchFold: true,
    })
    if err != nil {
        return err
    }
    defer rl.Close()

    // 加载历史会话
    historyMgr := history.NewManager(dbPath)
    sessions, _ := historyMgr.ListSessions(history.HistoryQuery{Limit: 100})

    // 预加载到 readline history
    for _, session := range sessions {
        messages, _ := historyMgr.GetMessages(session.ID)
        for _, msg := range messages {
            if msg.Role == "user" {
                rl.SaveHistory(msg.Content)
            }
        }
    }

    for {
        line, err := rl.Readline()
        if err != nil {
            break
        }

        // 处理输入...
    }
}
```

---

## UI/UX 增强

### 1. 美化历史记录显示

使用之前实现的 Markdown 渲染功能：

```go
func printMessage(msg history.Message) {
    // 时间戳
    timestamp := msg.Timestamp.Format("15:04:05")

    // 角色图标和颜色
    var icon, color string
    switch msg.Role {
    case "user":
        icon = "👤"
        color = banner.BrightCyan
    case "assistant":
        icon = "🤖"
        color = banner.BrightBlue
    case "system":
        icon = "⚙️"
        color = banner.Gray
    }

    // 输出
    fmt.Printf("\n%s[%s] %s%s %s%s%s\n",
        banner.Dim, timestamp, Reset,
        color, icon, msg.Role, banner.Reset)

    // Markdown 渲染内容
    rendered := render.Markdown(msg.Content)
    fmt.Println(rendered)
}
```

### 2. 分页显示

对于长会话，使用分页：

```bash
jiasinecli history show <session-id> --page 1 --per-page 10

# 底部导航
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Page 1 of 3  •  Use --page 2 for next page
  <  Prev  |  Next  >
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

### 3. 交互式选择

使用 `github.com/manifoldco/promptui` 实现交互式选择：

```go
import "github.com/manifoldco/promptui"

func selectSession() (*history.Session, error) {
    sessions, _ := historyMgr.ListSessions(history.HistoryQuery{Limit: 20})

    templates := &promptui.SelectTemplates{
        Label:    "{{ . }}",
        Active:   "▸ {{ .StartedAt.Format \"2006-01-02 15:04\" }} | {{ .MessageCount }} msgs | {{ .Tags }}",
        Inactive: "  {{ .StartedAt.Format \"2006-01-02 15:04\" }} | {{ .MessageCount }} msgs",
        Selected: "✓ {{ .StartedAt.Format \"2006-01-02 15:04\" }}",
    }

    prompt := promptui.Select{
        Label:     "Select a session",
        Items:     sessions,
        Templates: templates,
        Size:      10,
    }

    idx, _, err := prompt.Run()
    if err != nil {
        return nil, err
    }

    return &sessions[idx], nil
}
```

---

## 实施计划

### Phase 1: 基础存储 (2 周)
- [ ] SQLite 数据库设计和初始化
- [ ] HistoryManager 实现
- [ ] 基本的 CRUD 操作
- [ ] 单元测试

**依赖**: 无

### Phase 2: 命令行接口 (1 周)
- [ ] `history sessions` 命令
- [ ] `history show` 命令
- [ ] `history search` 命令
- [ ] `history delete` 命令

**依赖**: Phase 1

### Phase 3: 交互式浏览 (2 周)
- [ ] 集成 readline 库
- [ ] 实现 ↑/↓ 导航
- [ ] 实现 Ctrl+R 搜索
- [ ] 美化显示输出

**依赖**: Phase 1, Phase 2

### Phase 4: 上下文恢复 (1 周)
- [ ] `history resume` 命令
- [ ] 历史消息加载到 AI 上下文
- [ ] 会话状态恢复

**依赖**: Phase 1, Phase 2

### Phase 5: 高级功能 (2 周)
- [ ] 全文搜索 (SQLite FTS5)
- [ ] 智能标签生成
- [ ] 导出功能 (JSON/Markdown)
- [ ] 统计和分析

**依赖**: Phase 1-4

**总计**: ~8 周

---

## 技术选型

### 依赖库
```go
// 数据库
"database/sql"
"github.com/mattn/go-sqlite3"

// 交互式输入
"github.com/chzyer/readline"
"github.com/manifoldco/promptui"

// 时间处理
"time"

// JSON
"encoding/json"
```

### 性能优化
1. **索引**: 在 `timestamp`, `content`, `session_id` 上建索引
2. **分页**: 避免一次加载过多数据
3. **缓存**: 热门会话缓存在内存
4. **批量插入**: 使用事务批量写入

---

## 测试策略

### 单元测试
```go
func TestHistoryManager_SaveMessage(t *testing.T) {
    mgr := setupTestDB(t)
    defer cleanupTestDB(t, mgr)

    msg := &history.Message{
        ID:        "msg1",
        SessionID: "sess1",
        Role:      "user",
        Content:   "Hello",
        Timestamp: time.Now(),
    }

    err := mgr.SaveMessage(msg)
    assert.NoError(t, err)

    retrieved, err := mgr.GetMessage("msg1")
    assert.NoError(t, err)
    assert.Equal(t, msg.Content, retrieved.Content)
}
```

### 集成测试
```bash
# 创建测试会话
jiasinecli ai --test-mode <<EOF
你好
什么是依赖注入？
谢谢
exit
EOF

# 验证历史记录
jiasinecli history sessions | grep "3 msgs"

# 搜索测试
jiasinecli history search "依赖注入" | grep "什么是依赖注入"
```

---

## 安全和隐私

### 数据保护
1. **本地存储**: 数据仅存储在本地 SQLite，不上传
2. **敏感信息**: 支持排除敏感内容 (`--no-history` flag)
3. **加密**: 可选的数据库加密 (SQLCipher)
4. **清理**: 定期清理旧数据

### 配置选项
```yaml
# ~/.jiasinecli/config.yaml
history:
  enabled: true
  database: ~/.jiasinecli/history.db
  max_sessions: 1000
  max_age_days: 90
  auto_cleanup: true
  exclude_patterns:
    - "password"
    - "api_key"
    - "secret"
```

---

## 文档和帮助

### 用户文档
创建 `docs/HISTORY_GUIDE.md`：
- 快速开始
- 命令参考
- 快捷键说明
- 常见问题

### 内置帮助
```bash
jiasinecli history --help
jiasinecli history sessions --help
jiasinecli history search --help
```

---

## 下一步

完成历史对话查看功能后，可以扩展为：
1. **语义搜索**: 集成向量数据库，支持语义相似度搜索
2. **智能摘要**: 自动生成会话摘要
3. **导出分享**: 导出为 Markdown/PDF，分享给团队
4. **数据分析**: 统计常见问题、使用模式

---

**预计版本**: v0.2.0
**目标发布日期**: 2026-04-15
