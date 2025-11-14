package main

import (
	"context"
	"testing"

	"github.com/wordflowlab/agentsdk/pkg/backends"
	"github.com/wordflowlab/agentsdk/pkg/memory"
)

// TestWorkingMemoryBasic 验证 Working Memory 基本功能
func TestWorkingMemoryBasic(t *testing.T) {
	ctx := context.Background()
	backend := backends.NewStateBackend()

	// 创建 Working Memory 管理器
	manager, err := memory.NewWorkingMemoryManager(&memory.WorkingMemoryConfig{
		Backend:  backend,
		BasePath: "/working_memory/",
		Scope:    memory.ScopeThread,
	})
	if err != nil {
		t.Fatalf("create manager: %v", err)
	}

	threadID := "test-thread"
	resourceID := "test-resource"

	// 更新 Working Memory
	testContent := "# Test Profile\nName: Test User"
	err = manager.Update(ctx, threadID, resourceID, testContent)
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	// 读取
	content, err := manager.Get(ctx, threadID, resourceID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	if content != testContent {
		t.Errorf("expected %q, got %q", testContent, content)
	}

	t.Log("✓ Working Memory 基本功能验证通过")
}

// TestWorkingMemoryScope 验证作用域隔离
func TestWorkingMemoryScope(t *testing.T) {
	ctx := context.Background()
	backend := backends.NewStateBackend()

	threadManager, _ := memory.NewWorkingMemoryManager(&memory.WorkingMemoryConfig{
		Backend:  backend,
		BasePath: "/working_memory/",
		Scope:    memory.ScopeThread,
	})

	resourceManager, _ := memory.NewWorkingMemoryManager(&memory.WorkingMemoryConfig{
		Backend:  backend,
		BasePath: "/working_memory/",
		Scope:    memory.ScopeResource,
	})

	thread1 := "thread-1"
	thread2 := "thread-2"
	resource := "resource-1"

	// Thread scope - 不同 thread 应该隔离
	threadManager.Update(ctx, thread1, resource, "Thread 1 content")
	threadManager.Update(ctx, thread2, resource, "Thread 2 content")

	content1, _ := threadManager.Get(ctx, thread1, resource)
	content2, _ := threadManager.Get(ctx, thread2, resource)

	if content1 == content2 {
		t.Error("thread scope should isolate different threads")
	}

	// Resource scope - 相同 resource 应该共享
	resourceManager.Update(ctx, thread1, resource, "Shared content")

	contentFromThread1, _ := resourceManager.Get(ctx, thread1, resource)
	contentFromThread2, _ := resourceManager.Get(ctx, thread2, resource)

	if contentFromThread1 != contentFromThread2 {
		t.Error("resource scope should share content across threads")
	}

	t.Log("✓ Working Memory 作用域隔离验证通过")
}

// TestWorkingMemorySchema 验证 Schema 验证
func TestWorkingMemorySchema(t *testing.T) {
	ctx := context.Background()
	backend := backends.NewStateBackend()

	schema := &memory.JSONSchema{
		Type: "object",
		Properties: map[string]*memory.JSONSchema{
			"name": {Type: "string"},
			"age":  {Type: "integer"},
		},
		Required: []string{"name"},
	}

	manager, _ := memory.NewWorkingMemoryManager(&memory.WorkingMemoryConfig{
		Backend:  backend,
		BasePath: "/working_memory/",
		Scope:    memory.ScopeThread,
		Schema:   schema,
	})

	threadID := "test-thread"
	resourceID := "test-resource"

	// 有效的 JSON
	validJSON := `{"name": "Alice", "age": 30}`
	err := manager.Update(ctx, threadID, resourceID, validJSON)
	if err != nil {
		t.Errorf("valid JSON should be accepted: %v", err)
	}

	// 无效的 JSON（缺少必需字段）
	invalidJSON := `{"age": 30}`
	err = manager.Update(ctx, threadID, resourceID, invalidJSON)
	if err == nil {
		t.Error("invalid JSON should be rejected")
	}

	t.Log("✓ Working Memory Schema 验证通过")
}
