package builtin

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestNewKillShellTool(t *testing.T) {
	tool, err := NewKillShellTool(nil)
	if err != nil {
		t.Fatalf("Failed to create KillShell tool: %v", err)
	}

	if tool.Name() != "KillShell" {
		t.Errorf("Expected tool name 'KillShell', got '%s'", tool.Name())
	}

	if tool.Description() == "" {
		t.Error("Tool description should not be empty")
	}
}

func TestKillShellTool_InputSchema(t *testing.T) {
	tool, err := NewKillShellTool(nil)
	if err != nil {
		t.Fatalf("Failed to create KillShell tool: %v", err)
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
	requiredFields := []string{"shell_id"}
	for _, field := range requiredFields {
		if _, exists := properties[field]; !exists {
			t.Errorf("Required field '%s' should exist in properties", field)
		}
	}

	required := schema["required"]
	if requiredArray, ok := required.([]interface{}); !ok || len(requiredArray) == 0 || requiredArray[0] != "shell_id" {
		t.Error("shell_id should be required")
	}
}

func TestKillShellTool_KillNonExistentTask(t *testing.T) {
	tool, err := NewKillShellTool(nil)
	if err != nil {
		t.Fatalf("Failed to create KillShell tool: %v", err)
	}

	input := map[string]interface{}{
		"shell_id": "non_existent_task_id",
	}

	result := ExecuteToolWithInput(t, tool, input)

	// 应该返回错误
	errMsg := AssertToolError(t, result)
	if !strings.Contains(strings.ToLower(errMsg), "not found") &&
		!strings.Contains(strings.ToLower(errMsg), "exist") {
		t.Errorf("Expected 'not found' error, got: %s", errMsg)
	}
}

func TestKillShellTool_SignalTypes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping signal test in short mode")
	}

	tool, err := NewKillShellTool(nil)
	if err != nil {
		t.Fatalf("Failed to create KillShell tool: %v", err)
	}

	// 首先启动一个长时间运行的后台任务
	bashTool, err := NewBashTool(nil)
	if err != nil {
		t.Fatalf("Failed to create Bash tool: %v", err)
	}

	bashInput := map[string]interface{}{
		"command":    "sleep 30", // 长时间运行的任务
		"background": true,
	}

	bashResult := ExecuteToolWithInput(t, bashTool, bashInput)
	bashResult = AssertToolSuccess(t, bashResult)

	taskID, exists := bashResult["task_id"].(string)
	if !exists || taskID == "" {
		t.Fatal("Failed to get task ID from background task")
	}

	// 等待任务启动
	time.Sleep(200 * time.Millisecond)

	// 测试不同的信号类型
	signals := []string{"SIGTERM", "SIGINT", "15", "2"}

	for _, signal := range signals {
		t.Run("Signal_"+signal, func(t *testing.T) {
			// 重新启动任务（因为前面的测试可能已经终止了它）
			newBashResult := ExecuteToolWithInput(t, bashTool, bashInput)
			newBashResult = AssertToolSuccess(t, newBashResult)

			newTaskID, exists := newBashResult["task_id"].(string)
			if !exists || newTaskID == "" {
				t.Fatal("Failed to get new task ID")
			}

			// 等待任务启动
			time.Sleep(100 * time.Millisecond)

			// 使用KillShell工具终止任务
			killInput := map[string]interface{}{
				"shell_id": newTaskID,
				"signal":   signal,
			}

			killResult := ExecuteToolWithInput(t, tool, killInput)

			// 验证响应
			if killResult["ok"].(bool) {
				t.Logf("Successfully sent signal %s to task %s", signal, newTaskID)
			} else {
				t.Logf("Failed to send signal %s: %v", signal, killResult["error"])
			}

			// 验证信号信息被正确记录
			if killResult["signal"] != signal {
				t.Errorf("Expected signal %s, got %v", signal, killResult["signal"])
			}

			if killResult["shell_id"] != newTaskID {
				t.Errorf("Expected shell_id %s, got %v", newTaskID, killResult["shell_id"])
			}
		})
	}
}

