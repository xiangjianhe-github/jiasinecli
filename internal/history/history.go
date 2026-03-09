// Package history 提供对话历史记录的持久化存储和管理
package history

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// Manager 历史记录管理器
type Manager struct {
	db *sql.DB
}

// Session 对话会话
type Session struct {
	ID           string     `json:"id"`
	Title        string     `json:"title"`        // 对话主题（第一条用户消息）
	AgentName    string     `json:"agent_name"`
	Provider     string     `json:"provider"`
	Model        string     `json:"model"`
	StartedAt    time.Time  `json:"started_at"`
	EndedAt      *time.Time `json:"ended_at"`
	MessageCount int        `json:"message_count"`
	Tags         []string   `json:"tags"`
}

// Message 对话消息
type Message struct {
	ID        string                 `json:"id"`
	SessionID string                 `json:"session_id"`
	Role      string                 `json:"role"` // user, assistant, system
	Content   string                 `json:"content"`
	Timestamp time.Time              `json:"timestamp"`
	Tokens    int                    `json:"tokens"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// Query 历史查询条件
type Query struct {
	SessionID string
	Limit     int
	Offset    int
	Keyword   string
	StartTime *time.Time
	EndTime   *time.Time
	Role      string
	AgentName string
}

// NewManager 创建历史记录管理器
func NewManager(dbPath string) (*Manager, error) {
	// 确保目录存在
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("创建数据库目录失败: %w", err)
	}

	// 打开数据库
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}

	mgr := &Manager{db: db}

	// 初始化数据库表
	if err := mgr.initTables(); err != nil {
		db.Close()
		return nil, fmt.Errorf("初始化数据库表失败: %w", err)
	}

	return mgr, nil
}

// initTables 初始化数据库表
func (m *Manager) initTables() error {
	schema := `
	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		title TEXT DEFAULT '',
		agent_name TEXT NOT NULL,
		provider TEXT NOT NULL,
		model TEXT NOT NULL,
		started_at DATETIME NOT NULL,
		ended_at DATETIME,
		message_count INTEGER DEFAULT 0,
		tags TEXT
	);

	CREATE TABLE IF NOT EXISTS messages (
		id TEXT PRIMARY KEY,
		session_id TEXT NOT NULL,
		role TEXT NOT NULL,
		content TEXT NOT NULL,
		timestamp DATETIME NOT NULL,
		tokens INTEGER DEFAULT 0,
		metadata TEXT,
		FOREIGN KEY (session_id) REFERENCES sessions(id)
	);

	CREATE INDEX IF NOT EXISTS idx_messages_session ON messages(session_id, timestamp);
	CREATE INDEX IF NOT EXISTS idx_sessions_time ON sessions(started_at DESC);
	`

	if _, err := m.db.Exec(schema); err != nil {
		return err
	}

	// 迁移：如果 title 列不存在，添加它（兼容旧数据库）
	_, _ = m.db.Exec(`ALTER TABLE sessions ADD COLUMN title TEXT DEFAULT ''`)

	return nil
}

// CreateSession 创建新会话
// 如果 session.ID 为空，自动生成唯一 ID（时间戳 + 随机后缀）
func (m *Manager) CreateSession(session *Session) error {
	if session.ID == "" {
		b := make([]byte, 4)
		rand.Read(b)
		session.ID = time.Now().Format("20060102-150405") + "-" + hex.EncodeToString(b)
	}
	if session.StartedAt.IsZero() {
		session.StartedAt = time.Now()
	}

	tagsJSON, _ := json.Marshal(session.Tags)

	_, err := m.db.Exec(`
		INSERT INTO sessions (id, title, agent_name, provider, model, started_at, tags)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, session.ID, session.Title, session.AgentName, session.Provider, session.Model, session.StartedAt, string(tagsJSON))

	return err
}

// EndSession 结束会话
func (m *Manager) EndSession(sessionID string) error {
	now := time.Now()
	_, err := m.db.Exec(`
		UPDATE sessions SET ended_at = ? WHERE id = ?
	`, now, sessionID)
	return err
}

// UpdateSessionTitle 更新会话标题（通常是第一条用户消息）
func (m *Manager) UpdateSessionTitle(sessionID, title string) error {
	// 限制标题长度为 100 字符
	if len(title) > 100 {
		title = title[:100] + "..."
	}

	_, err := m.db.Exec(`
		UPDATE sessions SET title = ? WHERE id = ?
	`, title, sessionID)
	return err
}

