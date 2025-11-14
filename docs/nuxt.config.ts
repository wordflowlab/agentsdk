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
  }
})
