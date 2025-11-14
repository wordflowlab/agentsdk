export default defineAppConfig({
  // Docus 配置
  seo: {
    title: 'AgentSDK',
    description: '企业级AI Agent运行时框架，事件驱动、云端沙箱、安全可控',
    titleTemplate: '%s - AgentSDK',
  },

  header: {
    title: 'AgentSDK',
    logo: {
      alt: 'AgentSDK Logo',
      light: '',
      dark: ''
    },
  },

  github: {
    url: 'https://github.com/wordflowlab/agentsdk',
    branch: 'main',
    rootDir: 'docs'
  },

  socials: {
    github: 'https://github.com/wordflowlab/agentsdk',
  },

  toc: {
    title: '本页目录',
  }
})
