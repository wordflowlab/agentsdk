package backends

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// RouteConfig 路由配置
type RouteConfig struct {
	Prefix  string          // 路径前缀
	Backend BackendProtocol // 对应的后端
}

// CompositeBackend 组合路由后端
// 根据路径前缀将请求路由到不同的后端
// 例如: "/" -> StateBackend, "/memories/" -> StoreBackend, "/workspace/" -> FilesystemBackend
type CompositeBackend struct {
	defaultBackend BackendProtocol
	routes         []RouteConfig // 按前缀长度降序排列,优先匹配最长前缀
}

// NewCompositeBackend 创建 CompositeBackend 实例
func NewCompositeBackend(defaultBackend BackendProtocol, routes []RouteConfig) *CompositeBackend {
	// 按前缀长度降序排序,确保优先匹配最长前缀
	sortedRoutes := make([]RouteConfig, len(routes))
	copy(sortedRoutes, routes)
	sort.Slice(sortedRoutes, func(i, j int) bool {
		return len(sortedRoutes[i].Prefix) > len(sortedRoutes[j].Prefix)
	})

	return &CompositeBackend{
		defaultBackend: defaultBackend,
		routes:         sortedRoutes,
	}
}

// selectBackend 根据路径选择后端
// 返回后端和剥离前缀后的路径
func (b *CompositeBackend) selectBackend(path string) (BackendProtocol, string) {
	for _, route := range b.routes {
		if strings.HasPrefix(path, route.Prefix) {
			// 剥离匹配的前缀
			strippedPath := strings.TrimPrefix(path, route.Prefix)
			// 确保路径以 / 开头
			if strippedPath == "" {
				strippedPath = "/"
			} else if !strings.HasPrefix(strippedPath, "/") {
				strippedPath = "/" + strippedPath
			}
			return route.Backend, strippedPath
		}
	}
	return b.defaultBackend, path
}

// backendWithPrefix 后端及其路由前缀
type backendWithPrefix struct {
	backend BackendProtocol
	prefix  string // 路由前缀,默认后端为空字符串
}

// selectBackendsForPrefix 获取可能包含指定前缀的所有后端
// 返回后端及其对应的路由前缀
func (b *CompositeBackend) selectBackendsForPrefix(prefix string) []backendWithPrefix {
	backends := make(map[BackendProtocol]string)

	// 检查默认后端
	backends[b.defaultBackend] = ""

	// 检查所有路由
	for _, route := range b.routes {
		// 如果路由前缀与搜索前缀有交集,则包含该后端
		if strings.HasPrefix(route.Prefix, prefix) || strings.HasPrefix(prefix, route.Prefix) || prefix == "" {
			backends[route.Backend] = route.Prefix
		}
	}

	result := make([]backendWithPrefix, 0, len(backends))
	for backend, routePrefix := range backends {
		result = append(result, backendWithPrefix{
			backend: backend,
			prefix:  routePrefix,
		})
	}
	return result
}

// ListInfo 实现 BackendProtocol.ListInfo
func (b *CompositeBackend) ListInfo(ctx context.Context, path string) ([]FileInfo, error) {
	// 对于 List 操作,可能需要聚合多个后端的结果
	backends := b.selectBackendsForPrefix(path)

	var allResults []FileInfo
	seen := make(map[string]bool)

	for _, bwp := range backends {
		// 剥离路由前缀
		strippedPath := path
		if bwp.prefix != "" && strings.HasPrefix(path, bwp.prefix) {
			strippedPath = strings.TrimPrefix(path, bwp.prefix)
			if strippedPath == "" {
				strippedPath = "/"
			} else if !strings.HasPrefix(strippedPath, "/") {
				strippedPath = "/" + strippedPath
			}
		}

		results, err := bwp.backend.ListInfo(ctx, strippedPath)
		if err != nil {
			// 某个后端失败不影响其他后端
			continue
		}

		for _, info := range results {
			// 加回路由前缀
			fullPath := info.Path
			if bwp.prefix != "" {
				fullPath = strings.TrimSuffix(bwp.prefix, "/") + info.Path
			}

			if !seen[fullPath] {
				seen[fullPath] = true
				// 创建新的 FileInfo 副本,使用完整路径
				newInfo := info
				newInfo.Path = fullPath
				allResults = append(allResults, newInfo)
			}
		}
	}

	return allResults, nil
}

