package builtin

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestNewBashOutputTool(t *testing.T) {
	tool, err := NewBashOutputTool(nil)
	if err != nil {
		t.Fatalf("Failed to create BashOutput tool: %v", err)
	}

	if tool.Name() != "BashOutput" {
		t.Errorf("Expected tool name 'BashOutput', got '%s'", tool.Name())
	}

	if tool.Description() == "" {
		t.Error("Tool description should not be empty")
	}
}

func TestBashOutputTool_GetOutput(t *testing.T) {
	// 首先启动一个后台任务
	bashTool, err := NewBashTool(nil)
	if err != nil {
		t.Fatalf("Failed to create Bash tool: %v", err)
	}

	// 启动一个长时间运行的后台任务
	bashInput := map[string]interface{}{
		"command":    fmt.Sprintf("for i in {1..5}; do echo 'Line $i'; sleep 0.1; done"),
		"background": true,
	}

	bashResult := ExecuteToolWithInput(t, bashTool, bashInput)
	bashResult = AssertToolSuccess(t, bashResult)
	taskID, exists := bashResult["task_id"].(string)
	if !exists || taskID == "" {
		t.Fatal("Failed to get task ID from background task")
	}

	// 等待任务产生一些输出
	time.Sleep(200 * time.Millisecond)

	// 现在使用BashOutput工具获取输出
	bashOutputTool, err := NewBashOutputTool(nil)
	if err != nil {
		t.Fatalf("Failed to create BashOutput tool: %v", err)
	}

	outputInput := map[string]interface{}{
		"bash_id": taskID,
	}

	result := ExecuteToolWithInput(t, bashOutputTool, outputInput)
	result = AssertToolSuccess(t, result)

	// 验证响应字段
	if bashID, exists := result["bash_id"]; !exists {
		t.Error("Result should contain 'bash_id' field")
	} else if bashIDStr, ok := bashID.(string); !ok || bashIDStr != taskID {
		t.Error("bash_id should match the input task ID")
	}

	// 验证输出内容
	if stdout, exists := result["stdout"]; !exists {
		t.Error("Result should contain 'stdout' field")
	} else if stdoutStr, ok := stdout.(string); !ok {
		t.Error("stdout should be a string")
	} else {
		if stdoutStr == "" {
			t.Error("Expected some output from the background task")
		}
	}

	// 验证状态信息
	if _, exists := result["status"]; !exists {
		t.Error("Result should contain 'status' field")
	}
}

func TestBashOutputTool_WithFilter(t *testing.T) {
	// 启动后台任务
	bashTool, err := NewBashTool(nil)
	if err != nil {
		t.Fatalf("Failed to create Bash tool: %v", err)
	}

	bashInput := map[string]interface{}{
		"command":    "echo 'ERROR: Something went wrong'; echo 'INFO: Processing complete'; echo 'WARNING: Low disk space'",
		"background": true,
	}

	bashResult := ExecuteToolWithInput(t, bashTool, bashInput)
	bashResult = AssertToolSuccess(t, bashResult)
	taskID, exists := bashResult["task_id"].(string)
	if !exists || taskID == "" {
		t.Fatal("Failed to get task ID from background task")
	}

	// 等待任务完成
	time.Sleep(500 * time.Millisecond)

	// 使用BashOutput工具获取过滤后的输出
	bashOutputTool, err := NewBashOutputTool(nil)
	if err != nil {
		t.Fatalf("Failed to create BashOutput tool: %v", err)
	}

	outputInput := map[string]interface{}{
		"bash_id": taskID,
		"filter":  "ERROR|WARNING", // 只显示包含ERROR或WARNING的行
	}

	result := ExecuteToolWithInput(t, bashOutputTool, outputInput)
	result = AssertToolSuccess(t, result)

	if stdout, exists := result["stdout"]; !exists {
		t.Error("Result should contain 'stdout' field")
	} else if stdoutStr, ok := stdout.(string); !ok {
		t.Error("stdout should be a string")
	} else {
		// 应该只包含ERROR和WARNING行，不包含INFO行
		if !strings.Contains(stdoutStr, "ERROR") {
			t.Error("Expected to find ERROR in filtered output")
		}
		if !strings.Contains(stdoutStr, "WARNING") {
			t.Error("Expected to find WARNING in filtered output")
		}
		if strings.Contains(stdoutStr, "INFO") {
			t.Error("Should not contain INFO in filtered output")
		}
	}
}

