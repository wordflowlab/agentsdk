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
        确保已运行 <code>agentsdk serve</code> 或 <code>examples/server-http</code> 示例。
      </p>
    </div>

    <div class="space-y-2">
      <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
        Template ID
      </label>
      <input
        v-model="templateId"
        class="w-full px-3 py-2 rounded border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-sm"
        placeholder="assistant"
      />
    </div>

    <div class="space-y-2">
      <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
        Routing Profile (可选)
      </label>
      <select
        v-model="routingProfile"
        class="w-full px-3 py-2 rounded border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-sm"
      >
        <option value="">默认(使用模板/显式 model_config)</option>
        <option value="quality">quality - 高质量优先</option>
        <option value="cost">cost - 成本优先</option>
        <option value="latency">latency - 延迟优先</option>
      </select>
      <p class="text-xs text-gray-500 dark:text-gray-400">
        当服务端配置了 Router 时, 可通过此字段在多模型之间路由。
      </p>
    </div>

    <div class="space-y-2">
      <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
        User Input
      </label>
      <textarea
        v-model="inputText"
        rows="4"
        class="w-full px-3 py-2 rounded border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-sm"
        placeholder="例如: 请帮我总结一下 README"
      ></textarea>
    </div>

    <div class="flex items-center gap-3">
      <button
        :disabled="loading"
        class="px-4 py-2 rounded bg-blue-600 text-white text-sm disabled:opacity-50 disabled:cursor-not-allowed"
        @click="sendSync"
      >
        同步调用 /v1/agents/chat
      </button>
      <button
        :disabled="loading"
        class="px-4 py-2 rounded bg-emerald-600 text-white text-sm disabled:opacity-50 disabled:cursor-not-allowed"
        @click="sendStream"
      >
        流式调用 /v1/agents/chat/stream
      </button>
      <span v-if="loading" class="text-xs text-gray-500 dark:text-gray-400">
        正在请求...
      </span>
    </div>

    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
      <div>
        <h3 class="text-sm font-semibold mb-1">同步响应</h3>
        <pre class="h-64 overflow-auto text-xs bg-gray-50 dark:bg-gray-900 border border-gray-200 dark:border-gray-700 rounded p-2 whitespace-pre-wrap">
{{ syncResponse }}</pre>
      </div>
      <div>
        <h3 class="text-sm font-semibold mb-1">流式事件 (SSE)</h3>
        <pre class="h-64 overflow-auto text-xs bg-gray-50 dark:bg-gray-900 border border-gray-200 dark:border-gray-700 rounded p-2 whitespace-pre-wrap">
{{ streamEvents }}</pre>
      </div>
    </div>

    <div class="space-y-3 border-t border-dashed border-gray-200 dark:border-gray-700 pt-4">
      <h3 class="text-sm font-semibold">Evals (批量评估)</h3>
      <p class="text-xs text-gray-500 dark:text-gray-400">
        使用最近一次同步响应作为 answer，选择多个 Scorer 进行批量评估。
      </p>

      <div class="grid grid-cols-1 lg:grid-cols-3 gap-4">
        <!-- 左侧：输入配置 -->
        <div class="space-y-3">
          <div>
            <label class="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">
              Reference (可选)
            </label>
            <textarea
              v-model="evalReference"
              rows="2"
              class="w-full px-2 py-1 rounded border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-xs"
              placeholder="参考答案"
            ></textarea>
          </div>

          <div>
            <label class="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">
              Keywords (逗号分隔)
            </label>
            <input
              v-model="evalKeywords"
              class="w-full px-2 py-1 rounded border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-xs"
              placeholder="例如: 首都,法国"
            />
          </div>

          <div>
            <label class="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">
              Provider API Key (LLM Scorers)
            </label>
            <input
              v-model="providerApiKey"
              type="password"
              class="w-full px-2 py-1 rounded border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-xs"
              placeholder="sk-ant-..."
            />
            <p class="text-xs text-gray-400 mt-0.5">仅在选择 LLM-based scorers 时需要</p>
          </div>
        </div>

        <!-- 中间：Scorer 选择 -->
        <div class="space-y-2">
          <label class="block text-xs font-medium text-gray-700 dark:text-gray-300">
            选择 Scorers
          </label>
          <div class="space-y-1.5 max-h-64 overflow-y-auto border border-gray-200 dark:border-gray-700 rounded p-2 bg-gray-50 dark:bg-gray-900">
            <div class="text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">启发式 Scorers</div>
            <label v-for="scorer in heuristicScorers" :key="scorer.id" class="flex items-start gap-2 cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-800 p-1 rounded">
              <input
                type="checkbox"
                :value="scorer.id"
                v-model="selectedScorers"
                class="mt-0.5"
              />
              <div class="flex-1">
                <div class="text-xs font-medium">{{ scorer.name }}</div>
                <div class="text-xs text-gray-500 dark:text-gray-400">{{ scorer.desc }}</div>
              </div>
            </label>

            <div class="text-xs font-medium text-gray-500 dark:text-gray-400 mt-2 mb-1 pt-2 border-t border-gray-300 dark:border-gray-600">LLM-based Scorers</div>
            <label v-for="scorer in llmScorers" :key="scorer.id" class="flex items-start gap-2 cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-800 p-1 rounded">
              <input
                type="checkbox"
                :value="scorer.id"
                v-model="selectedScorers"
                class="mt-0.5"
              />
              <div class="flex-1">
                <div class="text-xs font-medium">{{ scorer.name }}</div>
                <div class="text-xs text-gray-500 dark:text-gray-400">{{ scorer.desc }}</div>
              </div>
            </label>
          </div>

          <button
            :disabled="evalLoading || selectedScorers.length === 0"
            class="w-full mt-2 px-3 py-1.5 rounded bg-purple-600 text-white text-xs disabled:opacity-50 disabled:cursor-not-allowed"
            @click="runBatchEval"
          >
            运行批量评估 ({{ selectedScorers.length }} scorers)
          </button>
          <span v-if="evalLoading" class="block text-xs text-center text-gray-500 dark:text-gray-400 mt-1">
            正在评估...
          </span>
        </div>

        <!-- 右侧：评估结果 -->
        <div class="space-y-2">
          <h4 class="text-xs font-semibold">评估结果</h4>
          <div v-if="evalResults.length > 0" class="space-y-2 max-h-64 overflow-y-auto">
            <div v-for="(result, idx) in evalResults" :key="idx"
                 class="p-2 rounded border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800">
              <div class="flex items-center justify-between mb-1">
                <span class="text-xs font-medium">{{ result.name }}</span>
                <span class="text-xs font-bold" :class="getScoreColor(result.value)">
                  {{ (result.value * 100).toFixed(0) }}%
                </span>
              </div>
              <div v-if="result.details?.reason" class="text-xs text-gray-600 dark:text-gray-400 mt-1">
                {{ result.details.reason }}
              </div>
              <div v-if="result.details?.matched" class="text-xs text-green-600 dark:text-green-400 mt-1">
                匹配: {{ result.details.matched.join(', ') }}
              </div>
              <div v-if="result.details?.unmatched && result.details.unmatched.length > 0" class="text-xs text-red-600 dark:text-red-400">
                未匹配: {{ result.details.unmatched.join(', ') }}
              </div>
            </div>
          </div>
          <div v-else class="text-xs text-gray-500 dark:text-gray-400 italic p-2">
            暂无评估结果
          </div>

          <div v-if="evalSummary" class="mt-2 p-2 rounded bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800">
            <div class="text-xs font-semibold mb-1">汇总</div>
            <div class="text-xs space-y-0.5">
              <div>用例数: {{ evalSummary.total_cases }}</div>
              <div>成功: {{ evalSummary.successful_cases }} / 失败: {{ evalSummary.failed_cases }}</div>
              <div>耗时: {{ evalSummary.duration_ms }}ms</div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <p class="text-xs text-gray-500 dark:text-gray-400">
      提示: 此 Playground 为演示用途, 未实现完整会话管理/多轮上下文, 仅用来验证 API 和流式事件。
    </p>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'

