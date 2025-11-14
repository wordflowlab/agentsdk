export default defineNuxtPlugin(async () => {
  // 获取导航数据
  const { data: navigation } = await useAsyncData('navigation', () => {
    return queryContent()
      .where({ navigation: { $ne: false } })
      .only(['title', '_path', 'navigation'])
      .find()
  })

  // 构建导航树
  const buildNavigationTree = (pages: any[]) => {
    const pathMap = new Map<string, any>()

    // 先创建所有分类节点
    pages.forEach((page: any) => {
      const path = page._path
      // 移除开头的 /agentsdk/ 和 /，以及 index
      const cleanPath = path.replace(/^\/agentsdk\//, '').replace(/^\//, '')
      const parts = cleanPath.split('/').filter((p: string) => p && p !== 'index')

      if (parts.length > 0) {
        const category = parts[0]
        if (!pathMap.has(category)) {
          // 查找分类的 index 页面或 overview 页面
          const categoryPage = pages.find((p: any) => {
            const pPath = p._path.replace(/^\/agentsdk\//, '').replace(/^\//, '')
            const pParts = pPath.split('/').filter((pp: string) => pp && pp !== 'index')
            return pParts.length === 1 && pParts[0] === category
          })

          pathMap.set(category, {
            title: categoryPage?.title || category,
            _path: categoryPage?._path || page._path,
            icon: categoryPage?.navigation?.icon,
            children: []
          })
        }
      }
    })

    // 将页面添加到对应的分类
    pages.forEach((page: any) => {
      const path = page._path
      const cleanPath = path.replace(/^\/agentsdk\//, '').replace(/^\//, '')
      const parts = cleanPath.split('/').filter((p: string) => p && p !== 'index')

      if (parts.length === 1) {
        // 这是分类页面本身
        const category = parts[0]
        const node = pathMap.get(category)
        if (node) {
          node.title = page.title || node.title
          node._path = page._path
          node.icon = page.navigation?.icon || node.icon
        }
      } else if (parts.length > 1) {
        // 这是分类下的子页面
        const category = parts[0]
        const node = pathMap.get(category)
        if (node) {
          node.children.push({
            title: page.title,
            _path: page._path,
            icon: page.navigation?.icon
          })
        }
      }
    })

    // 转换为数组并排序
    return Array.from(pathMap.values()).sort((a, b) => {
      // 按数字前缀排序（如果有，如 1.introduction）
      const aMatch = a._path.match(/\/(\d+)\./)
      const bMatch = b._path.match(/\/(\d+)\./)
      const aNum = aMatch ? parseInt(aMatch[1]) : 999
      const bNum = bMatch ? parseInt(bMatch[1]) : 999
      return aNum - bNum
    })
  }

  // 提供 navigation
  const navigationTree = computed(() => {
    if (!navigation.value) return []
    return buildNavigationTree(navigation.value)
  })

  provide('navigation', navigationTree)
})

