package memory

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/wordflowlab/agentsdk/pkg/backends"
)

func TestWorkingMemoryManager_Basic(t *testing.T) {
	ctx := context.Background()
	backend := backends.NewStateBackend()

	// 创建 Working Memory 管理器（thread scope）
	manager, err := NewWorkingMemoryManager(&WorkingMemoryConfig{
		Backend:  backend,
		BasePath: "/working_memory/",
		Scope:    ScopeThread,
	})
	if err != nil {
		t.Fatalf("create manager: %v", err)
	}

	threadID := "test-thread-1"
	resourceID := "test-resource-1"

	// 初始状态应该为空
	content, err := manager.Get(ctx, threadID, resourceID)
	if err != nil {
		t.Fatalf("get initial: %v", err)
	}
	if content != "" {
		t.Errorf("expected empty content, got: %s", content)
	}

	// 更新 Working Memory
	testContent := "# User Profile\n\nName: Alice\nPreferences: concise code reviews"
	err = manager.Update(ctx, threadID, resourceID, testContent)
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	// 读取更新后的内容
	content, err = manager.Get(ctx, threadID, resourceID)
	if err != nil {
		t.Fatalf("get after update: %v", err)
	}
	if content != testContent {
		t.Errorf("expected %q, got %q", testContent, content)
	}

	// 再次更新（覆盖）
	newContent := "# Updated Profile\n\nName: Alice Smith\nRole: Engineer"
	err = manager.Update(ctx, threadID, resourceID, newContent)
	if err != nil {
		t.Fatalf("update again: %v", err)
	}

	content, err = manager.Get(ctx, threadID, resourceID)
	if err != nil {
		t.Fatalf("get after second update: %v", err)
	}
	if content != newContent {
		t.Errorf("expected %q, got %q", newContent, content)
	}
}

func TestWorkingMemoryManager_Scope(t *testing.T) {
	ctx := context.Background()
	backend := backends.NewStateBackend()

	// 测试 thread scope
	threadManager, err := NewWorkingMemoryManager(&WorkingMemoryConfig{
		Backend:  backend,
		BasePath: "/working_memory/",
		Scope:    ScopeThread,
	})
	if err != nil {
		t.Fatalf("create thread manager: %v", err)
	}

	// 测试 resource scope
	resourceManager, err := NewWorkingMemoryManager(&WorkingMemoryConfig{
		Backend:  backend,
		BasePath: "/working_memory/",
		Scope:    ScopeResource,
	})
	if err != nil {
		t.Fatalf("create resource manager: %v", err)
	}

	threadID1 := "thread-1"
	threadID2 := "thread-2"
	resourceID := "resource-1"

	// Thread scope: 不同 thread 有独立的 working memory
	err = threadManager.Update(ctx, threadID1, resourceID, "Thread 1 content")
	if err != nil {
		t.Fatalf("update thread 1: %v", err)
	}

	err = threadManager.Update(ctx, threadID2, resourceID, "Thread 2 content")
	if err != nil {
		t.Fatalf("update thread 2: %v", err)
	}

	content1, _ := threadManager.Get(ctx, threadID1, resourceID)
	content2, _ := threadManager.Get(ctx, threadID2, resourceID)

	if content1 == content2 {
		t.Error("thread scope should have separate content for different threads")
	}

	// Resource scope: 相同 resource 的不同 thread 共享 working memory
	err = resourceManager.Update(ctx, threadID1, resourceID, "Shared resource content")
	if err != nil {
		t.Fatalf("update resource from thread 1: %v", err)
	}

	contentFromThread1, _ := resourceManager.Get(ctx, threadID1, resourceID)
	contentFromThread2, _ := resourceManager.Get(ctx, threadID2, resourceID)

	if contentFromThread1 != contentFromThread2 {
		t.Errorf("resource scope should share content: %q vs %q", contentFromThread1, contentFromThread2)
	}
}

