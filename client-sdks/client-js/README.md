# @agentsdk/client-js

Minimal JavaScript/TypeScript client for the AgentSDK HTTP API.

> This package is a lightweight helper built on top of the `/v1/agents/chat` endpoint implemented in `pkg/server`. It is intentionally small and framework-agnostic.

## Install

```bash
npm install @agentsdk/client-js
```

## Usage

```ts
import { AgentsdkClient } from '@agentsdk/client-js';

const client = new AgentsdkClient({
  baseUrl: 'http://localhost:8080',
});

const res = await client.chat({
  template_id: 'assistant',
  input: '请帮我总结一下 README',
  metadata: { user_id: 'alice' },
  middlewares: ['filesystem', 'agent_memory'],
});

console.log(res.status, res.text);
```

This client is intentionally minimal. For streaming responses, use the `/v1/agents/chat/stream` endpoint with `fetch` and a streaming reader in your application code.