func TestBashOutputTool_NonExistentTask(t *testing.T) {
	bashOutputTool, err := NewBashOutputTool(nil)
	if err != nil {
		t.Fatalf("Failed to create BashOutput tool: %v", err)
	}

	outputInput := map[string]interface{}{
		"bash_id": "non_existent_task_id",
	}

	result := ExecuteToolWithInput(t, bashOutputTool, outputInput)

	// 应该返回错误
	errMsg := AssertToolError(t, result)
	if !strings.Contains(strings.ToLower(errMsg), "not found") &&
		!strings.Contains(strings.ToLower(errMsg), "exist") {
		t.Errorf("Expected 'not found' error, got: %s", errMsg)
	}
}

func TestBashOutputTool_LinesLimit(t *testing.T) {
	// 启动产生多行输出的后台任务
	bashTool, err := NewBashTool(nil)
	if err != nil {
		t.Fatalf("Failed to create Bash tool: %v", err)
	}

	bashInput := map[string]interface{}{
		"command":    "for i in {1..20}; do echo 'Line $i'; done",
		"background": true,
	}

	bashResult := ExecuteToolWithInput(t, bashTool, bashInput)
	bashResult = AssertToolSuccess(t, bashResult)
	taskID, exists := bashResult["task_id"].(string)
	if !exists || taskID == "" {
		t.Fatal("Failed to get task ID from background task")
	}

	// 等待任务完成
	time.Sleep(500 * time.Millisecond)

	// 使用BashOutput工具获取限制行数的输出
	bashOutputTool, err := NewBashOutputTool(nil)
	if err != nil {
		t.Fatalf("Failed to create BashOutput tool: %v", err)
	}

	outputInput := map[string]interface{}{
		"bash_id": taskID,
		"lines":   5, // 只返回前5行
	}

	result := ExecuteToolWithInput(t, bashOutputTool, outputInput)
	result = AssertToolSuccess(t, result)

	if stdout, exists := result["stdout"]; !exists {
		t.Error("Result should contain 'stdout' field")
	} else if stdoutStr, ok := stdout.(string); !ok {
		t.Error("stdout should be a string")
	} else {
		lines := strings.Split(strings.TrimSpace(stdoutStr), "\n")
		if len(lines) > 5 {
			t.Errorf("Expected at most 5 lines, got %d", len(lines))
		}
	}
}

func TestBashOutputTool_ResourceInfo(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping resource info test in short mode")
	}

	// 启动CPU密集型后台任务
	bashTool, err := NewBashTool(nil)
	if err != nil {
		t.Fatalf("Failed to create Bash tool: %v", err)
	}

	bashInput := map[string]interface{}{
		"command":    "i=0; while [ $i -lt 1000 ]; do i=$((i+1)); done",
		"background": true,
	}

	bashResult := ExecuteToolWithInput(t, bashTool, bashInput)
	bashResult = AssertToolSuccess(t, bashResult)
	taskID, exists := bashResult["task_id"].(string)
	if !exists || taskID == "" {
		t.Fatal("Failed to get task ID from background task")
	}

	// 等待任务运行一段时间
	time.Sleep(200 * time.Millisecond)

	// 获取包含资源信息的输出
	bashOutputTool, err := NewBashOutputTool(nil)
	if err != nil {
		t.Fatalf("Failed to create BashOutput tool: %v", err)
	}

	outputInput := map[string]interface{}{
		"bash_id":       taskID,
		"resource_info": true,
	}

	result := ExecuteToolWithInput(t, bashOutputTool, outputInput)
	result = AssertToolSuccess(t, result)

	// 验证资源信息字段
	if _, exists := result["resource_usage"]; !exists {
		t.Error("Result should contain 'resource_usage' field when resource_info=true")
	}

	if _, exists := result["pid"]; !exists {
		t.Error("Result should contain 'pid' field")
	}
}

func TestBashOutputTool_FollowMode(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping follow mode test in short mode")
	}

	// 启动持续产生输出的任务
	bashTool, err := NewBashTool(nil)
	if err != nil {
		t.Fatalf("Failed to create Bash tool: %v", err)
	}

	bashInput := map[string]interface{}{
		"command":    "for i in {1..10}; do echo 'Line $i'; sleep 0.2; done",
		"background": true,
	}

	bashResult := ExecuteToolWithInput(t, bashTool, bashInput)
	bashResult = AssertToolSuccess(t, bashResult)
	taskID, exists := bashResult["task_id"].(string)
	if !exists || taskID == "" {
		t.Fatal("Failed to get task ID from background task")
	}

	// 使用跟随模式获取输出
	bashOutputTool, err := NewBashOutputTool(nil)
	if err != nil {
		t.Fatalf("Failed to create BashOutput tool: %v", err)
	}

	outputInput := map[string]interface{}{
		"bash_id": taskID,
		"follow":  true,
	}

	result := ExecuteToolWithInput(t, bashOutputTool, outputInput)
	result = AssertToolSuccess(t, result)

	// 验证跟随模式标识
	if followMode, exists := result["follow_mode"]; !exists {
		t.Error("Result should contain 'follow_mode' field")
	} else if followBool, ok := followMode.(bool); !ok || !followBool {
		t.Error("follow_mode should be true")
	}
}

