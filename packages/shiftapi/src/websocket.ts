/**
 * A typed WebSocket connection returned by {@link createWebSocket}.
 * Provides type-safe send/receive and an async iterable interface.
 * Automatically reconnects on unexpected disconnections by default.
 */
export interface WSConnection<Send, Recv> {
  /** Send a JSON-encoded message to the server. Throws if not connected. */
  send(data: Send): Promise<void>;
  /** Receive the next JSON message from the server. Survives reconnections. */
  receive(): Promise<Recv>;
  /** Async iterable that yields parsed JSON messages until close. Survives reconnections. */
  [Symbol.asyncIterator](): AsyncIterableIterator<Recv>;
  /** Close the WebSocket connection. Disables reconnection. */
  close(code?: number, reason?: string): void;
  /** The current readyState of the underlying WebSocket. */
  readonly readyState: number;
  /** Called when the connection opens (including after reconnection). */
  onopen: (() => void) | null;
  /** Called when the connection closes unexpectedly. Not called after explicit `close()`. */
  onclose: (() => void) | null;
}

/** Options for controlling automatic reconnection behavior. */
export interface ReconnectOptions {
  /** Maximum number of reconnect attempts before giving up. Default: Infinity. */
  maxRetries?: number;
  /** Initial delay in ms before the first reconnect attempt. Default: 1000. */
  baseDelay?: number;
  /** Maximum delay in ms between reconnect attempts (caps exponential backoff). Default: 30000. */
  maxDelay?: number;
}

/** Options accepted by a `websocket` function created via {@link createWebSocket}. */
export interface WebSocketOptions {
  params?: {
    path?: Record<string, unknown>;
    query?: Record<string, unknown>;
    header?: Record<string, unknown>;
  };
  protocols?: string[];
  /**
   * Controls automatic reconnection on unexpected disconnections.
   * - `true` or `undefined` (default): reconnect with default backoff.
   * - `false`: disable reconnection.
   * - `ReconnectOptions`: reconnect with custom settings.
   */
  reconnect?: boolean | ReconnectOptions;
}

/** A websocket function returned by {@link createWebSocket}. */
export type WebSocketFn = (
  path: string,
  options?: WebSocketOptions,
) => WSConnection<unknown, unknown>;

/**
 * Error thrown when the server rejects a WebSocket connection due to input
 * validation, setup errors, or other registered error types. The server
 * accepts the connection, sends the error as the first frame, then closes
 * with an application-defined status code (e.g. 4401 for unauthorized,
 * 4422 for validation errors).
 *
 * The type parameters narrow `code` and `details` when error types are
 * registered via `WithError` on the Go side — the generated `WSErrorFor<P>`
 * type maps each close code to its typed payload.
 */
export class WSError<
  Code extends number = number,
  Details = unknown,
> extends Error {
  /** The WebSocket close code (e.g. 4401, 4422). */
  readonly code: Code;
  /** The structured error payload sent by the server. */
  readonly details: Details;

  constructor(code: Code, details: Details) {
    const msg =
      typeof details === "object" &&
      details !== null &&
      "message" in details
        ? (details as { message: string }).message
        : "WebSocket connection error";
    super(msg);
    this.name = "WSError";
    this.code = code;
    this.details = details;
  }
}

const DEFAULT_RECONNECT: Required<ReconnectOptions> = {
  maxRetries: Infinity,
  baseDelay: 1000,
  maxDelay: 30000,
};

function resolveReconnect(
  opt: boolean | ReconnectOptions | undefined,
): Required<ReconnectOptions> | null {
  if (opt === false) return null;
  if (opt === true || opt === undefined) return DEFAULT_RECONNECT;
  return { ...DEFAULT_RECONNECT, ...opt };
}

