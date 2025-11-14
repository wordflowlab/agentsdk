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
  }
})
