package builtin

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestNewExitPlanModeTool(t *testing.T) {
	tool, err := NewExitPlanModeTool(nil)
	if err != nil {
		t.Fatalf("Failed to create ExitPlanMode tool: %v", err)
	}

	if tool.Name() != "ExitPlanMode" {
		t.Errorf("Expected tool name 'ExitPlanMode', got '%s'", tool.Name())
	}

	if tool.Description() == "" {
		t.Error("Tool description should not be empty")
	}
}

func TestExitPlanModeTool_InputSchema(t *testing.T) {
	tool, err := NewExitPlanModeTool(nil)
	if err != nil {
		t.Fatalf("Failed to create ExitPlanMode tool: %v", err)
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
	requiredFields := []string{"plan"}
	for _, field := range requiredFields {
		if _, exists := properties[field]; !exists {
			t.Errorf("Required field '%s' should exist in properties", field)
		}
	}

	// 验证可选字段存在
	optionalFields := []string{"plan_id", "estimated_duration", "dependencies", "risks", "success_criteria", "confirmation_required"}
	for _, field := range optionalFields {
		if _, exists := properties[field]; !exists {
			t.Errorf("Optional field '%s' should exist in properties", field)
		}
	}

	// 验证required字段
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

	if len(requiredArray) != 1 || requiredArray[0] != "plan" {
		t.Errorf("Required should contain only 'plan', got %v", requiredArray)
	}
}

func TestExitPlanModeTool_BasicPlanCreation(t *testing.T) {
	tool, err := NewExitPlanModeTool(nil)
	if err != nil {
		t.Fatalf("Failed to create ExitPlanMode tool: %v", err)
	}

	input := map[string]interface{}{
		"plan": `# Implementation Plan

## Phase 1: Setup (2 hours)
- Initialize project structure
- Set up development environment
- Install dependencies

## Phase 2: Implementation (1 week)
- Develop core functionality
- Write unit tests
- Create documentation

## Phase 3: Testing (2 days)
- Integration testing
- Performance testing
- User acceptance testing

## Phase 4: Deployment (1 day)
- Prepare production environment
- Deploy application
- Monitor and verify`,
		"estimated_duration": "2 weeks",
		"confirmation_required": true,
	}

	result := ExecuteToolWithInput(t, tool, input)
	result = AssertToolSuccess(t, result)

	// 验证基本响应字段
	if planID, exists := result["plan_id"]; !exists {
		t.Error("Result should contain 'plan_id' field")
	} else if planIDStr, ok := planID.(string); !ok || planIDStr == "" {
		t.Error("plan_id should be a non-empty string")
	}

	if status, exists := result["status"]; !exists {
		t.Error("Result should contain 'status' field")
	} else if statusStr, ok := status.(string); !ok || statusStr != "pending_approval" {
		t.Errorf("Expected status 'pending_approval', got %v", status)
	}

	if confirmation, exists := result["confirmation_required"]; !exists {
		t.Error("Result should contain 'confirmation_required' field")
	} else if confirmationBool, ok := confirmation.(bool); !ok || confirmationBool != true {
		t.Errorf("Expected confirmation_required=true, got %v", confirmation)
	}

	// 验证持久化存储信息
	if storage, exists := result["storage"]; !exists || storage.(string) != "persistent" {
		t.Error("Should indicate persistent storage")
	}

	if storageBackend, exists := result["storage_backend"]; !exists {
		t.Error("Result should contain 'storage_backend' field")
	} else if storageBackendStr, ok := storageBackend.(string); !ok {
		t.Errorf("storage_backend should be a string, got %T", storageBackend)
	} else {
		t.Logf("Debug: storage_backend = %s", storageBackendStr)
		// 验证是否是期望的后端类型
		expectedBackends := []string{"PlanManager", "FilePlanManager", "DefaultPlanManager"}
		found := false
		for _, expected := range expectedBackends {
			if storageBackendStr == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected PlanManager backend, got %s", storageBackendStr)
		}
	}
}

func TestExitPlanModeTool_ComprehensivePlan(t *testing.T) {
	tool, err := NewExitPlanModeTool(nil)
	if err != nil {
		t.Fatalf("Failed to create ExitPlanMode tool: %v", err)
	}

	input := map[string]interface{}{
		"plan": `# Website Redesign Plan

## Objectives
- Improve user experience
- Modernize design
- Increase performance

## Implementation Steps
1. **Design Phase** (1 week)
   - Create wireframes
   - Design mockups
   - Client approval

2. **Development Phase** (3 weeks)
   - Frontend development
   - Backend integration
   - Content migration

3. **Testing Phase** (1 week)
   - Functionality testing
   - Performance optimization
   - Security testing

4. **Launch Phase** (2 days)
   - Production deployment
   - User training
   - Support handover`,

		"plan_id":              "website_redesign_2024",
		"estimated_duration":   "5 weeks",
		"dependencies":         []string{"Content approval from marketing team", "Domain and hosting setup", "SSL certificate"},
		"risks":                []string{"Scope creep during development", "Browser compatibility issues", "Content migration data loss"},
		"success_criteria":     []string{"Page load time < 3 seconds", "Mobile responsive design", "100% content migration accuracy", "No security vulnerabilities"},
		"confirmation_required": false,
	}

	result := ExecuteToolWithInput(t, tool, input)
	result = AssertToolSuccess(t, result)

	// 验证自定义计划ID
	if result["plan_id"] != "website_redesign_2024" {
		t.Errorf("Expected custom plan_id, got %v", result["plan_id"])
	}

	// 验证估算时间
	if result["estimated_duration"] != "5 weeks" {
		t.Errorf("Expected estimated_duration='5 weeks', got %v", result["estimated_duration"])
	}

	// 验证依赖项
	if dependencies, exists := result["dependencies"]; !exists {
		t.Error("Result should contain 'dependencies' field")
	} else if depsArray, ok := dependencies.([]string); !ok || len(depsArray) != 3 {
		t.Errorf("Expected 3 dependencies, got %v", dependencies)
	}

	// 验证风险
	if risks, exists := result["risks"]; !exists {
		t.Error("Result should contain 'risks' field")
	} else if risksArray, ok := risks.([]string); !ok || len(risksArray) != 3 {
		t.Errorf("Expected 3 risks, got %v", risks)
	}

	// 验证成功标准
	if successCriteria, exists := result["success_criteria"]; !exists {
		t.Error("Result should contain 'success_criteria' field")
	} else if criteriaArray, ok := successCriteria.([]string); !ok || len(criteriaArray) != 4 {
		t.Errorf("Expected 4 success criteria, got %v", successCriteria)
	}

	// 验证确认设置
	if result["confirmation_required"] != false {
		t.Errorf("Expected confirmation_required=false, got %v", result["confirmation_required"])
	}
}

func TestExitPlanModeTool_MissingPlan(t *testing.T) {
	tool, err := NewExitPlanModeTool(nil)
	if err != nil {
		t.Fatalf("Failed to create ExitPlanMode tool: %v", err)
	}

	input := map[string]interface{}{
		"estimated_duration": "1 week",
		"plan_id":            "test_plan",
	}

	result := ExecuteToolWithInput(t, tool, input)

	// 应该返回错误
	errMsg := AssertToolError(t, result)
	if !strings.Contains(strings.ToLower(errMsg), "plan") && !strings.Contains(strings.ToLower(errMsg), "required") {
		t.Errorf("Expected plan required error, got: %s", errMsg)
	}
}

func TestExitPlanModeTool_EmptyPlan(t *testing.T) {
	tool, err := NewExitPlanModeTool(nil)
	if err != nil {
		t.Fatalf("Failed to create ExitPlanMode tool: %v", err)
	}

	input := map[string]interface{}{
		"plan": "", // 空计划
	}

	result := ExecuteToolWithInput(t, tool, input)

	// 应该返回错误
	errMsg := AssertToolError(t, result)
	if !strings.Contains(strings.ToLower(errMsg), "plan") && !strings.Contains(strings.ToLower(errMsg), "empty") {
		t.Errorf("Expected plan empty error, got: %s", errMsg)
	}
}

func TestExitPlanModeTool_AutoGeneratedPlanID(t *testing.T) {
	tool, err := NewExitPlanModeTool(nil)
	if err != nil {
		t.Fatalf("Failed to create ExitPlanMode tool: %v", err)
	}

	input := map[string]interface{}{
		"plan": "Simple test plan",
	}

	result := ExecuteToolWithInput(t, tool, input)
	result = AssertToolSuccess(t, result)

	// 验证自动生成的计划ID格式
	planID := result["plan_id"].(string)
	if !strings.HasPrefix(planID, "plan_") {
		t.Errorf("Generated plan_id should start with 'plan_', got: %s", planID)
	}
}

func TestExitPlanModeTool_Timestamps(t *testing.T) {
	tool, err := NewExitPlanModeTool(nil)
	if err != nil {
		t.Fatalf("Failed to create ExitPlanMode tool: %v", err)
	}

	beforeCreate := time.Now()

	input := map[string]interface{}{
		"plan": "Test plan with timestamps",
	}

	result := ExecuteToolWithInput(t, tool, input)
	result = AssertToolSuccess(t, result)

	afterCreate := time.Now()

	// 验证时间戳字段
	if updatedAt, exists := result["updated_at"]; !exists {
		t.Error("Result should contain 'updated_at' field")
	} else if updatedAtTime, ok := updatedAt.(time.Time); !ok {
		t.Error("updated_at should be a Time")
	} else {
		if updatedAtTime.Before(beforeCreate) {
			t.Error("updated_at should be after creation time")
		}
		if updatedAtTime.After(afterCreate.Add(time.Second)) {
			t.Error("updated_at should be within reasonable time")
		}
	}
}

func TestExitPlanModeTool_ConcurrentPlanCreation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent test in short mode")
	}

	tool, err := NewExitPlanModeTool(nil)
	if err != nil {
		t.Fatalf("Failed to create ExitPlanMode tool: %v", err)
	}

	concurrency := 3
	result := RunConcurrentTest(concurrency, func() error {
		input := map[string]interface{}{
			"plan":          fmt.Sprintf("Concurrent test plan %d", time.Now().UnixNano()),
			"plan_id":       fmt.Sprintf("concurrent_plan_%d", time.Now().UnixNano()),
			"dependencies":  []string{"Test dependency"},
			"risks":         []string{"Test risk"},
			"success_criteria": []string{"Test success"},
		}

		result := ExecuteToolWithInput(t, tool, input)
		if !result["ok"].(bool) {
			return fmt.Errorf("ExitPlanMode operation failed")
		}

		// 验证基本响应
		if _, exists := result["plan_id"]; !exists {
			return fmt.Errorf("Missing plan_id in result")
		}

		if result["status"].(string) != "pending_approval" {
			return fmt.Errorf("Expected pending_approval status")
		}

		return nil
	})

	if result.ErrorCount > 0 {
		t.Errorf("Concurrent ExitPlanMode operations failed: %d errors out of %d attempts",
			result.ErrorCount, concurrency)
	}

	t.Logf("Concurrent ExitPlanMode operations completed: %d success, %d errors in %v",
		result.SuccessCount, result.ErrorCount, result.Duration)
}

