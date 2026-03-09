package history

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
)

func setupTestDB(t *testing.T) *Manager {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_history.db")

	mgr, err := NewManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	return mgr
}

func TestCreateSession(t *testing.T) {
	mgr := setupTestDB(t)
	defer mgr.Close()

	session := &Session{
		ID:        uuid.NewString(),
		AgentName: "default",
		Provider:  "claude",
		Model:     "claude-3-sonnet",
		StartedAt: time.Now(),
		Tags:      []string{"test", "debug"},
	}

	err := mgr.CreateSession(session)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// 验证会话已创建
	retrieved, err := mgr.GetSession(session.ID)
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}

	if retrieved.ID != session.ID {
		t.Errorf("Session ID mismatch: got %s, want %s", retrieved.ID, session.ID)
	}

	if retrieved.AgentName != session.AgentName {
		t.Errorf("AgentName mismatch: got %s, want %s", retrieved.AgentName, session.AgentName)
	}
}

func TestSaveMessage(t *testing.T) {
	mgr := setupTestDB(t)
	defer mgr.Close()

	sessionID := uuid.NewString()
	session := &Session{
		ID:        sessionID,
		AgentName: "default",
		Provider:  "claude",
		Model:     "claude-3-sonnet",
		StartedAt: time.Now(),
	}

	mgr.CreateSession(session)

	msg := &Message{
		ID:        uuid.NewString(),
		SessionID: sessionID,
		Role:      "user",
		Content:   "Hello, AI!",
		Timestamp: time.Now(),
		Tokens:    10,
		Metadata:  map[string]interface{}{"test": true},
	}

	err := mgr.SaveMessage(msg)
	if err != nil {
		t.Fatalf("Failed to save message: %v", err)
	}

	// 验证消息已保存
	messages, err := mgr.GetMessages(sessionID)
	if err != nil {
		t.Fatalf("Failed to get messages: %v", err)
	}

	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}

	if messages[0].Content != msg.Content {
		t.Errorf("Content mismatch: got %s, want %s", messages[0].Content, msg.Content)
	}
}

func TestListSessions(t *testing.T) {
	mgr := setupTestDB(t)
	defer mgr.Close()

	// 创建多个会话
	for i := 0; i < 5; i++ {
		session := &Session{
			ID:        uuid.NewString(),
			AgentName: "default",
			Provider:  "claude",
			Model:     "claude-3-sonnet",
			StartedAt: time.Now().Add(-time.Duration(i) * time.Hour),
		}
		mgr.CreateSession(session)
	}

	// 列出会话
	sessions, err := mgr.ListSessions(Query{Limit: 10})
	if err != nil {
		t.Fatalf("Failed to list sessions: %v", err)
	}

	if len(sessions) != 5 {
		t.Errorf("Expected 5 sessions, got %d", len(sessions))
	}

	// 验证排序（最新的在前）
	for i := 0; i < len(sessions)-1; i++ {
		if sessions[i].StartedAt.Before(sessions[i+1].StartedAt) {
			t.Error("Sessions not sorted correctly")
		}
	}
}

func TestSearchMessages(t *testing.T) {
	mgr := setupTestDB(t)
	defer mgr.Close()

	sessionID := uuid.NewString()
	session := &Session{
		ID:        sessionID,
		AgentName: "default",
		Provider:  "claude",
		Model:     "claude-3-sonnet",
		StartedAt: time.Now(),
	}
	mgr.CreateSession(session)

	// 创建多条消息
	messages := []string{
		"如何实现单例模式？",
		"什么是依赖注入？",
		"单例模式的优缺点",
	}

	for i, content := range messages {
		msg := &Message{
			ID:        uuid.NewString(),
			SessionID: sessionID,
			Role:      "user",
			Content:   content,
			Timestamp: time.Now().Add(time.Duration(i) * time.Minute),
		}
		mgr.SaveMessage(msg)
	}

	// 搜索包含"单例"的消息
	results, err := mgr.SearchMessages(Query{Keyword: "单例", Limit: 10})
	if err != nil {
		t.Fatalf("Failed to search messages: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
}

func TestDeleteSession(t *testing.T) {
	mgr := setupTestDB(t)
	defer mgr.Close()

	sessionID := uuid.NewString()
	session := &Session{
		ID:        sessionID,
		AgentName: "default",
		Provider:  "claude",
		Model:     "claude-3-sonnet",
		StartedAt: time.Now(),
	}
	mgr.CreateSession(session)

	// 添加消息
	msg := &Message{
		ID:        uuid.NewString(),
		SessionID: sessionID,
		Role:      "user",
		Content:   "Test message",
		Timestamp: time.Now(),
	}
	mgr.SaveMessage(msg)

	// 删除会话
	err := mgr.DeleteSession(sessionID)
	if err != nil {
		t.Fatalf("Failed to delete session: %v", err)
	}

	// 验证会话已删除
	_, err = mgr.GetSession(sessionID)
	if err == nil {
		t.Error("Session should be deleted")
	}

	// 验证消息也已删除
	messages, _ := mgr.GetMessages(sessionID)
	if len(messages) != 0 {
		t.Errorf("Expected 0 messages, got %d", len(messages))
	}
}

func TestDeleteOldSessions(t *testing.T) {
	mgr := setupTestDB(t)
	defer mgr.Close()

	// 创建新旧会话
	oldTime := time.Now().Add(-100 * 24 * time.Hour)
	newTime := time.Now()

	oldSession := &Session{
		ID:        uuid.NewString(),
		AgentName: "default",
		Provider:  "claude",
		Model:     "claude-3-sonnet",
		StartedAt: oldTime,
	}
	mgr.CreateSession(oldSession)

	newSession := &Session{
		ID:        uuid.NewString(),
		AgentName: "default",
		Provider:  "claude",
		Model:     "claude-3-sonnet",
		StartedAt: newTime,
	}
	mgr.CreateSession(newSession)

	// 删除 90 天前的会话
	cutoff := time.Now().Add(-90 * 24 * time.Hour)
	count, err := mgr.DeleteOldSessions(cutoff)
	if err != nil {
		t.Fatalf("Failed to delete old sessions: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 deleted session, got %d", count)
	}

	// 验证只剩新会话
	sessions, _ := mgr.ListSessions(Query{Limit: 10})
	if len(sessions) != 1 {
		t.Errorf("Expected 1 remaining session, got %d", len(sessions))
	}

	if sessions[0].ID != newSession.ID {
		t.Error("Wrong session remained")
	}
}

func TestGetStats(t *testing.T) {
	mgr := setupTestDB(t)
	defer mgr.Close()

	// 创建会话和消息
	sessionID := uuid.NewString()
	session := &Session{
		ID:        sessionID,
		AgentName: "default",
		Provider:  "claude",
		Model:     "claude-3-sonnet",
		StartedAt: time.Now(),
	}
	mgr.CreateSession(session)

	for i := 0; i < 3; i++ {
		msg := &Message{
			ID:        uuid.NewString(),
			SessionID: sessionID,
			Role:      "user",
			Content:   "Test",
			Timestamp: time.Now(),
		}
		mgr.SaveMessage(msg)
	}

	// 获取统计
	stats, err := mgr.GetStats()
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if stats["total_sessions"].(int) != 1 {
		t.Errorf("Expected 1 session, got %v", stats["total_sessions"])
	}

	if stats["total_messages"].(int) != 3 {
		t.Errorf("Expected 3 messages, got %v", stats["total_messages"])
	}
}
