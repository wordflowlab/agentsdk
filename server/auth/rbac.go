package auth

import (
	"context"
	"sync"
)

// Permission 权限
type Permission struct {
	Resource string   `json:"resource"` // e.g., "agents", "workflows"
	Actions  []string `json:"actions"`  // e.g., ["create", "read", "update", "delete"]
}

// Role 角色
type Role struct {
	Name        string       `json:"name"`
	Description string       `json:"description,omitempty"`
	Permissions []Permission `json:"permissions"`
}

// RBAC 基于角色的访问控制
type RBAC struct {
	mu    sync.RWMutex
	roles map[string]*Role
}

// NewRBAC 创建 RBAC 实例
func NewRBAC() *RBAC {
	rbac := &RBAC{
		roles: make(map[string]*Role),
	}

	// 注册默认角色
	rbac.RegisterDefaultRoles()

	return rbac
}

// RegisterDefaultRoles 注册默认角色
func (r *RBAC) RegisterDefaultRoles() {
	// Admin 角色 - 完全权限
	r.AddRole(&Role{
		Name:        "admin",
		Description: "Administrator with full access",
		Permissions: []Permission{
			{Resource: "*", Actions: []string{"*"}},
		},
	})

	// User 角色 - 基础权限
	r.AddRole(&Role{
		Name:        "user",
		Description: "Regular user with basic access",
		Permissions: []Permission{
			{Resource: "agents", Actions: []string{"create", "read", "update", "delete"}},
			{Resource: "memory", Actions: []string{"create", "read", "update", "delete"}},
			{Resource: "sessions", Actions: []string{"create", "read", "update", "delete"}},
			{Resource: "workflows", Actions: []string{"read", "execute"}},
			{Resource: "tools", Actions: []string{"read", "execute"}},
		},
	})

	// Viewer 角色 - 只读权限
	r.AddRole(&Role{
		Name:        "viewer",
		Description: "Read-only access",
		Permissions: []Permission{
			{Resource: "*", Actions: []string{"read"}},
		},
	})

	// Developer 角色 - 开发者权限
	r.AddRole(&Role{
		Name:        "developer",
		Description: "Developer with extended access",
		Permissions: []Permission{
			{Resource: "agents", Actions: []string{"*"}},
			{Resource: "workflows", Actions: []string{"*"}},
			{Resource: "tools", Actions: []string{"*"}},
			{Resource: "eval", Actions: []string{"*"}},
		},
	})
}

// AddRole 添加角色
func (r *RBAC) AddRole(role *Role) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.roles[role.Name] = role
}

// GetRole 获取角色
func (r *RBAC) GetRole(name string) (*Role, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	role, exists := r.roles[name]
	return role, exists
}

// HasPermission 检查用户是否有指定权限
func (r *RBAC) HasPermission(ctx context.Context, user *User, resource string, action string) bool {
	if user == nil {
		return false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, roleName := range user.Roles {
		role, exists := r.roles[roleName]
		if !exists {
			continue
		}

		// 检查角色权限
		if r.roleHasPermission(role, resource, action) {
			return true
		}
	}

	return false
}

// roleHasPermission 检查角色是否有指定权限
func (r *RBAC) roleHasPermission(role *Role, resource string, action string) bool {
	for _, perm := range role.Permissions {
		// 检查资源匹配
		if perm.Resource == "*" || perm.Resource == resource {
			// 检查操作匹配
			for _, a := range perm.Actions {
				if a == "*" || a == action {
					return true
				}
			}
		}
	}
	return false
}

// CheckPermission 检查权限，返回错误
func (r *RBAC) CheckPermission(ctx context.Context, user *User, resource string, action string) error {
	if !r.HasPermission(ctx, user, resource, action) {
		return ErrUnauthorized
	}
	return nil
}

// GetUserPermissions 获取用户的所有权限
func (r *RBAC) GetUserPermissions(user *User) []Permission {
	if user == nil {
		return nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	permMap := make(map[string]map[string]bool) // resource -> actions

	for _, roleName := range user.Roles {
		role, exists := r.roles[roleName]
		if !exists {
			continue
		}

		for _, perm := range role.Permissions {
			if _, ok := permMap[perm.Resource]; !ok {
				permMap[perm.Resource] = make(map[string]bool)
			}
			for _, action := range perm.Actions {
				permMap[perm.Resource][action] = true
			}
		}
	}

	// 转换为 Permission 列表
	var permissions []Permission
	for resource, actions := range permMap {
		var actionList []string
		for action := range actions {
			actionList = append(actionList, action)
		}
		permissions = append(permissions, Permission{
			Resource: resource,
			Actions:  actionList,
		})
	}

	return permissions
}
