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
        确保已运行 <code>agentsdk serve</code>, 并暴露 <code>/v1/workflows/demo/run</code> 和 <code>/v1/workflows/demo/run-eval</code>。
      </p>
    </div>

    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
      <div class="space-y-3">
        <div class="space-y-2">
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
            Workflow ID
          </label>
          <select
            v-model="workflowId"
            class="w-full px-3 py-2 rounded border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-sm"
          >
            <option value="sequential_demo">sequential_demo - 顺序流水线</option>
            <option value="parallel_demo">parallel_demo - 并行多方案</option>
            <option value="loop_demo">loop_demo - 循环优化</option>
            <option value="nested_demo">nested_demo - 嵌套工作流</option>
          </select>
        </div>

        <div class="space-y-2">
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
            Input
          </label>
          <textarea
            v-model="inputText"
            rows="3"
            class="w-full px-3 py-2 rounded border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-sm"
            placeholder="例如: 处理用户数据 / 求解优化问题 / 优化代码质量"
          ></textarea>
        </div>

        <div class="space-y-2">
          <label class="inline-flex items-center text-xs text-gray-700 dark:text-gray-300">
            <input
              v-model="withEval"
              type="checkbox"
              class="mr-2 rounded border-gray-300 dark:border-gray-600"
            />
            同时对最终输出做 Evals (调用 /run-eval)
          </label>
          <div v-if="withEval" class="space-y-1">
            <label class="block text-xs font-medium text-gray-700 dark:text-gray-300">
              Reference (可选, 用于 lexical_similarity)
            </label>
            <textarea
              v-model="evalReference"
              rows="2"
              class="w-full px-2 py-1 rounded border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-xs"
              placeholder="期望的总结/回答(可留空)"
            ></textarea>

            <label class="block text-xs font-medium text-gray-700 dark:text-gray-300">
              Keywords (可选, 逗号分隔, 用于 keyword_coverage)
            </label>
            <input
              v-model="evalKeywords"
              class="w-full px-2 py-1 rounded border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-xs"
              placeholder="例如: 收集,分析,报告"
            />
          </div>
        </div>

        <button
          :disabled="loading"
          class="px-4 py-2 rounded bg-indigo-600 text-white text-sm disabled:opacity-50 disabled:cursor-not-allowed"
          @click="runWorkflow"
        >
          运行工作流{{ withEval ? '并评估输出' : '' }}
        </button>
        <span v-if="loading" class="text-xs text-gray-500 dark:text-gray-400">
          正在运行工作流...
        </span>
      </div>

      <div class="space-y-2">
        <h3 class="text-sm font-semibold mb-1">事件列表 (摘要)</h3>
        <div class="h-48 overflow-auto text-xs bg-gray-50 dark:bg-gray-900 border border-gray-200 dark:border-gray-700 rounded p-2">
          <template v-if="events.length">
            <div
              v-for="(ev, idx) in events"
              :key="ev.id || idx"
              class="mb-2 pb-2 border-b border-dashed border-gray-200 dark:border-gray-700 last:border-b-0 last:pb-0"
            >
              <div class="font-mono text-[10px] text-gray-500 dark:text-gray-400">
                {{ ev.timestamp }} · {{ ev.agent_id }} · {{ ev.branch || 'root' }}
              </div>
              <div class="mt-1">
                {{ ev.text }}
              </div>
              <div v-if="ev.metadata && Object.keys(ev.metadata).length" class="mt-1 text-[10px] text-gray-500 dark:text-gray-400">
                metadata: {{ formatMetadata(ev.metadata) }}
              </div>
            </div>
          </template>
          <template v-else>
            <div class="text-gray-400 dark:text-gray-500">
              (暂无事件, 请先运行一次工作流)
            </div>
          </template>
        </div>
        <div v-if="evalScores.length" class="space-y-1">
          <h3 class="text-sm font-semibold mt-2">Eval Scores</h3>
          <ul class="text-xs text-gray-700 dark:text-gray-300 list-disc list-inside">
            <li v-for="(score, idx) in evalScores" :key="score.name || idx">
              <span class="font-mono">{{ score.name }}:</span>
              <span class="ml-1">{{ score.value.toFixed(4) }}</span>
            </li>
          </ul>
        </div>
      </div>
    </div>

    <div class="space-y-1">
      <h3 class="text-sm font-semibold">原始 JSON 响应</h3>
      <pre class="h-40 overflow-auto text-xs bg-gray-50 dark:bg-gray-900 border border-gray-200 dark:border-gray-700 rounded p-2 whitespace-pre-wrap">
{{ rawResponse }}</pre>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'

const baseUrl = ref('http://localhost:8080')
const workflowId = ref<'sequential_demo' | 'parallel_demo' | 'loop_demo' | 'nested_demo'>('sequential_demo')
const inputText = ref('处理用户数据')
const loading = ref(false)
const rawResponse = ref<string>('(暂无)')
const withEval = ref(true)
const evalReference = ref('')
const evalKeywords = ref('')

interface WorkflowRunRequest {
  workflow_id: string
  input: string
}

interface WorkflowEvent {
  id?: string
  timestamp?: string
  agent_id?: string
  branch?: string
  author?: string
  text?: string
  metadata?: Record<string, unknown>
}

interface WorkflowRunResponse {
  events?: WorkflowEvent[]
  error_message?: string
}

interface ScoreResult {
  name: string
  value: number
  details?: Record<string, unknown>
}

interface WorkflowRunEvalResponse extends WorkflowRunResponse {
  eval_scores?: ScoreResult[]
}

const events = ref<WorkflowEvent[]>([])
const evalScores = ref<ScoreResult[]>([])

async function runWorkflow() {
  if (!inputText.value.trim()) {
    rawResponse.value = '请输入 Input 文本'
    return
  }

  loading.value = true
  rawResponse.value = '(运行中...)'
  events.value = []
  evalScores.value = []

  try {
    const body: any = {
      workflow_id: workflowId.value,
      input: inputText.value,
    }

    let path = '/v1/workflows/demo/run'
    if (withEval.value) {
      path = '/v1/workflows/demo/run-eval'
      body.reference = evalReference.value || undefined
      const kws = evalKeywords.value
        .split(',')
        .map((k) => k.trim())
        .filter((k) => k.length > 0)
      if (kws.length) {
        body.keywords = kws
      }
      body.scorers = ['keyword_coverage', 'lexical_similarity']
    }

    const resp = await fetch(normalizeUrl(path), {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(body),
    })

    if (!resp.ok) {
      rawResponse.value = `HTTP ${resp.status}`
      return
    }

    const data = (await resp.json()) as WorkflowRunEvalResponse
    rawResponse.value = JSON.stringify(data, null, 2)
    events.value = Array.isArray(data.events) ? data.events : []
    if (withEval.value && Array.isArray(data.eval_scores)) {
      evalScores.value = data.eval_scores
    }
  } catch (err: any) {
    rawResponse.value = `Error: ${err?.message ?? String(err)}`
  } finally {
    loading.value = false
  }
}

function normalizeUrl(path: string): string {
  const base = baseUrl.value.replace(/\/+$/, '')
  return base + path
}

function formatMetadata(meta: Record<string, unknown>): string {
  const keys = Object.keys(meta)
  if (!keys.length) return '{}'
  const max = 4
  const shown = keys.slice(0, max).map((k) => `${k}=${String(meta[k])}`)
  if (keys.length > max) {
    shown.push(`…+${keys.length - max} keys`)
  }
  return shown.join(', ')
}
</script>

<style scoped>
code {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono",
    "Courier New", monospace;
  font-size: 0.75rem;
}
</style>
