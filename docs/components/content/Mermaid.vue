<template>
  <div class="mermaid-wrapper my-6">
    <div v-if="!rendered" class="flex justify-center items-center py-8">
      <div class="animate-pulse text-gray-500 dark:text-gray-400">
        加载图表中...
      </div>
    </div>
    <pre ref="el" class="mermaid" :style="{ display: rendered ? 'flex' : 'none', justifyContent: 'center' }">
      <slot />
    </pre>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useColorMode } from '#imports'

const el = ref<HTMLElement | null>(null)
const rendered = ref(false)
const colorMode = useColorMode()

let mermaid: any = null

async function renderDiagram() {
  if (!el.value) return

  // 如果已经渲染过，跳过
  if (el.value.querySelector('svg')) return

  try {
    // 动态导入 mermaid
    if (!mermaid) {
      const module = await import('mermaid')
      mermaid = module.default
    }

    // 配置 mermaid
    const isDark = colorMode.value === 'dark'
    mermaid.initialize({
      startOnLoad: false,
      theme: isDark ? 'dark' : 'default',
      securityLevel: 'loose',
      fontFamily: 'ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif',
      themeVariables: isDark ? {
        primaryColor: '#34d399',
        primaryTextColor: '#fff',
        primaryBorderColor: '#34d399',
        lineColor: '#60a5fa',
        secondaryColor: '#3b82f6',
        tertiaryColor: '#1f2937'
      } : {
        primaryColor: '#10b981',
        primaryTextColor: '#fff',
        primaryBorderColor: '#10b981',
        lineColor: '#3b82f6',
        secondaryColor: '#60a5fa',
        tertiaryColor: '#f3f4f6'
      }
    })

    // 移除注释节点
    for (const child of Array.from(el.value.childNodes)) {
      if (child.nodeType === Node.COMMENT_NODE) {
        el.value.removeChild(child)
      }
    }

    // 渲染图表
    await mermaid.run({ nodes: [el.value] })
    rendered.value = true
  } catch (e: any) {
    console.error('Mermaid rendering error:', e)
    // 显示错误信息
    if (el.value) {
      el.value.innerHTML = `<div class="text-red-500 dark:text-red-400 p-4">图表渲染失败: ${e.message}</div>`
      rendered.value = true
    }
  }
}

// 监听主题变化，重新渲染
watch(() => colorMode.value, async () => {
  if (el.value && rendered.value) {
    const svg = el.value.querySelector('svg')
    if (svg) {
      svg.remove()
      rendered.value = false
      await renderDiagram()
    }
  }
})

onMounted(() => {
  renderDiagram()
})
</script>

<style scoped>
.mermaid-wrapper {
  @apply overflow-x-auto rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 p-4;
}

.mermaid {
  @apply m-0 bg-transparent;
}

/* 确保 mermaid 图表样式不受全局 pre 样式影响 */
.mermaid-wrapper pre {
  background: transparent !important;
  border: none !important;
  padding: 0 !important;
  margin: 0 !important;
}
</style>
