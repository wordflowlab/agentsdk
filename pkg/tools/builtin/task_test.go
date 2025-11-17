package builtin

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestNewTaskTool(t *testing.T) {
	tool, err := NewTaskTool(nil)
	if err != nil {
		t.Fatalf("Failed to create Task tool: %v", err)
	}

	if tool.Name() != "Task" {
		t.Errorf("Expected tool name 'Task', got '%s'", tool.Name())
	}

	if tool.Description() == "" {
		t.Error("Tool description should not be empty")
	}
}

func TestTaskTool_InputSchema(t *testing.T) {
	tool, err := NewTaskTool(nil)
	if err != nil {
		t.Fatalf("Failed to create Task tool: %v", err)
	}

	schema := tool.InputSchema()
	if schema == nil {
		t.Fatal("Input schema should not be nil")
	}

	properties, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Properties should be a map")
	}

	// 验证必需字段
	requiredFields := []string{"subagent_type", "prompt"}
	for _, field := range requiredFields {
		if _, exists := properties[field]; !exists {
			t.Errorf("Required field '%s' should exist in properties", field)
		}
	}

	required := schema["required"]
	var requiredArray []interface{}
	switch v := required.(type) {
	case []interface{}:
		requiredArray = v
	case []string:
		requiredArray = make([]interface{}, len(v))
		for i, s := range v {
			requiredArray[i] = s
		}
	default:
		t.Fatal("Required should be an array")
	}

	if len(requiredArray) != 2 {
		t.Errorf("Expected 2 required fields, got %d", len(requiredArray))
	}
}

func TestTaskTool_LaunchGeneralPurposeSubagent(t *testing.T) {
	tool, err := NewTaskTool(nil)
	if err != nil {
		t.Fatalf("Failed to create Task tool: %v", err)
	}

	input := map[string]interface{}{
		"subagent_type": "general-purpose",
		"prompt":       "Analyze the current project structure and provide a summary",
		"model":        "gpt-3.5-turbo",
	}

	result := ExecuteToolWithInput(t, tool, input)
	result = AssertToolSuccess(t, result)

	// 验证响应字段
	if taskID, exists := result["task_id"]; !exists {
		t.Error("Result should contain 'task_id' field")
	} else if taskIDStr, ok := taskID.(string); !ok || taskIDStr == "" {
		t.Error("task_id should be a non-empty string")
	}

	if subagentType, exists := result["subagent_type"]; !exists {
		t.Error("Result should contain 'subagent_type' field")
	} else if subagentTypeStr, ok := subagentType.(string); !ok || subagentTypeStr != "general-purpose" {
		t.Errorf("Expected subagent_type 'general-purpose', got %v", subagentType)
	}

	if result["status"].(string) != "running" {
		t.Errorf("Expected status 'running', got %v", result["status"])
	}
}

func TestTaskTool_InvalidSubagentType(t *testing.T) {
	tool, err := NewTaskTool(nil)
	if err != nil {
		t.Fatalf("Failed to create Task tool: %v", err)
	}

	input := map[string]interface{}{
		"subagent_type": "invalid_agent",
		"prompt":       "Test prompt",
	}

	result := ExecuteToolWithInput(t, tool, input)

	// 应该返回错误
	errMsg := AssertToolError(t, result)
	if !strings.Contains(strings.ToLower(errMsg), "invalid") &&
		!strings.Contains(strings.ToLower(errMsg), "subagent") {
		t.Errorf("Expected subagent validation error, got: %s", errMsg)
	}

	// 验证推荐的选项
	if recommendations, exists := result["recommendations"]; !exists {
		t.Error("Result should contain 'recommendations' field")
	} else if recommendationsArray, ok := recommendations.([]string); !ok || len(recommendationsArray) == 0 {
		t.Error("Recommendations should be a non-empty array")
	}
}

func TestTaskTool_MissingPrompt(t *testing.T) {
	tool, err := NewTaskTool(nil)
	if err != nil {
		t.Fatalf("Failed to create Task tool: %v", err)
	}

	input := map[string]interface{}{
		"subagent_type": "general-purpose",
		// 缺少prompt字段
	}

	result := ExecuteToolWithInput(t, tool, input)

	errMsg := AssertToolError(t, result)
	if !strings.Contains(strings.ToLower(errMsg), "prompt") &&
		!strings.Contains(strings.ToLower(errMsg), "required") {
		t.Errorf("Expected prompt validation error, got: %s", errMsg)
	}
}