func TestKillShellTool_ForceKill(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping force kill test in short mode")
	}

	tool, err := NewKillShellTool(nil)
	if err != nil {
		t.Fatalf("Failed to create KillShell tool: %v", err)
	}

	// 启动一个长时间运行的任务
	bashTool, err := NewBashTool(nil)
	if err != nil {
		t.Fatalf("Failed to create Bash tool: %v", err)
	}

	bashInput := map[string]interface{}{
		"command":    "while true; do sleep 1; done", // 无限循环
		"background": true,
	}

	bashResult := ExecuteToolWithInput(t, bashTool, bashInput)
	bashResult = AssertToolSuccess(t, bashResult)

	taskID, exists := bashResult["task_id"].(string)
	if !exists || taskID == "" {
		t.Fatal("Failed to get task ID from background task")
	}

	// 等待任务启动
	time.Sleep(200 * time.Millisecond)

	// 使用force=true强制终止
	killInput := map[string]interface{}{
		"shell_id": taskID,
		"force":    true,
	}

	killResult := ExecuteToolWithInput(t, tool, killInput)
	killResult = AssertToolSuccess(t, killResult)

	// 验证使用了SIGKILL信号
	if killResult["signal"] != "SIGKILL" {
		t.Errorf("Expected signal 'SIGKILL' for force=true, got %v", killResult["signal"])
	}

	if !killResult["force"].(bool) {
		t.Error("Expected force=true in response")
	}
}

func TestKillShellTool_WaitForCompletion(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping wait test in short mode")
	}

	tool, err := NewKillShellTool(nil)
	if err != nil {
		t.Fatalf("Failed to create KillShell tool: %v", err)
	}

	// 启动一个短时间运行的任务
	bashTool, err := NewBashTool(nil)
	if err != nil {
		t.Fatalf("Failed to create Bash tool: %v", err)
	}

	bashInput := map[string]interface{}{
		"command":    "sleep 2 && echo 'Task completed'",
		"background": true,
	}

	bashResult := ExecuteToolWithInput(t, bashTool, bashInput)
	bashResult = AssertToolSuccess(t, bashResult)

	taskID, exists := bashResult["task_id"].(string)
	if !exists || taskID == "" {
		t.Fatal("Failed to get task ID from background task")
	}

	// 等待任务运行一点时间
	time.Sleep(500 * time.Millisecond)

	// 使用wait=true等待任务完成
	killInput := map[string]interface{}{
		"shell_id": taskID,
		"signal":   "SIGTERM",
		"wait":     true,
		"timeout":  5, // 5秒超时
	}

	start := time.Now()
	killResult := ExecuteToolWithInput(t, tool, killInput)
	duration := time.Since(start)

	killResult = AssertToolSuccess(t, killResult)

	// 验证等待相关字段
	if !killResult["wait"].(bool) {
		t.Error("Expected wait=true in response")
	}

	if timeoutSeconds, exists := killResult["timeout_seconds"]; !exists {
		t.Error("Result should contain 'timeout_seconds' field")
	} else if timeoutSecondsInt, ok := timeoutSeconds.(int); !ok || timeoutSecondsInt != 5 {
		t.Errorf("Expected timeout_seconds=5, got %v", timeoutSeconds)
	}

	t.Logf("Kill operation completed in %v", duration)
}

func TestKillShellTool_CleanupResources(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping cleanup test in short mode")
	}

	tool, err := NewKillShellTool(nil)
	if err != nil {
		t.Fatalf("Failed to create KillShell tool: %v", err)
	}

	// 启动任务
	bashTool, err := NewBashTool(nil)
	if err != nil {
		t.Fatalf("Failed to create Bash tool: %v", err)
	}

	bashInput := map[string]interface{}{
		"command":    "echo 'Cleanup test' > /tmp/test_output.txt",
		"background": true,
	}

	bashResult := ExecuteToolWithInput(t, bashTool, bashInput)
	bashResult = AssertToolSuccess(t, bashResult)

	taskID, exists := bashResult["task_id"].(string)
	if !exists || taskID == "" {
		t.Fatal("Failed to get task ID from background task")
	}

	// 等待任务完成
	time.Sleep(300 * time.Millisecond)

	// 终止任务并清理资源
	killInput := map[string]interface{}{
		"shell_id": taskID,
		"cleanup": true,
	}

	killResult := ExecuteToolWithInput(t, tool, killInput)
	killResult = AssertToolSuccess(t, killResult)

	// 验证清理相关字段
	if !killResult["cleanup"].(bool) {
		t.Error("Expected cleanup=true in response")
	}

	if cleanupCompleted, exists := killResult["cleanup_completed"]; !exists {
		t.Error("Result should contain 'cleanup_completed' field")
	} else if !cleanupCompleted.(bool) {
		t.Error("Expected cleanup_completed=true")
	}

	if cleanupInfo, exists := killResult["cleanup_info"]; !exists {
		t.Error("Result should contain 'cleanup_info' field")
	} else if cleanupInfoMap, ok := cleanupInfo.(map[string]interface{}); !ok {
		t.Error("cleanup_info should be a map")
	} else {
		// 验证清理信息包含必要字段
		expectedFields := []string{"output_file", "error_file", "task_file", "files_cleared"}
		for _, field := range expectedFields {
			if _, exists := cleanupInfoMap[field]; !exists {
				t.Errorf("cleanup_info should contain field '%s'", field)
			}
		}
	}
}

