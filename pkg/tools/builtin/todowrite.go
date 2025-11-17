package builtin

import (
	"context"
	"fmt"
	"time"

	"github.com/wordflowlab/agentsdk/pkg/tools"
)

// TodoWriteTool 任务管理工具
// 支持创建和管理结构化任务列表
type TodoWriteTool struct{}

// TodoItem 单个任务项
type TodoItem struct {
	ID          string                 `json:"id"`
	Content     string                 `json:"content"`
	Status      string                 `json:"status"` // "pending", "in_progress", "completed"
	ActiveForm  string                 `json:"activeForm"`
	Priority    int                    `json:"priority,omitempty"`
	CreatedAt   time.Time              `json:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt"`
	CompletedAt *time.Time             `json:"completedAt,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// TodoList 任务列表
type TodoList struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Todos     []TodoItem             `json:"todos"`
	CreatedAt time.Time              `json:"createdAt"`
	UpdatedAt time.Time              `json:"updatedAt"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// NewTodoWriteTool 创建TodoWrite工具
func NewTodoWriteTool(config map[string]interface{}) (tools.Tool, error) {
	return &TodoWriteTool{}, nil
}

func (t *TodoWriteTool) Name() string {
	return "TodoWrite"
}

func (t *TodoWriteTool) Description() string {
	return "创建和管理结构化任务列表"
}

func (t *TodoWriteTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"todos": map[string]interface{}{
				"type": "array",
				"description": "任务项数组，包含content、status、activeForm等字段",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"content": map[string]interface{}{
							"type":        "string",
							"description": "任务描述内容",
						},
						"status": map[string]interface{}{
							"type":        "string",
							"enum":        []string{"pending", "in_progress", "completed"},
							"description": "任务状态",
						},
						"activeForm": map[string]interface{}{
							"type":        "string",
							"description": "任务的主动形式描述（进行中的状态描述）",
						},
						"priority": map[string]interface{}{
							"type":        "integer",
							"description": "任务优先级（数值越大优先级越高）",
						},
					},
					"required": []string{"content", "status", "activeForm"},
				},
			},
			"list_name": map[string]interface{}{
				"type":        "string",
				"description": "任务列表名称，默认为'default'",
			},
			"action": map[string]interface{}{
				"type":        "string",
				"description": "操作类型：create, update, delete, clear，默认为create",
			},
			"todo_id": map[string]interface{}{
				"type":        "string",
				"description": "要更新或删除的任务ID",
			},
		},
		"required": []string{"todos"},
	}
}