func TestExitPlanModeTool_Metadata(t *testing.T) {
	tool, err := NewExitPlanModeTool(nil)
	if err != nil {
		t.Fatalf("Failed to create ExitPlanMode tool: %v", err)
	}

	input := map[string]interface{}{
		"plan": "Test plan for metadata validation",
	}

	result := ExecuteToolWithInput(t, tool, input)
	result = AssertToolSuccess(t, result)

	// 验证元数据字段
	if metadata, exists := result["metadata"]; !exists {
		t.Error("Result should contain 'metadata' field")
	} else if metadataMap, ok := metadata.(map[string]interface{}); !ok {
		t.Error("metadata should be a map")
	} else {
		// 验证特定的元数据字段
		if exitCall, exists := metadataMap["exit_plan_mode_call"]; !exists {
			t.Error("metadata should contain 'exit_plan_mode_call' field")
		} else if exitCall != true {
			t.Error("exit_plan_mode_call should be true")
		}
	}

	// 验证代理和会话信息
	if agentID, exists := result["agent_id"]; !exists || agentID.(string) != "agent_default" {
		t.Error("Should use default agent_id")
	}

	if sessionID, exists := result["session_id"]; !exists || sessionID.(string) != "session_default" {
		t.Error("Should use default session_id")
	}
}