func TestBashOutputTool_IncludeStderr(t *testing.T) {
	// 启动产生stderr输出的任务
	bashTool, err := NewBashTool(nil)
	if err != nil {
		t.Fatalf("Failed to create Bash tool: %v", err)
	}

	bashInput := map[string]interface{}{
		"command":    "echo 'stdout message'; echo 'stderr message' >&2",
		"background": true,
	}

	bashResult := ExecuteToolWithInput(t, bashTool, bashInput)
	bashResult = AssertToolSuccess(t, bashResult)
	taskID, exists := bashResult["task_id"].(string)
	if !exists || taskID == "" {
		t.Fatal("Failed to get task ID from background task")
	}

	// 等待任务完成
	time.Sleep(500 * time.Millisecond)

	// 获取包含stderr的输出
	bashOutputTool, err := NewBashOutputTool(nil)
	if err != nil {
		t.Fatalf("Failed to create BashOutput tool: %v", err)
	}

	outputInput := map[string]interface{}{
		"bash_id":         taskID,
		"include_stderr": true,
	}

	result := ExecuteToolWithInput(t, bashOutputTool, outputInput)
	result = AssertToolSuccess(t, result)

	// 验证stderr字段
	if stderr, exists := result["stderr"]; !exists {
		t.Error("Result should contain 'stderr' field when include_stderr=true")
	} else if stderrStr, ok := stderr.(string); !ok {
		t.Error("stderr should be a string")
	} else {
		if !strings.Contains(stderrStr, "stderr message") {
			t.Error("Expected to find stderr message in stderr output")
		}
	}

	// 验证总行数
	if totalLines, exists := result["total_lines"]; !exists {
		t.Error("Result should contain 'total_lines' field")
	} else if totalLinesInt, ok := totalLines.(int); !ok || totalLinesInt < 1 {
		t.Error("Should have at least 1 total line")
	}
}

func TestBashOutputTool_ClearCache(t *testing.T) {
	// 启动一个简单任务
	bashTool, err := NewBashTool(nil)
	if err != nil {
		t.Fatalf("Failed to create Bash tool: %v", err)
	}

	bashInput := map[string]interface{}{
		"command":    "echo 'Test output'",
		"background": true,
	}

	bashResult := ExecuteToolWithInput(t, bashTool, bashInput)
	bashResult = AssertToolSuccess(t, bashResult)
	taskID, exists := bashResult["task_id"].(string)
	if !exists || taskID == "" {
		t.Fatal("Failed to get task ID from background task")
	}

	// 等待任务完成
	time.Sleep(200 * time.Millisecond)

	// 第一次获取输出
	bashOutputTool, err := NewBashOutputTool(nil)
	if err != nil {
		t.Fatalf("Failed to create BashOutput tool: %v", err)
	}

	outputInput1 := map[string]interface{}{
		"bash_id": taskID,
	}

	result1 := ExecuteToolWithInput(t, bashOutputTool, outputInput1)
	result1 = AssertToolSuccess(t, result1)

	// 第二次获取输出并清除缓存
	outputInput2 := map[string]interface{}{
		"bash_id":     taskID,
		"clear_cache": true,
	}

	result2 := ExecuteToolWithInput(t, bashOutputTool, outputInput2)
	result2 = AssertToolSuccess(t, result2)

	// 验证清除缓存标识
	if _, exists := result2["cache_cleared"]; !exists {
		t.Error("Result should indicate that cache was cleared")
	}
}

func BenchmarkBashOutputTool_GetOutput(b *testing.B) {
	// 先启动一个后台任务
	bashTool, err := NewBashTool(nil)
	if err != nil {
		b.Fatalf("Failed to create Bash tool: %v", err)
	}

	bashInput := map[string]interface{}{
		"command":    "echo 'Benchmark test'",
		"background": true,
	}

	bashResult := ExecuteToolWithInput(&testing.T{}, bashTool, bashInput)
	bashResult = AssertToolSuccess(&testing.T{}, bashResult)
	taskID, exists := bashResult["task_id"].(string)
	if !exists || taskID == "" {
		b.Fatal("Failed to get task ID from background task")
	}

	// 等待任务完成
	time.Sleep(100 * time.Millisecond)

	// 基准测试BashOutput工具
	bashOutputTool, err := NewBashOutputTool(nil)
	if err != nil {
		b.Fatalf("Failed to create BashOutput tool: %v", err)
	}

	input := map[string]interface{}{
		"bash_id": taskID,
	}

	BenchmarkTool(b, bashOutputTool, input)
}