// SaveMessage 保存消息
func (m *Manager) SaveMessage(msg *Message) error {
	metadataJSON, _ := json.Marshal(msg.Metadata)

	_, err := m.db.Exec(`
		INSERT INTO messages (id, session_id, role, content, timestamp, tokens, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, msg.ID, msg.SessionID, msg.Role, msg.Content, msg.Timestamp, msg.Tokens, string(metadataJSON))

	if err != nil {
		return err
	}

	// 更新会话消息计数
	_, err = m.db.Exec(`
		UPDATE sessions SET message_count = message_count + 1 WHERE id = ?
	`, msg.SessionID)

	return err
}

// GetSession 获取会话详情
func (m *Manager) GetSession(sessionID string) (*Session, error) {
	var session Session
	var tagsJSON string

	err := m.db.QueryRow(`
		SELECT id, title, agent_name, provider, model, started_at, ended_at, message_count, tags
		FROM sessions WHERE id = ?
	`, sessionID).Scan(
		&session.ID, &session.Title, &session.AgentName, &session.Provider, &session.Model,
		&session.StartedAt, &session.EndedAt, &session.MessageCount, &tagsJSON,
	)

	if err != nil {
		return nil, err
	}

	json.Unmarshal([]byte(tagsJSON), &session.Tags)
	return &session, nil
}

// ListSessions 列出会话
func (m *Manager) ListSessions(query Query) ([]Session, error) {
	if query.Limit == 0 {
		query.Limit = 10
	}

	sqlQuery := `SELECT id, title, agent_name, provider, model, started_at, ended_at, message_count, tags
		FROM sessions WHERE 1=1`
	args := []interface{}{}

	if query.AgentName != "" {
		sqlQuery += " AND agent_name = ?"
		args = append(args, query.AgentName)
	}

	if query.StartTime != nil {
		sqlQuery += " AND started_at >= ?"
		args = append(args, *query.StartTime)
	}

	if query.EndTime != nil {
		sqlQuery += " AND started_at <= ?"
		args = append(args, *query.EndTime)
	}

	sqlQuery += " ORDER BY started_at DESC LIMIT ? OFFSET ?"
	args = append(args, query.Limit, query.Offset)

	rows, err := m.db.Query(sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var session Session
		var tagsJSON string

		err := rows.Scan(
			&session.ID, &session.Title, &session.AgentName, &session.Provider, &session.Model,
			&session.StartedAt, &session.EndedAt, &session.MessageCount, &tagsJSON,
		)
		if err != nil {
			continue
		}

		json.Unmarshal([]byte(tagsJSON), &session.Tags)
		sessions = append(sessions, session)
	}

	return sessions, nil
}

// GetMessages 获取会话的所有消息
func (m *Manager) GetMessages(sessionID string) ([]Message, error) {
	rows, err := m.db.Query(`
		SELECT id, session_id, role, content, timestamp, tokens, metadata
		FROM messages WHERE session_id = ? ORDER BY timestamp ASC
	`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		var metadataJSON string

		err := rows.Scan(
			&msg.ID, &msg.SessionID, &msg.Role, &msg.Content,
			&msg.Timestamp, &msg.Tokens, &metadataJSON,
		)
		if err != nil {
			continue
		}

		json.Unmarshal([]byte(metadataJSON), &msg.Metadata)
		messages = append(messages, msg)
	}

	return messages, nil
}

// SearchMessages 搜索消息
func (m *Manager) SearchMessages(query Query) ([]Message, error) {
	if query.Limit == 0 {
		query.Limit = 10
	}

	sqlQuery := `SELECT id, session_id, role, content, timestamp, tokens, metadata
		FROM messages WHERE 1=1`
	args := []interface{}{}

	if query.SessionID != "" {
		sqlQuery += " AND session_id = ?"
		args = append(args, query.SessionID)
	}

	if query.Role != "" {
		sqlQuery += " AND role = ?"
		args = append(args, query.Role)
	}

	if query.Keyword != "" {
		sqlQuery += " AND content LIKE ?"
		args = append(args, "%"+query.Keyword+"%")
	}

	sqlQuery += " ORDER BY timestamp DESC LIMIT ? OFFSET ?"
	args = append(args, query.Limit, query.Offset)

	rows, err := m.db.Query(sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		var metadataJSON string

		err := rows.Scan(
			&msg.ID, &msg.SessionID, &msg.Role, &msg.Content,
			&msg.Timestamp, &msg.Tokens, &metadataJSON,
		)
		if err != nil {
			continue
		}

		json.Unmarshal([]byte(metadataJSON), &msg.Metadata)
		messages = append(messages, msg)
	}

	return messages, nil
}

// DeleteSession 删除会话及其所有消息
func (m *Manager) DeleteSession(sessionID string) error {
	tx, err := m.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 删除消息
	_, err = tx.Exec("DELETE FROM messages WHERE session_id = ?", sessionID)
	if err != nil {
		return err
	}

	// 删除会话
	_, err = tx.Exec("DELETE FROM sessions WHERE id = ?", sessionID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// DeleteOldSessions 删除指定日期之前的会话
func (m *Manager) DeleteOldSessions(before time.Time) (int64, error) {
	// 获取要删除的会话 ID
	rows, err := m.db.Query("SELECT id FROM sessions WHERE started_at < ?", before)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var sessionIDs []string
	for rows.Next() {
		var id string
		rows.Scan(&id)
		sessionIDs = append(sessionIDs, id)
	}

	if len(sessionIDs) == 0 {
		return 0, nil
	}

	tx, err := m.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	// 删除消息
	for _, id := range sessionIDs {
		_, err = tx.Exec("DELETE FROM messages WHERE session_id = ?", id)
		if err != nil {
			return 0, err
		}
	}

	// 删除会话
	result, err := tx.Exec("DELETE FROM sessions WHERE started_at < ?", before)
	if err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// Close 关闭数据库连接
func (m *Manager) Close() error {
	return m.db.Close()
}

// GetStats 获取统计信息
func (m *Manager) GetStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// 总会话数
	var totalSessions int
	err := m.db.QueryRow("SELECT COUNT(*) FROM sessions").Scan(&totalSessions)
	if err != nil {
		return nil, err
	}
	stats["total_sessions"] = totalSessions

	// 总消息数
	var totalMessages int
	err = m.db.QueryRow("SELECT COUNT(*) FROM messages").Scan(&totalMessages)
	if err != nil {
		return nil, err
	}
	stats["total_messages"] = totalMessages

	// 最近会话
	var lastSession time.Time
	err = m.db.QueryRow("SELECT MAX(started_at) FROM sessions").Scan(&lastSession)
	if err == nil {
		stats["last_session"] = lastSession
	}

	return stats, nil
}
