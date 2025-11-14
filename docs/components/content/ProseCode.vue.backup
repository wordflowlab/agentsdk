<template>
  <!-- 如果是mermaid代码块，使用Mermaid组件渲染 -->
  <Mermaid v-if="language === 'mermaid'" :code="code" />

  <!-- 否则使用默认的代码高亮 -->
  <pre v-else :class="`language-${language}`"><code><slot /></code></pre>
</template>

<script setup lang="ts">
defineProps<{
  code?: string
  language?: string
  filename?: string
  highlights?: Array<number>
  meta?: string
}>()
</script>

<style scoped>
pre {
  @apply rounded-lg p-4 overflow-x-auto my-4;
}

code {
  @apply text-sm font-mono;
}
</style>