const baseUrl = ref('http://localhost:8080')
const templateId = ref('assistant')
const inputText = ref('')
const routingProfile = ref<string>('') // '', 'cost', 'quality', 'latency'

const syncResponse = ref<string>('(暂无)')
const streamEvents = ref<string>('(暂无)')
const loading = ref(false)
const evalLoading = ref(false)
const evalReference = ref<string>('')
const evalKeywords = ref<string>('')
const providerApiKey = ref<string>('')
const selectedScorers = ref<string[]>(['keyword_coverage', 'lexical_similarity'])
const evalResults = ref<any[]>([])
const evalSummary = ref<any>(null)

// Scorer 列表
const heuristicScorers = [
  { id: 'keyword_coverage', name: '关键词覆盖率', desc: '检查关键词是否出现在答案中' },
  { id: 'lexical_similarity', name: '词汇相似度', desc: '基于 Jaccard 相似度比较' },
]

const llmScorers = [
  { id: 'faithfulness', name: '忠实度', desc: '答案是否忠实于上下文' },
  { id: 'hallucination', name: '幻觉检测', desc: '检测虚假或无法验证的信息' },
  { id: 'answer_relevancy', name: '答案相关性', desc: '答案是否直接回答问题' },
  { id: 'context_relevancy', name: '上下文相关性', desc: '上下文是否对回答有帮助' },
  { id: 'toxicity', name: '毒性检测', desc: '检测有害或不当内容' },
  { id: 'tone_consistency', name: '语气一致性', desc: '文本语气是否统一' },
  { id: 'coherence', name: '连贯性', desc: '逻辑结构和流畅度' },
  { id: 'completeness', name: '完整性', desc: '答案是否全面' },
]