/**
 * Creates a type-safe `websocket` function bound to the given base URL.
 *
 * The returned function opens a WebSocket connection, applying path and query
 * parameter substitution, and wraps it in a typed {@link WSConnection}.
 * Connections automatically reconnect on unexpected disconnections by default.
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

    const reconnectOpts = resolveReconnect(options.reconnect);

    // --- Shared state across reconnections ---

    // Queue of pending receive() callers waiting for a message.
    const recvQueue: {
      resolve: (value: Recv) => void;
      reject: (reason: unknown) => void;
    }[] = [];
    // Buffer of messages received before anyone called receive().
    const recvBuffer: Recv[] = [];

    let ws: WebSocket;
    let closed = false; // user called close()
    let fatalError: WSError | undefined; // server-sent error frame
    let retryCount = 0;
    let retryTimer: ReturnType<typeof setTimeout> | undefined;

    // Resolves when the initial connection is open and ready.
    let firstOpen: Promise<void>;
    let resolveFirstOpen: () => void;
    let rejectFirstOpen: (err: unknown) => void;

    function initFirstOpen() {
      firstOpen = new Promise<void>((resolve, reject) => {
        resolveFirstOpen = resolve;
        rejectFirstOpen = reject;
      });
    }
    initFirstOpen();

    function terminate(err: unknown) {
      closed = true;
      rejectFirstOpen(err);
      for (const pending of recvQueue) pending.reject(err);
      recvQueue.length = 0;
    }

    function scheduleReconnect() {
      if (!reconnectOpts || retryCount >= reconnectOpts.maxRetries) {
        terminate(new Error("WebSocket connection lost"));
        return;
      }
      const delay = Math.min(
        reconnectOpts.baseDelay * 2 ** retryCount,
        reconnectOpts.maxDelay,
      );
      retryCount++;
      retryTimer = setTimeout(() => {
        if (closed) return;
        connect();
      }, delay);
    }

    function connect() {
      ws = new WebSocket(url, options.protocols);

      ws.addEventListener("open", () => {
        retryCount = 0;
        resolveFirstOpen();
        conn.onopen?.();
      });

      ws.addEventListener("message", (event) => {
        const parsed = JSON.parse(event.data as string);

        // Error frames have {"error": true, "code": 4xxx, "data": ...}.
        // Data frames have {"type": "...", "data": ...}.
        if (
          parsed &&
          parsed.error === true &&
          typeof parsed.code === "number"
        ) {
          fatalError = new WSError(parsed.code, parsed.data);
          terminate(fatalError);
          return;
        }

        const data = parsed as Recv;
        const pending = recvQueue.shift();
        if (pending) {
          pending.resolve(data);
        } else {
          recvBuffer.push(data);
        }
      });

      ws.addEventListener("close", () => {
        if (closed || fatalError) return;
        conn.onclose?.();
        scheduleReconnect();
      });
    }

    const conn: WSConnection<Send, Recv> = {
      onopen: null,
      onclose: null,
      async send(data: Send): Promise<void> {
        await firstOpen;
        if (ws.readyState !== WebSocket.OPEN) {
          throw new Error("WebSocket is not open");
        }
        ws.send(JSON.stringify(data));
      },
      receive(): Promise<Recv> {
        const buffered = recvBuffer.shift();
        if (buffered !== undefined) {
          return Promise.resolve(buffered);
        }
        if (fatalError) {
          return Promise.reject(fatalError);
        }
        if (closed) {
          return Promise.reject(new Error("WebSocket is closed"));
        }
        return new Promise<Recv>((resolve, reject) => {
          recvQueue.push({ resolve, reject });
        });
      },
      async *[Symbol.asyncIterator](): AsyncIterableIterator<Recv> {
        try {
          while (true) {
            yield await this.receive();
          }
        } catch (err) {
          if (err instanceof WSError) throw err;
          // Normal close — stop iteration.
        }
      },
      close(code?: number, reason?: string): void {
        closed = true;
        if (retryTimer !== undefined) clearTimeout(retryTimer);
        ws.close(code, reason);
        const err = new Error("WebSocket is closed");
        rejectFirstOpen(err);
        for (const pending of recvQueue) pending.reject(err);
        recvQueue.length = 0;
      },
      get readyState(): number {
        return ws.readyState;
      },
    };

    // Establish the initial connection.
    connect();

    return conn;
  };
}