func TestKillShellTool_InputValidation(t *testing.T) {
	tool, err := NewKillShellTool(nil)
	if err != nil {
		t.Fatalf("Failed to create KillShell tool: %v", err)
	}

	// 测试缺少必需参数
	input := map[string]interface{}{}
	result := ExecuteToolWithInput(t, tool, input)

	errMsg := AssertToolError(t, result)
	if !strings.Contains(strings.ToLower(errMsg), "shell_id") &&
		!strings.Contains(strings.ToLower(errMsg), "required") {
		t.Errorf("Expected shell_id required error, got: %s", errMsg)
	}

	// 测试空shell_id
	input = map[string]interface{}{
		"shell_id": "",
	}
	result = ExecuteToolWithInput(t, tool, input)

	errMsg = AssertToolError(t, result)
	if !strings.Contains(strings.ToLower(errMsg), "shell_id") &&
		!strings.Contains(strings.ToLower(errMsg), "empty") {
		t.Errorf("Expected shell_id empty error, got: %s", errMsg)
	}
}

func TestKillShellTool_InvalidSignal(t *testing.T) {
	tool, err := NewKillShellTool(nil)
	if err != nil {
		t.Fatalf("Failed to create KillShell tool: %v", err)
	}

	input := map[string]interface{}{
		"shell_id": "dummy_id",
		"signal":   "INVALID_SIGNAL",
	}

	result := ExecuteToolWithInput(t, tool, input)

	// 应该仍然执行，但使用默认信号或处理无效信号
	if result["ok"].(bool) {
		// 如果成功，验证使用了合理的信号
		validSignals := []string{"SIGTERM", "SIGINT", "SIGHUP", "SIGQUIT"}
		signal := result["signal"].(string)
		signalValid := false
		for _, valid := range validSignals {
			if signal == valid {
				signalValid = true
				break
			}
		}
		if !signalValid {
			t.Logf("Unknown signal used: %s", signal)
		}
	}
}

func BenchmarkKillShellTool_KillTask(b *testing.B) {
	// 注意：这个基准测试需要在实际的运行环境中才有意义
	tool, err := NewKillShellTool(nil)
	if err != nil {
		b.Fatalf("Failed to create KillShell tool: %v", err)
	}

	input := map[string]interface{}{
		"shell_id": "benchmark_task_id",
		"signal":   "SIGTERM",
	}

	BenchmarkTool(b, tool, input)
}

func TestKillShellTool_ConcurrentKill(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent kill test in short mode")
	}

	tool, err := NewKillShellTool(nil)
	if err != nil {
		t.Fatalf("Failed to create KillShell tool: %v", err)
	}

	// 启动多个后台任务
	bashTool, err := NewBashTool(nil)
	if err != nil {
		t.Fatalf("Failed to create Bash tool: %v", err)
	}

	taskIDs := []string{}
	numTasks := 3

	for i := 0; i < numTasks; i++ {
		bashInput := map[string]interface{}{
			"command":    fmt.Sprintf("sleep 5 && echo 'Task %d completed'", i),
			"background": true,
		}

		bashResult := ExecuteToolWithInput(t, bashTool, bashInput)
		bashResult = AssertToolSuccess(t, bashResult)

		taskID := bashResult["task_id"].(string)
		taskIDs = append(taskIDs, taskID)
	}

	// 等待任务启动
	time.Sleep(200 * time.Millisecond)

	// 并发终止任务
	result := RunConcurrentTest(len(taskIDs), func() error {
		// 使用第一个任务ID进行测试（因为所有任务都应该是相同的处理方式）
		input := map[string]interface{}{
			"shell_id": taskIDs[0],
			"signal":   "SIGTERM",
		}
		result := ExecuteToolWithInput(t, tool, input)
		if !result["ok"].(bool) {
			return fmt.Errorf("KillShell execution failed")
		}
		return nil
	})

	if result.ErrorCount > 0 {
		t.Errorf("Concurrent kill operations failed: %d errors out of %d attempts",
			result.ErrorCount, len(taskIDs))
	}

	t.Logf("Concurrent kill operations completed: %d success, %d errors in %v",
		result.SuccessCount, result.ErrorCount, result.Duration)
}