// Read 实现 BackendProtocol.Read
func (b *CompositeBackend) Read(ctx context.Context, path string, offset, limit int) (string, error) {
	backend, strippedPath := b.selectBackend(path)
	return backend.Read(ctx, strippedPath, offset, limit)
}

// Write 实现 BackendProtocol.Write
func (b *CompositeBackend) Write(ctx context.Context, path, content string) (*WriteResult, error) {
	backend, strippedPath := b.selectBackend(path)
	return backend.Write(ctx, strippedPath, content)
}

// Edit 实现 BackendProtocol.Edit
func (b *CompositeBackend) Edit(ctx context.Context, path, oldStr, newStr string, replaceAll bool) (*EditResult, error) {
	backend, strippedPath := b.selectBackend(path)
	return backend.Edit(ctx, strippedPath, oldStr, newStr, replaceAll)
}

// GrepRaw 实现 BackendProtocol.GrepRaw
func (b *CompositeBackend) GrepRaw(ctx context.Context, pattern, path, glob string) ([]GrepMatch, error) {
	// Grep 操作可能需要搜索多个后端
	backends := b.selectBackendsForPrefix(path)

	var allMatches []GrepMatch

	for _, bwp := range backends {
		// 剥离路由前缀
		strippedPath := path
		if bwp.prefix != "" && strings.HasPrefix(path, bwp.prefix) {
			strippedPath = strings.TrimPrefix(path, bwp.prefix)
			if strippedPath == "" {
				strippedPath = "/"
			} else if !strings.HasPrefix(strippedPath, "/") {
				strippedPath = "/" + strippedPath
			}
		}

		matches, err := bwp.backend.GrepRaw(ctx, pattern, strippedPath, glob)
		if err != nil {
			// 某个后端失败不影响其他后端
			continue
		}

		// 为每个匹配加回路由前缀
		for _, match := range matches {
			if bwp.prefix != "" {
				match.Path = strings.TrimSuffix(bwp.prefix, "/") + match.Path
			}
			allMatches = append(allMatches, match)
		}
	}

	return allMatches, nil
}

// GlobInfo 实现 BackendProtocol.GlobInfo
func (b *CompositeBackend) GlobInfo(ctx context.Context, pattern, path string) ([]FileInfo, error) {
	// Glob 操作可能需要搜索多个后端
	backends := b.selectBackendsForPrefix(path)

	var allResults []FileInfo
	seen := make(map[string]bool)

	for _, bwp := range backends {
		// 剥离路由前缀
		strippedPath := path
		if bwp.prefix != "" && strings.HasPrefix(path, bwp.prefix) {
			strippedPath = strings.TrimPrefix(path, bwp.prefix)
			if strippedPath == "" {
				strippedPath = "/"
			} else if !strings.HasPrefix(strippedPath, "/") {
				strippedPath = "/" + strippedPath
			}
		}

		results, err := bwp.backend.GlobInfo(ctx, pattern, strippedPath)
		if err != nil {
			// 某个后端失败不影响其他后端
			continue
		}

		for _, info := range results {
			// 加回路由前缀
			fullPath := info.Path
			if bwp.prefix != "" {
				fullPath = strings.TrimSuffix(bwp.prefix, "/") + info.Path
			}

			if !seen[fullPath] {
				seen[fullPath] = true
				// 创建新的 FileInfo 副本,使用完整路径
				newInfo := info
				newInfo.Path = fullPath
				allResults = append(allResults, newInfo)
			}
		}
	}

	return allResults, nil
}

// String 返回组合后端的描述
func (b *CompositeBackend) String() string {
	var desc strings.Builder
	desc.WriteString("CompositeBackend{\n")
	for _, route := range b.routes {
		desc.WriteString(fmt.Sprintf("  %s -> %T\n", route.Prefix, route.Backend))
	}
	desc.WriteString(fmt.Sprintf("  default -> %T\n", b.defaultBackend))
	desc.WriteString("}")
	return desc.String()
}