interface ChatRequest {
  template_id: string
  input: string
  routing_profile?: string
  middlewares?: string[]
  metadata?: Record<string, unknown>
}

interface BatchEvalRequest {
  test_cases: {
    id: string
    answer: string
    context?: string[]
    reference?: string
  }[]
  scorers: string[]
  keywords?: string[]
  provider_config?: {
    provider: string
    model: string
    api_key: string
  }
  concurrency?: number
}

async function sendSync() {
  if (!inputText.value.trim()) {
    syncResponse.value = '请输入问题文本'
    return
  }

  loading.value = true
  streamEvents.value = '(暂无)'

  try {
    const body: ChatRequest = {
      template_id: templateId.value || 'assistant',
      input: inputText.value,
      // 示例中默认启用文件系统 + memory 中间件, 若服务端未配置会自动忽略未知中间件
      middlewares: ['filesystem', 'agent_memory'],
      metadata: {
        user_id: 'playground-user',
      },
    }

    if (routingProfile.value) {
      body.routing_profile = routingProfile.value
    }

    const resp = await fetch(normalizeUrl('/v1/agents/chat'), {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(body),
    })

    if (!resp.ok) {
      syncResponse.value = `HTTP ${resp.status}`
      return
    }

    const data = await resp.json()
    syncResponse.value = JSON.stringify(data, null, 2)
  } catch (err: any) {
    syncResponse.value = `Error: ${err?.message ?? String(err)}`
  } finally {
    loading.value = false
  }
}

