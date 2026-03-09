/**
 * A typed WebSocket connection returned by {@link createWebSocket}.
 * Provides type-safe send/receive and an async iterable interface.
 */
export interface WSConnection<Send, Recv> {
  /** Send a JSON-encoded message to the server. Waits for the connection to open. */
  send(data: Send): Promise<void>;
  /** Receive the next JSON message from the server. */
  receive(): Promise<Recv>;
  /** Async iterable that yields parsed JSON messages until close. */
  [Symbol.asyncIterator](): AsyncIterableIterator<Recv>;
  /** Close the WebSocket connection. */
  close(code?: number, reason?: string): void;
  /** The current readyState of the underlying WebSocket. */
  readonly readyState: number;
}

/** Options accepted by a `websocket` function created via {@link createWebSocket}. */
export interface WebSocketOptions {
  params?: {
    path?: Record<string, unknown>;
    query?: Record<string, unknown>;
    header?: Record<string, unknown>;
  };
  protocols?: string[];
}

/** A websocket function returned by {@link createWebSocket}. */
export type WebSocketFn = (
  path: string,
  options?: WebSocketOptions,
) => WSConnection<unknown, unknown>;

/**
 * Creates a type-safe `websocket` function bound to the given base URL.
 *
 * The returned function opens a WebSocket connection, applying path and query
 * parameter substitution, and wraps it in a typed {@link WSConnection}.
 */
export function createWebSocket(baseUrl: string) {
  return function websocket<Send, Recv>(
    path: string,
    options: WebSocketOptions = {},
  ): WSConnection<Send, Recv> {
    let url = baseUrl + path;
    const { params } = options;

    if (params?.path) {
      for (const [k, v] of Object.entries(params.path))
        url = url.replace("{" + k + "}", encodeURIComponent(String(v)));
    }

    if (params?.query) {
      const qs = new URLSearchParams();
      for (const [k, v] of Object.entries(params.query)) qs.set(k, String(v));
      const str = qs.toString();
      if (str) url += "?" + str;
    }

    // Replace http(s):// with ws(s)://
    url = url.replace(/^http/, "ws");

    const ws = new WebSocket(url, options.protocols);

    // Resolves when the connection is open and ready to send.
    const ready = new Promise<void>((resolve, reject) => {
      ws.addEventListener("open", () => resolve(), { once: true });
      ws.addEventListener("error", (e) => reject(e), { once: true });
    });

    // Queue of pending receive resolvers.
    const queue: {
      resolve: (value: Recv) => void;
      reject: (reason: unknown) => void;
    }[] = [];
    // Buffer of messages received before anyone called receive().
    const buffer: Recv[] = [];
    let closeError: Event | undefined;

    ws.addEventListener("message", (event) => {
      const data = JSON.parse(event.data as string) as Recv;
      const pending = queue.shift();
      if (pending) {
        pending.resolve(data);
      } else {
        buffer.push(data);
      }
    });

    ws.addEventListener("close", (event) => {
      closeError = event;
      // Reject all pending receivers.
      for (const pending of queue) {
        pending.reject(event);
      }
      queue.length = 0;
    });

    ws.addEventListener("error", (event) => {
      closeError = event;
      for (const pending of queue) {
        pending.reject(event);
      }
      queue.length = 0;
    });

    return {
      async send(data: Send): Promise<void> {
        await ready;
        ws.send(JSON.stringify(data));
      },
      receive(): Promise<Recv> {
        const buffered = buffer.shift();
        if (buffered !== undefined) {
          return Promise.resolve(buffered);
        }
        if (closeError) {
          return Promise.reject(closeError);
        }
        return new Promise<Recv>((resolve, reject) => {
          queue.push({ resolve, reject });
        });
      },
      async *[Symbol.asyncIterator](): AsyncIterableIterator<Recv> {
        try {
          while (true) {
            yield await this.receive();
          }
        } catch {
          // Connection closed — stop iteration.
        }
      },
      close(code?: number, reason?: string): void {
        ws.close(code, reason);
      },
      get readyState(): number {
        return ws.readyState;
      },
    };
  };
}