func TestWorkingMemoryManager_FindAndReplace(t *testing.T) {
	ctx := context.Background()
	backend := backends.NewStateBackend()

	manager, err := NewWorkingMemoryManager(&WorkingMemoryConfig{
		Backend:  backend,
		BasePath: "/working_memory/",
		Scope:    ScopeThread,
	})
	if err != nil {
		t.Fatalf("create manager: %v", err)
	}

	threadID := "test-thread"
	resourceID := "test-resource"

	// 初始内容
	initialContent := "Line 1\nLine 2\nLine 3"
	err = manager.Update(ctx, threadID, resourceID, initialContent)
	if err != nil {
		t.Fatalf("update initial: %v", err)
	}

	// Find and replace
	err = manager.FindAndReplace(ctx, threadID, resourceID, "Line 2", "Line 2 (updated)")
	if err != nil {
		t.Fatalf("find and replace: %v", err)
	}

	content, _ := manager.Get(ctx, threadID, resourceID)
	if !strings.Contains(content, "Line 2 (updated)") {
		t.Errorf("expected to find 'Line 2 (updated)', got: %s", content)
	}

	// Append (empty search string)
	err = manager.FindAndReplace(ctx, threadID, resourceID, "", "Line 4")
	if err != nil {
		t.Fatalf("append: %v", err)
	}

	content, _ = manager.Get(ctx, threadID, resourceID)
	if !strings.Contains(content, "Line 4") {
		t.Errorf("expected to find 'Line 4', got: %s", content)
	}
}

func TestWorkingMemoryManager_TTL(t *testing.T) {
	ctx := context.Background()
	backend := backends.NewStateBackend()

	// 创建带 TTL 的管理器（1 秒过期）
	manager, err := NewWorkingMemoryManager(&WorkingMemoryConfig{
		Backend:    backend,
		BasePath:   "/working_memory/",
		Scope:      ScopeThread,
		DefaultTTL: 1 * time.Second,
	})
	if err != nil {
		t.Fatalf("create manager: %v", err)
	}

	threadID := "test-thread"
	resourceID := "test-resource"

	// 更新内容
	err = manager.Update(ctx, threadID, resourceID, "Temporary content")
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	// 立即读取应该成功
	content, _ := manager.Get(ctx, threadID, resourceID)
	if content == "" {
		t.Error("expected content immediately after update")
	}

	// 等待过期
	time.Sleep(2 * time.Second)

	// 过期后应该返回空
	content, err = manager.Get(ctx, threadID, resourceID)
	if err != nil {
		t.Fatalf("get after expiry: %v", err)
	}
	if content != "" {
		t.Errorf("expected empty content after expiry, got: %s", content)
	}
}

func TestWorkingMemoryManager_Schema(t *testing.T) {
	ctx := context.Background()
	backend := backends.NewStateBackend()

	// 创建带 Schema 的管理器
	schema := &JSONSchema{
		Type: "object",
		Properties: map[string]*JSONSchema{
			"name": {Type: "string"},
			"age":  {Type: "integer"},
		},
		Required: []string{"name"},
	}

	manager, err := NewWorkingMemoryManager(&WorkingMemoryConfig{
		Backend:  backend,
		BasePath: "/working_memory/",
		Scope:    ScopeThread,
		Schema:   schema,
	})
	if err != nil {
		t.Fatalf("create manager: %v", err)
	}

	threadID := "test-thread"
	resourceID := "test-resource"

	// 有效的 JSON
	validJSON := `{"name": "Alice", "age": 30}`
	err = manager.Update(ctx, threadID, resourceID, validJSON)
	if err != nil {
		t.Errorf("valid JSON should be accepted: %v", err)
	}

	// 缺少必需字段
	invalidJSON := `{"age": 30}`
	err = manager.Update(ctx, threadID, resourceID, invalidJSON)
	if err == nil {
		t.Error("invalid JSON (missing required field) should be rejected")
	}

	// 无效的 JSON 格式
	notJSON := "This is not JSON"
	err = manager.Update(ctx, threadID, resourceID, notJSON)
	if err == nil {
		t.Error("non-JSON content should be rejected when schema is configured")
	}
}

func TestWorkingMemoryManager_Delete(t *testing.T) {
	ctx := context.Background()
	backend := backends.NewStateBackend()

	manager, err := NewWorkingMemoryManager(&WorkingMemoryConfig{
		Backend:  backend,
		BasePath: "/working_memory/",
		Scope:    ScopeThread,
	})
	if err != nil {
		t.Fatalf("create manager: %v", err)
	}

	threadID := "test-thread"
	resourceID := "test-resource"

	// 创建内容
	err = manager.Update(ctx, threadID, resourceID, "Some content")
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	// 验证内容存在
	content, _ := manager.Get(ctx, threadID, resourceID)
	if content == "" {
		t.Error("content should exist before delete")
	}

	// 删除
	err = manager.Delete(ctx, threadID, resourceID)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}

	// 验证内容已删除
	content, _ = manager.Get(ctx, threadID, resourceID)
	if content != "" {
		t.Errorf("content should be empty after delete, got: %s", content)
	}
}