async function sendStream() {
  if (!inputText.value.trim()) {
    streamEvents.value = '请输入问题文本'
    return
  }

  loading.value = true
  syncResponse.value = '(暂无)'
  streamEvents.value = ''

  try {
    const body: ChatRequest = {
      template_id: templateId.value || 'assistant',
      input: inputText.value,
      middlewares: ['filesystem', 'agent_memory'],
      metadata: {
        user_id: 'playground-user',
      },
    }

    if (routingProfile.value) {
      body.routing_profile = routingProfile.value
    }

    const resp = await fetch(normalizeUrl('/v1/agents/chat/stream'), {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(body),
    })

    if (!resp.ok || !resp.body) {
      streamEvents.value = `HTTP ${resp.status}`
      return
    }

    const reader = resp.body.getReader()
    const decoder = new TextDecoder('utf-8')

    while (true) {
      const { value, done } = await reader.read()
      if (done) break

      const chunk = decoder.decode(value, { stream: true })
      // 按行处理 SSE: data: {...}
      const lines = chunk.split('\n')
      for (const line of lines) {
        const trimmed = line.trim()
        if (!trimmed) continue
        if (trimmed.startsWith('data:')) {
          const jsonStr = trimmed.substring('data:'.length).trim()
          try {
            const obj = JSON.parse(jsonStr)
            streamEvents.value += JSON.stringify(obj) + '\n'
          } catch {
            streamEvents.value += trimmed + '\n'
          }
        } else {
          streamEvents.value += trimmed + '\n'
        }
      }
    }
  } catch (err: any) {
    streamEvents.value += `Error: ${err?.message ?? String(err)}\n`
  } finally {
    loading.value = false
  }
}

async function runBatchEval() {
  if (!syncResponse.value || syncResponse.value === '(暂无)') {
    evalResults.value = []
    evalSummary.value = { error: '请先完成一次同步调用' }
    return
  }

  // 解析 answer
  let answerText = ''
  try {
    const parsed = JSON.parse(syncResponse.value)
    if (parsed && typeof parsed.text === 'string' && parsed.text.trim()) {
      answerText = parsed.text
    }
  } catch {
    // syncResponse 不是 JSON
  }

  if (!answerText) {
    evalResults.value = []
    evalSummary.value = { error: '无法从响应中解析出 text 字段' }
    return
  }

  // 准备 keywords
  const keywords = evalKeywords.value
    .split(',')
    .map((k) => k.trim())
    .filter((k) => k.length > 0)

  // 检查是否选择了 LLM scorers
  const hasLLMScorer = selectedScorers.value.some(s =>
    !['keyword_coverage', 'lexical_similarity'].includes(s)
  )

  // 构建请求
  const body: BatchEvalRequest = {
    test_cases: [
      {
        id: 'case1',
        answer: answerText,
        reference: evalReference.value.trim() || undefined,
      },
    ],
    scorers: selectedScorers.value,
    keywords: keywords.length > 0 ? keywords : undefined,
    concurrency: 5,
  }

  // 如果有 LLM scorer，添加 provider_config
  if (hasLLMScorer) {
    if (!providerApiKey.value.trim()) {
      evalResults.value = []
      evalSummary.value = { error: 'LLM-based scorers 需要 Provider API Key' }
      return
    }
    body.provider_config = {
      provider: 'anthropic',
      model: 'claude-sonnet-4-5',
      api_key: providerApiKey.value.trim(),
    }
  }

  evalLoading.value = true
  evalResults.value = []
  evalSummary.value = null

  try {
    const resp = await fetch(normalizeUrl('/v1/evals/batch'), {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(body),
    })

    if (!resp.ok) {
      const error = await resp.text()
      evalSummary.value = { error: `HTTP ${resp.status}: ${error}` }
      return
    }

    const data = await resp.json()

    // 提取第一个测试用例的结果
    if (data.results && data.results.length > 0) {
      evalResults.value = data.results[0].scores || []
    }

    // 设置汇总信息
    if (data.summary) {
      evalSummary.value = {
        total_cases: data.summary.total_cases,
        successful_cases: data.summary.successful_cases,
        failed_cases: data.summary.failed_cases,
        duration_ms: data.total_duration_ms,
      }
    }
  } catch (err: any) {
    evalSummary.value = { error: `Error: ${err?.message ?? String(err)}` }
  } finally {
    evalLoading.value = false
  }
}

function getScoreColor(score: number): string {
  if (score >= 0.8) return 'text-green-600 dark:text-green-400'
  if (score >= 0.6) return 'text-yellow-600 dark:text-yellow-400'
  if (score >= 0.4) return 'text-orange-600 dark:text-orange-400'
  return 'text-red-600 dark:text-red-400'
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
  font-size: 0.75rem;
}
</style>