func (t *TodoWriteTool) Execute(ctx context.Context, input map[string]interface{}, tc *tools.ToolContext) (interface{}, error) {
	// 验证必需参数
	if err := ValidateRequired(input, []string{"todos"}); err != nil {
		return NewClaudeErrorResponse(err), nil
	}

	action := GetStringParam(input, "action", "create")
	listName := GetStringParam(input, "list_name", "default")
	todoID := GetStringParam(input, "todo_id", "")

	// 获取任务项数据
	todosData, ok := input["todos"].([]interface{})
	if !ok {
		return NewClaudeErrorResponse(fmt.Errorf("todos must be an array")), nil
	}

	// 转换为TodoItem
	todos := make([]TodoItem, 0, len(todosData))
	for _, todoData := range todosData {
		todoMap, ok := todoData.(map[string]interface{})
		if !ok {
			return NewClaudeErrorResponse(fmt.Errorf("each todo must be an object")), nil
		}

		content := GetStringParam(todoMap, "content", "")
		status := GetStringParam(todoMap, "status", "pending")
		activeForm := GetStringParam(todoMap, "activeForm", "")
		priority := GetIntParam(todoMap, "priority", 0)

		if content == "" {
			return NewClaudeErrorResponse(fmt.Errorf("todo content cannot be empty")), nil
		}

		if activeForm == "" {
			return NewClaudeErrorResponse(fmt.Errorf("todo activeForm cannot be empty")), nil
		}

		// 验证状态
		validStatuses := []string{"pending", "in_progress", "completed"}
		statusValid := false
		for _, validStatus := range validStatuses {
			if status == validStatus {
				statusValid = true
				break
			}
		}
		if !statusValid {
			return NewClaudeErrorResponse(
				fmt.Errorf("invalid status: %s", status),
				"支持的状态: pending, in_progress, completed",
			), nil
		}

		todo := TodoItem{
			Content:    content,
			Status:     status,
			ActiveForm: activeForm,
			Priority:   priority,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
			Metadata:   make(map[string]interface{}),
		}

		// 如果有todo_id，使用它
		if id, exists := todoMap["id"]; exists {
			if idStr, ok := id.(string); ok {
				todo.ID = idStr
			}
		}

		// 生成ID（如果没有提供）
		if todo.ID == "" {
			todo.ID = fmt.Sprintf("todo_%d", time.Now().UnixNano())
		}

		// 处理completed状态的完成时间
		if status == "completed" {
			now := time.Now()
			todo.CompletedAt = &now
		}

		todos = append(todos, todo)
	}

	start := time.Now()

	// 获取全局任务列表管理器
	todoManager := GetGlobalTodoManager()

	// 加载现有任务列表
	todoList, err := todoManager.LoadTodoList(listName)
	if err != nil {
		// 如果不存在，创建新的任务列表
		todoList = &TodoList{
			ID:        fmt.Sprintf("list_%s_%d", listName, time.Now().UnixNano()),
			Name:      listName,
			Todos:     []TodoItem{},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Metadata:  make(map[string]interface{}),
		}
	}

	// 执行操作
	var result interface{}
	var operationErr error

	switch action {
	case "create":
		result = t.createTodos(todoList, todos)
		operationErr = todoManager.StoreTodoList(todoList)
	case "update":
		if todoID == "" {
			return NewClaudeErrorResponse(fmt.Errorf("todo_id is required for update action")), nil
		}
		result = t.updateTodo(todoList, todoID, todos[0])
		operationErr = todoManager.StoreTodoList(todoList)
	case "delete":
		if todoID == "" {
			return NewClaudeErrorResponse(fmt.Errorf("todo_id is required for delete action")), nil
		}
		result = t.deleteTodo(todoList, todoID)
		operationErr = todoManager.StoreTodoList(todoList)
	case "clear":
		result = t.clearTodos(todoList)
		operationErr = todoManager.StoreTodoList(todoList)
	default:
		return NewClaudeErrorResponse(
			fmt.Errorf("invalid action: %s", action),
			"支持的操作: create, update, delete, clear",
		), nil
	}

	// 检查操作是否成功
	if operationErr != nil {
		return map[string]interface{}{
			"ok": false,
			"error": fmt.Sprintf("failed to save todo list: %v", operationErr),
			"action": action,
			"list_name": listName,
			"duration_ms": time.Since(start).Milliseconds(),
		}, nil
	}

	duration := time.Since(start)

	// 构建响应
	response := map[string]interface{}{
		"ok": true,
		"action": action,
		"list_name": listName,
		"list_id": todoList.ID,
		"todos": todoList.Todos,
		"total_todos": len(todoList.Todos),
		"duration_ms": duration.Milliseconds(),
		"updated_at": todoList.UpdatedAt.Unix(),
		"storage": "persistent",
		"storage_backend": "FileTodoManager",
	}

	// 添加统计信息
	response["pending_count"] = t.countTodosByStatus(todoList.Todos, "pending")
	response["in_progress_count"] = t.countTodosByStatus(todoList.Todos, "in_progress")
	response["completed_count"] = t.countTodosByStatus(todoList.Todos, "completed")

	// 添加操作结果
	if resultMap, ok := result.(map[string]interface{}); ok {
		for k, v := range resultMap {
			response[k] = v
		}
	}

	return response, nil
}