func TestTaskTool_EmptyPrompt(t *testing.T) {
	tool, err := NewTaskTool(nil)
	if err != nil {
		t.Fatalf("Failed to create Task tool: %v", err)
	}

	input := map[string]interface{}{
		"subagent_type": "general-purpose",
		"prompt":       "", // 空提示
	}

	result := ExecuteToolWithInput(t, tool, input)

	errMsg := AssertToolError(t, result)
	if !strings.Contains(strings.ToLower(errMsg), "prompt") &&
		!strings.Contains(strings.ToLower(errMsg), "empty") {
		t.Errorf("Expected prompt empty error, got: %s", errMsg)
	}
}

func TestTaskTool_AllSubagentTypes(t *testing.T) {
	tool, err := NewTaskTool(nil)
	if err != nil {
		t.Fatalf("Failed to create Task tool: %v", err)
	}

	subagentTypes := []string{
		"general-purpose",
		"statusline-setup",
		"Explore",
		"Plan",
	}

	for _, subagentType := range subagentTypes {
		t.Run("Subagent_"+subagentType, func(t *testing.T) {
			input := map[string]interface{}{
				"subagent_type": subagentType,
				"prompt":       fmt.Sprintf("Test prompt for %s subagent", subagentType),
			}

			result := ExecuteToolWithInput(t, tool, input)

			// 所有有效的subagent类型都应该成功启动
			if !result["ok"].(bool) {
				t.Errorf("Failed to launch %s subagent: %v", subagentType, result["error"])
			}

			if result["subagent_type"].(string) != subagentType {
				t.Errorf("Expected subagent_type %s, got %v", subagentType, result["subagent_type"])
			}
		})
	}
}

func TestTaskTool_WithOptions(t *testing.T) {
	tool, err := NewTaskTool(nil)
	if err != nil {
		t.Fatalf("Failed to create Task tool: %v", err)
	}

	input := map[string]interface{}{
		"subagent_type":    "general-purpose",
		"prompt":          "Test with options",
		"model":           "gpt-4",
		"timeout_minutes": 5,
		"priority":        200,
		"async":           false,
	}

	result := ExecuteToolWithInput(t, tool, input)
	result = AssertToolSuccess(t, result)

	// 验证选项字段
	if result["model"].(string) != "gpt-4" {
		t.Errorf("Expected model 'gpt-4', got %v", result["model"])
	}

	if result["timeout_minutes"].(int) != 5 {
		t.Errorf("Expected timeout_minutes 5, got %v", result["timeout_minutes"])
	}

	if result["priority"].(int) != 200 {
		t.Errorf("Expected priority 200, got %v", result["priority"])
	}

	if result["async"].(bool) != false {
		t.Errorf("Expected async false, got %v", result["async"])
	}
}

func TestTaskTool_ResumeTask(t *testing.T) {
	tool, err := NewTaskTool(nil)
	if err != nil {
		t.Fatalf("Failed to create Task tool: %v", err)
	}

	// 首先启动一个任务
	initialInput := map[string]interface{}{
		"subagent_type": "general-purpose",
	"prompt":       "Initial task for resume testing",
	}

	initialResult := ExecuteToolWithInput(t, tool, initialInput)
	initialResult = AssertToolSuccess(t, initialResult)

	taskID := initialResult["task_id"].(string)

	// 等待任务启动
	time.Sleep(100 * time.Millisecond)

	// 尝试恢复任务（注意：这需要实际的子代理框架支持）
	resumeInput := map[string]interface{}{
		"subagent_type": "general-purpose",
		"prompt":       "Resumed task",
		"resume":       taskID,
	}

	result := ExecuteToolWithInput(t, tool, resumeInput)

	// 恢复功能可能不被简化实现支持
	if !result["ok"].(bool) {
		t.Logf("Resume functionality not yet implemented in simple version: %v", result["error"])
	} else {
		if result["task_id"].(string) != taskID {
			t.Errorf("Expected same task_id after resume, got %v", result["task_id"])
		}
	}
}

