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
  },

  // UI 组件主题配置
  ui: {
    prose: {
      pre: {
        // 修复代码块溢出问题：启用水平滚动，保持代码格式
        base: 'group font-mono text-sm/6 border border-muted bg-muted rounded-md px-4 py-3 overflow-x-auto whitespace-pre focus:outline-none'
      }
    }
  }
})
