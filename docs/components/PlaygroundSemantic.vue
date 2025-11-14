<template>
  <div class="border border-gray-200 dark:border-gray-700 rounded-lg p-4 space-y-4">
    <div class="space-y-2">
      <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
        Server Base URL
      </label>
      <input
        v-model="baseUrl"
        class="w-full px-3 py-2 rounded border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-sm"
        placeholder="http://localhost:8080"
      />
      <p class="text-xs text-gray-500 dark:text-gray-400">
        确保已运行 <code>agentsdk serve</code>, 并启用了 <code>/v1/memory/semantic/search</code>。
      </p>
    </div>

    <div class="space-y-2">
      <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
        Query
      </label>
      <textarea
        v-model="query"
        rows="3"
        class="w-full px-3 py-2 rounded border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-sm"
        placeholder="例如: What is the capital of France?"
      ></textarea>
    </div>

    <div class="space-y-2">
      <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
        Metadata (可选, JSON)
      </label>
      <textarea
        v-model="metadataText"
        rows="3"
        class="w-full px-3 py-2 rounded border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-xs"
        placeholder='例如: { "user_id": "alice", "project_id": "demo" }'
      ></textarea>
      <p class="text-xs text-gray-500 dark:text-gray-400">
        元数据用于构造命名空间(user/project/resource), 建议至少提供 <code>user_id</code>。
      </p>
    </div>

    <div class="flex items-center gap-3">
      <button
        :disabled="loading"
        class="px-4 py-2 rounded bg-indigo-600 text-white text-sm disabled:opacity-50 disabled:cursor-not-allowed"
        @click="runSemanticSearch"
      >
        调用 /v1/memory/semantic/search
      </button>
      <span v-if="loading" class="text-xs text-gray-500 dark:text-gray-400">
        正在检索...
      </span>
    </div>

    <div class="space-y-2">
      <h3 class="text-sm font-semibold">结果</h3>
      <pre class="h-52 overflow-auto text-xs bg-gray-50 dark:bg-gray-900 border border-gray-200 dark:border-gray-700 rounded p-2 whitespace-pre-wrap">
{{ responseText }}</pre>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'

const baseUrl = ref('http://localhost:8080')
const query = ref('')
const metadataText = ref<string>('{\n  "user_id": "alice",\n  "resource_id": "europe-notes"\n}')
const loading = ref(false)
const responseText = ref<string>('(暂无)')

interface SemanticSearchRequest {
  query: string
  top_k?: number
  metadata?: Record<string, unknown>
}

async function runSemanticSearch() {
  if (!query.value.trim()) {
    responseText.value = '请输入 Query 文本'
    return
  }

  let metadata: Record<string, unknown> | undefined
  if (metadataText.value.trim()) {
    try {
      const parsed = JSON.parse(metadataText.value)
      if (parsed && typeof parsed === 'object') {
        metadata = parsed as Record<string, unknown>
      }
    } catch (err: any) {
      responseText.value = `Metadata JSON 解析失败: ${err?.message ?? String(err)}`
      return
    }
  }

  const body: SemanticSearchRequest = {
    query: query.value.trim(),
    top_k: 5,
    metadata,
  }

  loading.value = true
  responseText.value = '(检索中...)'

  try {
    const resp = await fetch(normalizeUrl('/v1/memory/semantic/search'), {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(body),
    })

    if (!resp.ok) {
      responseText.value = `HTTP ${resp.status}`
      return
    }

    const data = await resp.json()
    responseText.value = JSON.stringify(data, null, 2)
  } catch (err: any) {
    responseText.value = `Error: ${err?.message ?? String(err)}`
  } finally {
    loading.value = false
  }
}

function normalizeUrl(path: string): string {
  const base = baseUrl.value.replace(/\/+$/, '')
  return base + path
}
</script>

<style scoped>
code {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono",
    "Courier New", monospace;
  font-size: 0.75rem; /* text-xs */
}
</style>