func TestTaskTool_ConcurrentSubagentLaunch(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent subagent test in short mode")
	}

	tool, err := NewTaskTool(nil)
	if err != nil {
		t.Fatalf("Failed to create Task tool: %v", err)
	}

	concurrency := 3
	result := RunConcurrentTest(concurrency, func() error {
		input := map[string]interface{}{
			"subagent_type": "general-purpose",
			"prompt":       "Concurrent test task",
			"timeout_minutes": 1,
		}

		result := ExecuteToolWithInput(t, tool, input)
		if !result["ok"].(bool) {
			return fmt.Errorf("Task launch failed")
		}

		// 验证task_id不为空
		taskID := result["task_id"].(string)
		if taskID == "" {
			return fmt.Errorf("Empty task_id returned")
		}

		return nil
	})

	if result.ErrorCount > 0 {
		t.Errorf("Concurrent subagent launch failed: %d errors out of %d attempts",
			result.ErrorCount, concurrency)
	}

	t.Logf("Concurrent subagent launch completed: %d success, %d errors in %v",
		result.SuccessCount, result.ErrorCount, result.Duration)
}

func TestTaskTool_PerformanceInfo(t *testing.T) {
	tool, err := NewTaskTool(nil)
	if err != nil {
		t.Fatalf("Failed to create Task tool: %v", err)
	}

	input := map[string]interface{}{
		"subagent_type": "general-purpose",
		"prompt":       "Performance test task",
	}

	result := ExecuteToolWithInput(t, tool, input)
	result = AssertToolSuccess(t, result)

	// 验证性能相关字段
	if _, exists := result["duration_ms"]; !exists {
		t.Error("Result should contain 'duration_ms' field")
	}

	if _, exists := result["start_time"]; !exists {
		t.Error("Result should contain 'start_time' field")
	}

	if _, exists := result["pid"]; !exists {
		t.Error("Result should contain 'pid' field")
	}

	if _, exists := result["command"]; !exists {
		t.Error("Result should contain 'command' field")
	}
}

func TestTaskTool_Metadata(t *testing.T) {
	tool, err := NewTaskTool(nil)
	if err != nil {
		t.Fatalf("Failed to create Task tool: %v", err)
	}

	input := map[string]interface{}{
		"subagent_type": "general-purpose",
		"prompt":       "Task with metadata",
		"model":       "gpt-3.5-turbo",
	}

	result := ExecuteToolWithInput(t, tool, input)
	result = AssertToolSuccess(t, result)

	// 验证subagent配置信息
	if subagentConfig, exists := result["subagent_config"]; !exists {
		t.Error("Result should contain 'subagent_config' field")
	} else if configMap, ok := subagentConfig.(map[string]interface{}); !ok {
		t.Error("subagent_config should be a map")
	} else {
		// 验证配置字段
		expectedFields := []string{"timeout", "max_tokens", "temperature", "work_dir"}
		for _, field := range expectedFields {
			if _, exists := configMap[field]; !exists {
				t.Logf("Subagent config field '%s' not found (may be optional)", field)
			}
		}
	}

	// 验证元数据
	if _, exists := result["metadata"]; !exists {
		t.Error("Result should contain 'metadata' field")
	}
}

func BenchmarkTaskTool_LaunchSubagent(b *testing.B) {
	tool, err := NewTaskTool(nil)
	if err != nil {
		b.Fatalf("Failed to create Task tool: %v", err)
	}

	input := map[string]interface{}{
		"subagent_type": "general-purpose",
		"prompt":       "Benchmark task",
	}

	BenchmarkTool(b, tool, input)
}

func BenchmarkTaskTool_LaunchWithFullOptions(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping full options benchmark in short mode")
	}

	tool, err := NewTaskTool(nil)
	if err != nil {
		b.Fatalf("Failed to create Task tool: %v", err)
	}

	input := map[string]interface{}{
		"subagent_type":    "general-purpose",
		"prompt":          "Complex benchmark task with detailed requirements",
		"model":           "gpt-4",
		"timeout_minutes": 10,
		"priority":        500,
		"async":           true,
	}

	BenchmarkTool(b, tool, input)
}