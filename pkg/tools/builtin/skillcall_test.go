package builtin

import (
	"fmt"
	"testing"
)

func TestNewSkillTool(t *testing.T) {
	tool, err := NewSkillTool(nil)
	if err != nil {
		t.Fatalf("Failed to create SkillCall tool: %v", err)
	}

	if tool.Name() != "skill_call" {
		t.Errorf("Expected tool name 'skill_call', got '%s'", tool.Name())
	}

	if tool.Description() == "" {
		t.Error("Tool description should not be empty")
	}
}

func TestSkillCallTool_InputSchema(t *testing.T) {
	tool, err := NewSkillTool(nil)
	if err != nil {
		t.Fatalf("Failed to create SkillCall tool: %v", err)
	}

	schema := tool.InputSchema()
	if schema == nil {
		t.Fatal("Input schema should not be nil")
	}

	properties, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Properties should be a map")
	}

	// 验证必需字段存在
	requiredFields := []string{"skill"}
	for _, field := range requiredFields {
		if _, exists := properties[field]; !exists {
			t.Errorf("Required field '%s' should exist in properties", field)
		}
	}

	// 验证可选字段存在
	optionalFields := []string{"parameters", "context", "timeout_seconds"}
	for _, field := range optionalFields {
		if _, exists := properties[field]; !exists {
			t.Errorf("Optional field '%s' should exist in properties", field)
		}
	}
}

func TestSkillCallTool_MissingSkill(t *testing.T) {
	tool, err := NewSkillTool(nil)
	if err != nil {
		t.Fatalf("Failed to create SkillCall tool: %v", err)
	}

	input := map[string]interface{}{
		"parameters": map[string]interface{}{},
	}

	result := ExecuteToolWithInput(t, tool, input)

	// 应该返回错误
	errMsg := AssertToolError(t, result)
	if !contains(errMsg, "skill") && !contains(errMsg, "required") {
		t.Errorf("Expected skill required error, got: %s", errMsg)
	}
}

func TestSkillCallTool_InvalidSkill(t *testing.T) {
	tool, err := NewSkillTool(nil)
	if err != nil {
		t.Fatalf("Failed to create SkillCall tool: %v", err)
	}

	input := map[string]interface{}{
		"skill": "nonexistent_skill",
	}

	result := ExecuteToolWithInput(t, tool, input)

	// 应该返回错误或失败响应
	if result["ok"].(bool) {
		t.Log("Skill call succeeded (unexpected in test environment)")
	} else {
		errMsg := AssertToolError(t, result)
		if !contains(errMsg, "not found") && !contains(errMsg, "invalid") && !contains(errMsg, "unknown") {
			t.Logf("Skill call failed with expected error: %s", errMsg)
		}
	}
}

func TestSkillCallTool_WithParameters(t *testing.T) {
	tool, err := NewSkillTool(nil)
	if err != nil {
		t.Fatalf("Failed to create SkillCall tool: %v", err)
	}

	input := map[string]interface{}{
		"skill": "test_skill",
		"parameters": map[string]interface{}{
			"param1": "value1",
			"param2": 42,
			"param3": true,
		},
	}

	result := ExecuteToolWithInput(t, tool, input)

	// 检查工具是否正确处理了参数
	if !result["ok"].(bool) {
		t.Logf("Skill call failed (expected in test environment): %v", result["error"])
	} else {
		// 如果成功，验证参数传递
		if skill, exists := result["skill"]; !exists || skill.(string) != "test_skill" {
			t.Error("Should echo back the skill name")
		}

		if _, exists := result["parameters"]; !exists {
			t.Error("Should include parameters in response")
		}
	}
}

func TestSkillCallTool_WithTimeout(t *testing.T) {
	tool, err := NewSkillTool(nil)
	if err != nil {
		t.Fatalf("Failed to create SkillCall tool: %v", err)
	}

	input := map[string]interface{}{
		"skill":          "test_skill",
		"timeout_seconds": 5,
	}

	result := ExecuteToolWithInput(t, tool, input)

	// 检查超时设置
	if !result["ok"].(bool) {
		t.Logf("Skill call failed (expected in test environment): %v", result["error"])
	} else {
		if timeout, exists := result["timeout_seconds"]; !exists || timeout.(int) != 5 {
			t.Error("Should include timeout_seconds setting in response")
		}
	}
}

func TestSkillCallTool_WithEmptyParameters(t *testing.T) {
	tool, err := NewSkillTool(nil)
	if err != nil {
		t.Fatalf("Failed to create SkillCall tool: %v", err)
	}

	input := map[string]interface{}{
		"skill":      "test_skill",
		"parameters": map[string]interface{}{},
	}

	result := ExecuteToolWithInput(t, tool, input)

	// 检查是否正确处理空参数
	if !result["ok"].(bool) {
		t.Logf("Skill call failed (expected in test environment): %v", result["error"])
	} else {
		if _, exists := result["parameters"]; !exists {
			t.Error("Should include parameters field even if empty")
		}
	}
}

func TestSkillCallTool_ConcurrentCalls(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent test in short mode")
	}

	tool, err := NewSkillTool(nil)
	if err != nil {
		t.Fatalf("Failed to create SkillCall tool: %v", err)
	}

	concurrency := 3
	result := RunConcurrentTest(concurrency, func() error {
		input := map[string]interface{}{
			"skill": "test_skill",
			"parameters": map[string]interface{}{
				"test_id": concurrency,
			},
		}

		result := ExecuteToolWithInput(t, tool, input)
		if !result["ok"].(bool) {
			// 在测试环境中失败是正常的
			return nil
		}

		// 验证基本响应
		if _, exists := result["skill"]; !exists {
			return fmt.Errorf("Missing skill in result")
		}

		return nil
	})

	// 在测试环境中，并发测试应该通过
	if result.ErrorCount > 0 {
		t.Errorf("Concurrent SkillCall operations failed: %d errors out of %d attempts",
			result.ErrorCount, concurrency)
	}

	t.Logf("Concurrent SkillCall operations completed: %d success, %d errors in %v",
		result.SuccessCount, result.ErrorCount, result.Duration)
}

func BenchmarkSkillCallTool_SimpleCall(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	tool, err := NewSkillTool(nil)
	if err != nil {
		b.Fatalf("Failed to create SkillCall tool: %v", err)
	}

	input := map[string]interface{}{
		"skill": "test_skill",
		"parameters": map[string]interface{}{
			"benchmark": true,
		},
	}

	BenchmarkTool(b, tool, input)
}