func TestExitPlanModeTool_ArrayFieldHandling(t *testing.T) {
	tool, err := NewExitPlanModeTool(nil)
	if err != nil {
		t.Fatalf("Failed to create ExitPlanMode tool: %v", err)
	}

	// 测试空数组
	input := map[string]interface{}{
		"plan":         "Test plan with empty arrays",
		"dependencies": []string{},
		"risks":        []string{},
		"success_criteria": []string{},
	}

	result := ExecuteToolWithInput(t, tool, input)
	result = AssertToolSuccess(t, result)

	// 验证空数组字段存在
	dependencies := result["dependencies"].([]string)
	if len(dependencies) != 0 {
		t.Errorf("Expected empty dependencies array, got %d items", len(dependencies))
	}

	risks := result["risks"].([]string)
	if len(risks) != 0 {
		t.Errorf("Expected empty risks array, got %d items", len(risks))
	}

	successCriteria := result["success_criteria"].([]string)
	if len(successCriteria) != 0 {
		t.Errorf("Expected empty success_criteria array, got %d items", len(successCriteria))
	}
}

func BenchmarkExitPlanModeTool_CreatePlan(b *testing.B) {
	tool, err := NewExitPlanModeTool(nil)
	if err != nil {
		b.Fatalf("Failed to create ExitPlanMode tool: %v", err)
	}

	input := map[string]interface{}{
		"plan": `# Benchmark Plan

## Implementation Steps
1. Setup environment
2. Write code
3. Test functionality
4. Deploy application`,
		"dependencies":      []string{"Dependency 1", "Dependency 2"},
		"risks":             []string{"Risk 1", "Risk 2"},
		"success_criteria": []string{"Criteria 1", "Criteria 2"},
	}

	BenchmarkTool(b, tool, input)
}

