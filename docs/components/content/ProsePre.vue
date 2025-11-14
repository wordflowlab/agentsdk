<template>
  <div v-if="language === 'mermaid'">
    <ClientOnly>
      <MermaidRenderer :code="code" />
      <template #fallback>
        <pre class="language-mermaid p-4 overflow-x-auto bg-gray-50 dark:bg-gray-900 rounded"><code>{{ code }}</code></pre>
      </template>
    </ClientOnly>
  </div>
  <pre v-else :class="$props.class"><slot /></pre>
</template>

<script setup lang="ts">

// 定义 MermaidRenderer 组件，确保在 ClientOnly 内部正确渲染
const MermaidRenderer = defineComponent({
  props: {
    code: {
      type: String,
      required: true
    }
  },
  setup(props) {
    const container = ref(null)

    onMounted(async () => {
      console.log('[MermaidRenderer] onMounted', {
        hasCode: !!props.code,
        codeLength: props.code?.length,
        hasContainer: !!container.value
      })

      if (!container.value) {
        console.error('[MermaidRenderer] Container not found')
        return
      }

      if (!props.code || props.code.trim().length === 0) {
        console.error('[MermaidRenderer] No code provided')
        container.value.innerHTML = '<div class="text-red-500 p-4">No Mermaid code provided</div>'
        return
      }

      try {
        const { default: mermaid } = await import('mermaid')

        const isDark = document.documentElement.classList.contains('dark')

        mermaid.initialize({
          startOnLoad: false,
          theme: isDark ? 'dark' : 'default',
          securityLevel: 'loose',
          fontFamily: 'inherit',
          themeVariables: {
            darkMode: isDark
          }
        })

        const id = `mermaid-${Math.random().toString(36).substr(2, 9)}`
        const { svg } = await mermaid.render(id, props.code)

        console.log('[MermaidRenderer] Rendered successfully')

        if (container.value) {
          container.value.innerHTML = svg
        }
      } catch (error: any) {
        console.error('[MermaidRenderer] Error:', error)
        if (container.value) {
          container.value.innerHTML = `<div class="text-red-500 dark:text-red-400 text-sm p-4 bg-red-50 dark:bg-red-900/20 rounded">图表渲染失败: ${error.message || '未知错误'}</div>`
        }
      }
    })

    return () => h('div', { ref: container, class: 'mermaid-diagram my-6 overflow-x-auto' })
  }
})

defineProps({
  code: {
    type: String,
    default: ''
  },
  language: {
    type: String,
    default: null
  },
  filename: {
    type: String,
    default: null
  },
  highlights: {
    type: Array as () => number[],
    default: () => []
  },
  meta: {
    type: String,
    default: null
  },
  class: {
    type: String,
    default: null
  }
})
</script>
