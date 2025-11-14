export default defineNuxtConfig({
  // 扩展 Docus 主题 (与 veadk-python 保持一致)
  extends: ['docus'],

  // 应用配置
  app: {
    baseURL: '/agentsdk/'
  },

  // 图片配置: 禁用 IPX
  image: {
    provider: 'none'
  },

  // 关闭 robots.txt 生成, 避免 baseURL + robots 的冲突错误
  robots: {
    robotsTxt: false
  },

  // 为 nuxt-llms 提供一个域名, 避免生成阶段的警告(不影响功能)
  llms: {
    domain: 'https://agentsdk.local'
  },

  // Nuxt Content 配置 - 启用代码高亮
  content: {
    build: {
      markdown: {
        highlight: {
          // 支持的编程语言列表（必须显式声明以覆盖父主题配置）
          langs: [
            'bash', 'sh', 'shell', 'zsh',
            'diff', 'json',
            'javascript', 'js', 'typescript', 'ts',
            'html', 'css', 'vue',
            'markdown', 'md', 'yaml', 'yml',
            'go',        // Go 语言支持（关键）
            'python', 'py',
            'java',
          ],
          // Shiki 主题配置
          theme: {
            default: 'material-theme-lighter',
            dark: 'material-theme-palenight'
          }
        }
      }
    }
  }
})
