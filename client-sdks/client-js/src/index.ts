export interface ChatRequest {
  template_id: string;
  input: string;
  routing_profile?: string;
  model_config?: {
    provider?: string;
    model?: string;
    api_key?: string;
  };
  sandbox?: {
    kind?: string;
    work_dir?: string;
  };
  middlewares?: string[];
  metadata?: Record<string, unknown>;
}

export interface ChatResponse {
  agent_id?: string;
  text?: string;
  status: string;
  error_message?: string | null;
}

export interface ClientOptions {
  baseUrl: string;
  fetchImpl?: typeof fetch;
}

export class AgentsdkClient {
  private baseUrl: string;
  private fetchImpl: typeof fetch;

  constructor(options: ClientOptions) {
    this.baseUrl = options.baseUrl.replace(/\/+$/, "");
    this.fetchImpl = options.fetchImpl ?? fetch;
  }

  /**
   * Perform a synchronous chat call.
   */
  async chat(request: ChatRequest, signal?: AbortSignal): Promise<ChatResponse> {
    const resp = await this.fetchImpl(`${this.baseUrl}/v1/agents/chat`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json"
      },
      body: JSON.stringify(request),
      signal
    });

    if (!resp.ok) {
      throw new Error(`HTTP error ${resp.status}`);
    }

    const data = (await resp.json()) as ChatResponse;
    return data;
  }
}
