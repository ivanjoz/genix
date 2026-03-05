export interface SSEClientEvent<TPayload = unknown> {
  eventName: string;
  payload: TPayload;
  rawPayloadText: string;
  receivedAt: Date;
}

export interface SSEClientComment {
  commentText: string;
  receivedAt: Date;
}

export interface SSEClientConfig<TPayload = unknown> {
  endpointUrl: string;
  requestHeaders?: Record<string, string>;
  onOpen?: () => void;
  onMessage?: (event: SSEClientEvent<TPayload>) => void;
  onComment?: (comment: SSEClientComment) => void;
  onError?: (error: Error) => void;
  onClose?: () => void;
}

export class SSEClient<TPayload = unknown> {
  private readonly endpointUrl: string;
  private readonly requestHeaders: Record<string, string>;
  private readonly onOpen?: () => void;
  private readonly onMessage?: (event: SSEClientEvent<TPayload>) => void;
  private readonly onComment?: (comment: SSEClientComment) => void;
  private readonly onError?: (error: Error) => void;
  private readonly onClose?: () => void;

  private currentAbortController: AbortController | null = null;
  private runningRequest: Promise<void> | null = null;

  constructor(clientConfig: SSEClientConfig<TPayload>) {
    this.endpointUrl = clientConfig.endpointUrl;
    this.requestHeaders = clientConfig.requestHeaders || {};
    this.onOpen = clientConfig.onOpen;
    this.onMessage = clientConfig.onMessage;
    this.onComment = clientConfig.onComment;
    this.onError = clientConfig.onError;
    this.onClose = clientConfig.onClose;
  }

  isConnected() {
    return this.currentAbortController !== null;
  }

  connect() {
    if (this.runningRequest) return this.runningRequest;

    this.currentAbortController = new AbortController();
    this.runningRequest = this.consumeStream(this.currentAbortController)
      .catch((streamError: unknown) => {
        // Forward stream errors through a dedicated callback so callers can handle retries.
        if (!this.currentAbortController?.signal.aborted) {
          const normalizedError = streamError instanceof Error ? streamError : new Error(String(streamError || 'unknown sse error'));
          this.onError?.(normalizedError);
        }
      })
      .finally(() => {
        this.currentAbortController = null;
        this.runningRequest = null;
        this.onClose?.();
      });

    return this.runningRequest;
  }

  disconnect() {
    if (!this.currentAbortController) return;
    this.currentAbortController.abort();
  }

  private async consumeStream(streamAbortController: AbortController) {
    const streamResponse = await fetch(this.endpointUrl, {
      method: 'GET',
      headers: {
        Accept: 'text/event-stream',
        ...this.requestHeaders
      },
      signal: streamAbortController.signal
    });

    if (!streamResponse.ok) {
      const errorBody = await streamResponse.text();
      throw new Error(`HTTP ${streamResponse.status}: ${errorBody || 'sse stream request failed'}`);
    }

    if (!streamResponse.body) {
      throw new Error('SSE response body is empty.');
    }

    this.onOpen?.();

    const streamReader = streamResponse.body.getReader();
    const streamTextDecoder = new TextDecoder();
    let bufferedChunkText = '';

    while (true) {
      const nextChunk = await streamReader.read();
      if (nextChunk.done) {
        break;
      }

      bufferedChunkText += streamTextDecoder.decode(nextChunk.value, { stream: true });
      const pendingBlocks = bufferedChunkText.split('\n\n');
      bufferedChunkText = pendingBlocks.pop() || '';

      for (const blockText of pendingBlocks) {
        this.processBlock(blockText);
      }
    }
  }

  private processBlock(rawBlockText: string) {
    const blockLines = rawBlockText
      .split('\n')
      .map((lineText) => lineText.trimEnd())
      .filter((lineText) => lineText.length > 0);

    if (blockLines.length === 0) return;

    let parsedEventName = 'message';
    const dataLines: string[] = [];

    for (const lineText of blockLines) {
      if (lineText.startsWith(':')) {
        this.onComment?.({
          commentText: lineText.substring(1).trim(),
          receivedAt: new Date()
        });
        return;
      }

      if (lineText.startsWith('event:')) {
        parsedEventName = lineText.substring(6).trim();
        continue;
      }

      if (lineText.startsWith('data:')) {
        dataLines.push(lineText.substring(5).trim());
      }
    }

    if (dataLines.length === 0) return;

    const rawPayloadText = dataLines.join('\n');
    let parsedPayload: unknown = rawPayloadText;

    try {
      parsedPayload = JSON.parse(rawPayloadText);
    } catch {
      // Non-JSON payloads are still valid SSE data and should be forwarded as plain text.
    }

    this.onMessage?.({
      eventName: parsedEventName,
      payload: parsedPayload as TPayload,
      rawPayloadText,
      receivedAt: new Date()
    });
  }
}