func BenchmarkExitPlanModeTool_ComplexPlan(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping complex plan benchmark in short mode")
	}

	tool, err := NewExitPlanModeTool(nil)
	if err != nil {
		b.Fatalf("Failed to create ExitPlanMode tool: %v", err)
	}

	input := map[string]interface{}{
		"plan": `# Complex Implementation Plan

## Phase 1: Discovery and Planning (2 weeks)
### Stakeholder Interviews
- Conduct interviews with key stakeholders
- Gather requirements and expectations
- Document business processes

### Technical Analysis
- Review current architecture
- Identify technical constraints
- Define technical requirements

### Risk Assessment
- Identify potential risks
- Develop mitigation strategies
- Create contingency plans

## Phase 2: Design (3 weeks)
### System Architecture
- Design system architecture
- Create data flow diagrams
- Define API specifications

### User Interface Design
- Create wireframes
- Develop mockups
- Design responsive layouts

### Database Design
- Design database schema
- Create ER diagrams
- Optimize query performance

## Phase 3: Development (8 weeks)
### Backend Development
- Implement core business logic
- Develop API endpoints
- Integrate third-party services

### Frontend Development
- Build user interface
- Implement responsive design
- Add interactive features

### Testing
- Unit testing
- Integration testing
- Performance testing

## Phase 4: Deployment (2 weeks)
### Production Setup
- Configure production environment
- Set up monitoring
- Implement backup strategies

### Launch
- Deploy to production
- Monitor performance
- Address issues`,

		"plan_id":            "complex_implementation_plan",
		"estimated_duration": "15 weeks",
		"dependencies": []string{
			"Stakeholder approval",
			"Technical resources allocation",
			"Development environment setup",
			"Third-party service agreements",
		},
		"risks": []string{
			"Scope creep during development",
			"Technical integration challenges",
			"Resource availability constraints",
			"Timeline delays due to dependencies",
			"Budget overruns",
		},
		"success_criteria": []string{
			"All requirements implemented",
			"Performance benchmarks met",
			"Security standards satisfied",
			"User acceptance achieved",
			"Documentation complete",
			"Training materials prepared",
		},
	}

	BenchmarkTool(b, tool, input)
}