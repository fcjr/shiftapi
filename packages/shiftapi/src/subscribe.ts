/**
 * SSE stream returned by {@link createSubscribe}.
 * An async iterable of parsed events with a `close()` method to abort.
 */
export interface SSEStream<T> {
  [Symbol.asyncIterator](): AsyncIterableIterator<T>;
  close(): void;
}

/** Options accepted by a `subscribe` function created via {@link createSubscribe}. */
export interface SubscribeOptions {
  params?: {
    path?: Record<string, unknown>;
    query?: Record<string, unknown>;
    header?: Record<string, unknown>;
  };
  signal?: AbortSignal;
}

/** A subscribe function returned by {@link createSubscribe}. */
export type SubscribeFn = (
  path: string,
  options?: SubscribeOptions,
) => SSEStream<unknown>;

/**
 * Creates a type-safe SSE `subscribe` function bound to the given base URL.
 *
 * The returned function connects to an SSE endpoint via `fetch`, parses the
 * event stream, and yields parsed JSON events as an async iterable.
 *
 * Named events (with an `event:` field) are yielded as `{ event, data }`.
 * Unnamed events are yielded as the parsed `data` value directly.
 */
export function createSubscribe(baseUrl: string) {
  return function subscribe<T>(
    path: string,
    options: SubscribeOptions = {},
  ): SSEStream<T> {
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

    const headers: Record<string, string> = { Accept: "text/event-stream" };
    if (params?.header) {
      for (const [k, v] of Object.entries(params.header))
        headers[k] = String(v);
    }

    const controller = new AbortController();
    const signal = options.signal
      ? AbortSignal.any([options.signal, controller.signal])
      : controller.signal;

    const response = fetch(url, { method: "GET", signal, headers });

    return {
      async *[Symbol.asyncIterator]() {
        const res = await response;
        if (!res.ok) throw new Error("SSE request failed: " + res.status);
        const reader = res.body!
          .pipeThrough(new TextDecoderStream())
          .getReader();
        let buf = "";
        try {
          while (true) {
            const { done, value } = await reader.read();
            if (done) break;
            buf += value;
            const parts = buf.split("\n\n");
            buf = parts.pop()!;
            for (const part of parts) {
              const lines = part.split("\n");
              const eventLine = lines.find((l) => l.startsWith("event: "));
              const event = eventLine ? eventLine.slice(7) : undefined;
              const data = lines
                .filter((l) => l.startsWith("data: "))
                .map((l) => l.slice(6))
                .join("\n");
              if (data) {
                yield (
                  event !== undefined
                    ? { event, data: JSON.parse(data) }
                    : JSON.parse(data)
                ) as T;
              }
            }
          }
        } finally {
          reader.releaseLock();
        }
      },
      close() {
        controller.abort();
      },
    };
  };
}
