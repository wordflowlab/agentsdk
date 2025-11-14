package memory

import (
	"path/filepath"
	"strings"
)

// Scope 描述一段记忆所属的逻辑作用域,用于生成 namespace 字符串。
//
// 设计目标:
// - 支持多用户系统(user级隔离)
// - 支持项目级、资源级(文章/小说/歌曲/PPT等)场景
// - 支持用户专属 vs 全局共享两种模式
//
// 注意: Scope 只负责生成 namespace, 最终路径仍然是:
//   /memories/<BaseNamespace>/<namespace>/<file>
// 其中 BaseNamespace 由 AgentMemoryMiddleware 决定, 通常为:
//   - ""                  : 无用户隔离(单用户或纯共享)
//   - "users/<user-id>"   : 多用户系统中的用户私有前缀
type Scope struct {
	// UserID 当前用户ID,用于业务侧记录/审计; 是否叠加到路径中由 Shared 控制。
	UserID string

	// ProjectID 项目标识,为空表示不绑定项目。
	ProjectID string

	// ResourceType 资源类型: "article" | "novel" | "song" | "ppt" | ...
	ResourceType string

	// ResourceID 资源ID,与 ResourceType 配合使用。
	ResourceID string

	// Shared 是否作为全局/共享记忆:
	// - Shared=false: 生成的 namespace 不以 "/" 开头, 会叠加 BaseNamespace(通常是 users/<user-id>)。
	// - Shared=true : 生成的 namespace 以 "/" 开头, 不叠加 BaseNamespace, 落在全局共享空间。
	Shared bool
}

// Namespace 根据 Scope 生成 namespace 字符串。
//
// 规则:
// - ProjectID 不为空时: 追加 "projects/<project-id>"
// - ResourceType & ResourceID 不为空时: 追加 "resources/<type>/<id>"
// - Shared=false => 不加前导 "/", 交由 BaseNamespace 叠加 (用户级/租户级记忆)
// - Shared=true  => 加前导 "/", 作为全局/共享记忆
//
// 示例:
//   Scope{UserID:"alice", ProjectID:"demo", Shared:false}.Namespace()
//     -> "projects/demo"
//   Scope{ProjectID:"demo", Shared:true}.Namespace()
//     -> "/projects/demo"
//   Scope{ResourceType:"article", ResourceID:"abc", Shared:false}.Namespace()
//     -> "resources/article/abc"
//   Scope{ProjectID:"demo", ResourceType:"article", ResourceID:"abc", Shared:true}.Namespace()
//     -> "/projects/demo/resources/article/abc"
func (s Scope) Namespace() string {
	var parts []string

	if strings.TrimSpace(s.ProjectID) != "" {
		parts = append(parts, "projects", strings.TrimSpace(s.ProjectID))
	}

	if strings.TrimSpace(s.ResourceType) != "" && strings.TrimSpace(s.ResourceID) != "" {
		parts = append(parts,
			"resources",
			strings.TrimSpace(s.ResourceType),
			strings.TrimSpace(s.ResourceID),
		)
	}

	if len(parts) == 0 {
		// 未绑定项目或资源, 返回空字符串
		return ""
	}

	ns := filepath.ToSlash(filepath.Join(parts...))
	if s.Shared {
		// 共享/全局命名空间, 使用前导 "/" 阻止 BaseNamespace 叠加
		return "/" + ns
	}
	// 用户级命名空间, 不加 "/", 由 BaseNamespace 叠加
	return ns
}