// createTodos 创建新任务
func (t *TodoWriteTool) createTodos(todoList *TodoList, todos []TodoItem) map[string]interface{} {
	addedTodos := make([]TodoItem, 0, len(todos))

	for _, todo := range todos {
		// 检查是否已存在相同ID的任务
		exists := false
		for _, existing := range todoList.Todos {
			if existing.ID == todo.ID {
				exists = true
				break
			}
		}

		if !exists {
			todoList.Todos = append(todoList.Todos, todo)
			addedTodos = append(addedTodos, todo)
		}
	}

	todoList.UpdatedAt = time.Now()

	return map[string]interface{}{
		"added_count": len(addedTodos),
		"added_todos": addedTodos,
	}
}

// updateTodo 更新任务
func (t *TodoWriteTool) updateTodo(todoList *TodoList, todoID string, updatedTodo TodoItem) map[string]interface{} {
	for i, existing := range todoList.Todos {
		if existing.ID == todoID {
			// 保留创建时间
			updatedTodo.CreatedAt = existing.CreatedAt
			updatedTodo.ID = existing.ID

			// 更新时间
			updatedTodo.UpdatedAt = time.Now()

			// 如果状态变为completed，设置完成时间
			if updatedTodo.Status == "completed" && existing.Status != "completed" {
				now := time.Now()
				updatedTodo.CompletedAt = &now
			} else if updatedTodo.Status != "completed" {
				updatedTodo.CompletedAt = nil
			}

			// 保留元数据
			if existing.Metadata != nil {
				for k, v := range existing.Metadata {
					if _, exists := updatedTodo.Metadata[k]; !exists {
						updatedTodo.Metadata[k] = v
					}
				}
			}

			todoList.Todos[i] = updatedTodo
			todoList.UpdatedAt = time.Now()

			return map[string]interface{}{
				"updated": true,
				"previous_status": existing.Status,
				"new_status": updatedTodo.Status,
			}
		}
	}

	return map[string]interface{}{
		"updated": false,
		"reason": "todo not found",
	}
}

// deleteTodo 删除任务
func (t *TodoWriteTool) deleteTodo(todoList *TodoList, todoID string) map[string]interface{} {
	for i, existing := range todoList.Todos {
		if existing.ID == todoID {
			// 删除任务
			todoList.Todos = append(todoList.Todos[:i], todoList.Todos[i+1:]...)
			todoList.UpdatedAt = time.Now()

			return map[string]interface{}{
				"deleted": true,
				"deleted_todo": existing,
			}
		}
	}

	return map[string]interface{}{
		"deleted": false,
		"reason": "todo not found",
	}
}

// clearTodos 清空任务列表
func (t *TodoWriteTool) clearTodos(todoList *TodoList) map[string]interface{} {
	deletedCount := len(todoList.Todos)
	todoList.Todos = []TodoItem{}
	todoList.UpdatedAt = time.Now()

	return map[string]interface{}{
		"deleted_count": deletedCount,
		"action": "cleared_all_todos",
	}
}

// countTodosByStatus 按状态统计任务数量
func (t *TodoWriteTool) countTodosByStatus(todos []TodoItem, status string) int {
	count := 0
	for _, todo := range todos {
		if todo.Status == status {
			count++
		}
	}
	return count
}

func (t *TodoWriteTool) Prompt() string {
	return `创建和管理结构化任务列表。

功能特性：
- 创建、更新、删除任务项
- 支持任务状态管理（pending, in_progress, completed）
- 任务优先级设置
- 自动ID生成和时间戳
- 任务统计和进度跟踪

使用指南：
- todos: 必需参数，任务项数组
- list_name: 可选参数，任务列表名称
- action: 可选参数，操作类型（create/update/delete/clear）
- todo_id: 可选参数，要操作的任务ID

任务状态：
- pending: 待处理任务
- in_progress: 进行中任务
- completed: 已完成任务

任务字段：
- content: 任务描述内容（必需）
- status: 任务状态（必需）
- activeForm: 任务的主动形式描述（必需）
- priority: 任务优先级（可选）

注意事项：
- 使用持久化存储系统，数据安全可靠
- 支持任务完成时间自动记录
- 提供详细的任务统计信息
- 支持任务列表的备份和恢复

存储特性：
- 基于文件系统的JSON格式存储
- 自动备份和恢复机制
- 支持多任务列表管理
- 数据导入导出功能
- 集成全局存储管理器`
}