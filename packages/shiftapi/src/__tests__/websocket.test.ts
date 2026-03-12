import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { createWebSocket, WSError } from "../websocket";

// --- Mock WebSocket ---

type WSListener = (event: Record<string, unknown>) => void;

// When true, new MockWebSocket instances will NOT auto-open.
let suppressAutoOpen = false;

class MockWebSocket {
  static CONNECTING = 0;
  static OPEN = 1;
  static CLOSING = 2;
  static CLOSED = 3;

  url: string;
  protocols?: string | string[];
  readyState = MockWebSocket.CONNECTING;

  private listeners: Record<string, WSListener[]> = {};
  private sent: string[] = [];

  constructor(url: string, protocols?: string | string[]) {
    this.url = url;
    this.protocols = protocols;
    if (!suppressAutoOpen) {
      // Auto-open on next microtask by default.
      queueMicrotask(() => {
        if (this.readyState === MockWebSocket.CONNECTING) {
          this.serverOpen();
        }
      });
    }
  }

  serverOpen() {
    this.readyState = MockWebSocket.OPEN;
    this.emit("open", {});
  }

  addEventListener(event: string, fn: WSListener) {
    (this.listeners[event] ??= []).push(fn);
  }

  send(data: string) {
    this.sent.push(data);
  }

  close(_code?: number, _reason?: string) {
    this.readyState = MockWebSocket.CLOSED;
  }

  // --- Test helpers ---

  emit(event: string, data: Record<string, unknown>) {
    for (const fn of this.listeners[event] ?? []) fn(data);
  }

  serverSend(data: unknown) {
    this.emit("message", { data: JSON.stringify(data) });
  }

  serverClose() {
    this.readyState = MockWebSocket.CLOSED;
    this.emit("close", {});
  }

  getSent(): unknown[] {
    return this.sent.map((s) => JSON.parse(s));
  }
}

// Track all created MockWebSocket instances so tests can access them.
let mockInstances: MockWebSocket[] = [];

function latestMock(): MockWebSocket {
  return mockInstances[mockInstances.length - 1];
}

beforeEach(() => {
  mockInstances = [];
  vi.stubGlobal(
    "WebSocket",
    Object.assign(
      function (this: MockWebSocket, url: string, protocols?: string | string[]) {
        const instance = new MockWebSocket(url, protocols);
        mockInstances.push(instance);
        return instance;
      } as unknown as typeof WebSocket,
      {
        CONNECTING: MockWebSocket.CONNECTING,
        OPEN: MockWebSocket.OPEN,
        CLOSING: MockWebSocket.CLOSING,
        CLOSED: MockWebSocket.CLOSED,
      },
    ),
  );
});

afterEach(() => {
  suppressAutoOpen = false;
  vi.unstubAllGlobals();
  vi.useRealTimers();
});

// Helper to create a connection and wait for it to open.
function connect(options?: Parameters<ReturnType<typeof createWebSocket>>[1]) {
  const websocket = createWebSocket("http://localhost:8080");
  const conn = websocket<{ text: string }, { text: string }>("/echo", options);
  return conn;
}

// --- Tests ---

describe("createWebSocket", () => {
  describe("URL building", () => {
    it("replaces http with ws", () => {
      connect();
      expect(latestMock().url).toBe("ws://localhost:8080/echo");
    });

    it("replaces https with wss", () => {
      const websocket = createWebSocket("https://example.com");
      websocket("/chat");
      expect(latestMock().url).toBe("wss://example.com/chat");
    });

    it("substitutes path parameters", () => {
      const websocket = createWebSocket("http://localhost:8080");
      websocket("/rooms/{room}", { params: { path: { room: "general" } } });
      expect(latestMock().url).toBe("ws://localhost:8080/rooms/general");
    });

    it("appends query parameters", () => {
      const websocket = createWebSocket("http://localhost:8080");
      websocket("/echo", { params: { query: { token: "abc" } } });
      expect(latestMock().url).toBe("ws://localhost:8080/echo?token=abc");
    });

    it("passes protocols", () => {
      const websocket = createWebSocket("http://localhost:8080");
      websocket("/echo", { protocols: ["graphql-ws"] });
      expect(latestMock().protocols).toEqual(["graphql-ws"]);
    });
  });

  describe("send", () => {
    it("waits for open then sends JSON", async () => {
      const conn = connect();
      await conn.send({ text: "hello" });
      expect(latestMock().getSent()).toEqual([{ text: "hello" }]);
    });

    it("throws if connection is not open", async () => {
      const conn = connect();
      // Wait for open, then close the underlying socket.
      await conn.send({ text: "first" });
      latestMock().readyState = MockWebSocket.CLOSED;
      await expect(conn.send({ text: "second" })).rejects.toThrow(
        "WebSocket is not open",
      );
    });
  });

  describe("receive", () => {
    it("resolves with the next message", async () => {
      const conn = connect();
      // Wait for connection to open.
      await conn.send({ text: "ping" });

      const promise = conn.receive();
      latestMock().serverSend({ type: "echo", data: { text: "pong" } });
      const msg = await promise;
      expect(msg).toEqual({ type: "echo", data: { text: "pong" } });
    });

    it("returns buffered messages immediately", async () => {
      const conn = connect();
      await conn.send({ text: "ping" });

      // Server sends before anyone calls receive.
      latestMock().serverSend({ text: "buffered" });
      const msg = await conn.receive();
      expect(msg).toEqual({ text: "buffered" });
    });

    it("rejects when connection is closed", async () => {
      const conn = connect({ reconnect: false });
      await conn.send({ text: "ping" });

      const promise = conn.receive();
      latestMock().serverClose();
      await expect(promise).rejects.toThrow();
    });
  });

  describe("async iterator", () => {
    it("yields messages until close", async () => {
      const conn = connect({ reconnect: false });
      await conn.send({ text: "ping" });

      const messages: unknown[] = [];
      const done = (async () => {
        for await (const msg of conn) {
          messages.push(msg);
        }
      })();

      latestMock().serverSend({ text: "one" });
      latestMock().serverSend({ text: "two" });
      // Allow microtasks to process.
      await new Promise((r) => setTimeout(r, 10));
      latestMock().serverClose();
      await done;

      expect(messages).toEqual([{ text: "one" }, { text: "two" }]);
    });

    it("throws WSError through iterator", async () => {
      const conn = connect();
      await conn.send({ text: "ping" });

      let caughtError: unknown;
      const done = (async () => {
        try {
          for await (const _ of conn) {
            // consume
          }
        } catch (err) {
          caughtError = err;
        }
      })();

      latestMock().serverSend({
        error: true,
        code: 4401,
        data: { message: "unauthorized" },
      });
      await done;

      expect(caughtError).toBeInstanceOf(WSError);
      expect((caughtError as WSError).code).toBe(4401);
      expect((caughtError as WSError).details).toEqual({
        message: "unauthorized",
      });
    });
  });

  describe("WSError", () => {
    it("rejects pending receives on error frame", async () => {
      const conn = connect();
      await conn.send({ text: "ping" });

      const promise = conn.receive();
      latestMock().serverSend({
        error: true,
        code: 4422,
        data: { message: "validation failed" },
      });
      await expect(promise).rejects.toBeInstanceOf(WSError);
    });

    it("rejects future receives after error frame", async () => {
      const conn = connect();
      await conn.send({ text: "ping" });

      latestMock().serverSend({
        error: true,
        code: 4401,
        data: { message: "unauthorized" },
      });
      // Allow microtasks.
      await new Promise((r) => setTimeout(r, 0));
      await expect(conn.receive()).rejects.toBeInstanceOf(WSError);
    });
  });

  describe("close", () => {
    it("rejects pending receives", async () => {
      const conn = connect();
      await conn.send({ text: "ping" });

      const promise = conn.receive();
      conn.close();
      await expect(promise).rejects.toThrow("WebSocket is closed");
    });

    it("rejects future receives", async () => {
      const conn = connect();
      await conn.send({ text: "ping" });
      conn.close();
      await expect(conn.receive()).rejects.toThrow("WebSocket is closed");
    });
  });

  describe("reconnection", () => {
    it("reconnects on unexpected close by default", async () => {
      vi.useFakeTimers();
      const conn = connect();
      await vi.runAllTimersAsync(); // open
      expect(mockInstances).toHaveLength(1);

      latestMock().serverClose();
      // Advance past reconnect delay (1000ms default).
      await vi.advanceTimersByTimeAsync(1000);
      expect(mockInstances).toHaveLength(2);

      // New connection opens — send should work.
      await vi.runAllTimersAsync();
      expect(latestMock().readyState).toBe(MockWebSocket.OPEN);
    });

    it("does not reconnect after explicit close", async () => {
      vi.useFakeTimers();
      const conn = connect();
      await vi.runAllTimersAsync();

      conn.close();
      await vi.advanceTimersByTimeAsync(5000);
      expect(mockInstances).toHaveLength(1);
    });

    it("does not reconnect on WSError", async () => {
      vi.useFakeTimers();
      const conn = connect();
      await vi.runAllTimersAsync();

      latestMock().serverSend({
        error: true,
        code: 4401,
        data: { message: "unauthorized" },
      });
      // WSError sets fatalError, close handler should not reconnect.
      latestMock().serverClose();
      await vi.advanceTimersByTimeAsync(5000);
      expect(mockInstances).toHaveLength(1);
    });

    it("respects reconnect: false", async () => {
      vi.useFakeTimers();
      const conn = connect({ reconnect: false });
      await vi.runAllTimersAsync();

      latestMock().serverClose();
      await vi.advanceTimersByTimeAsync(5000);
      expect(mockInstances).toHaveLength(1);
    });

    it("gives up after maxRetries", async () => {
      vi.useFakeTimers();
      const conn = connect({
        reconnect: { maxRetries: 2, baseDelay: 100, maxDelay: 1000 },
      });
      await vi.runAllTimersAsync();

      // Suppress auto-open so reconnected sockets don't reset retryCount.
      suppressAutoOpen = true;

      // First disconnect → reconnect attempt 1.
      latestMock().serverClose();
      await vi.advanceTimersByTimeAsync(100);
      expect(mockInstances).toHaveLength(2);

      // Second disconnect → reconnect attempt 2.
      latestMock().serverClose();
      await vi.advanceTimersByTimeAsync(200);
      expect(mockInstances).toHaveLength(3);

      // Third disconnect → no more retries.
      latestMock().serverClose();
      await vi.advanceTimersByTimeAsync(5000);
      expect(mockInstances).toHaveLength(3);

      suppressAutoOpen = false;
    });

    it("uses exponential backoff on consecutive failures", async () => {
      vi.useFakeTimers();
      const conn = connect({
        reconnect: { baseDelay: 100, maxDelay: 10000 },
      });
      await vi.runAllTimersAsync(); // open first connection

      // Suppress auto-open so reconnected sockets stay CONNECTING
      // (simulates server still being down).
      suppressAutoOpen = true;

      // Close the connection — triggers reconnect attempt 1 at 100ms.
      latestMock().serverClose();
      await vi.advanceTimersByTimeAsync(99);
      expect(mockInstances).toHaveLength(1);
      await vi.advanceTimersByTimeAsync(1); // 100ms — reconnect fires
      expect(mockInstances).toHaveLength(2);

      // Socket never opened — close it to trigger attempt 2 at 200ms.
      latestMock().serverClose();
      await vi.advanceTimersByTimeAsync(199);
      expect(mockInstances).toHaveLength(2);
      await vi.advanceTimersByTimeAsync(1); // 200ms — reconnect fires
      expect(mockInstances).toHaveLength(3);

      // Close again — attempt 3 at 400ms.
      latestMock().serverClose();
      await vi.advanceTimersByTimeAsync(399);
      expect(mockInstances).toHaveLength(3);
      await vi.advanceTimersByTimeAsync(1); // 400ms — reconnect fires
      expect(mockInstances).toHaveLength(4);

      suppressAutoOpen = false;
    });

    it("receive survives reconnection", async () => {
      vi.useFakeTimers();
      const conn = connect({ reconnect: { baseDelay: 100 } });
      await vi.runAllTimersAsync();

      // Start waiting for a message.
      const promise = conn.receive();

      // Disconnect and reconnect.
      latestMock().serverClose();
      await vi.advanceTimersByTimeAsync(100);
      await vi.runAllTimersAsync();

      // New connection delivers a message — pending receive should resolve.
      latestMock().serverSend({ text: "after-reconnect" });
      const msg = await promise;
      expect(msg).toEqual({ text: "after-reconnect" });
    });
  });

  describe("onopen / onclose", () => {
    it("calls onopen when connection opens", async () => {
      const conn = connect();
      const onopen = vi.fn();
      conn.onopen = onopen;

      // Wait for open.
      await conn.send({ text: "ping" });
      expect(onopen).toHaveBeenCalledTimes(1);
    });

    it("calls onclose on unexpected disconnect", async () => {
      const conn = connect();
      const onclose = vi.fn();
      conn.onclose = onclose;
      await conn.send({ text: "ping" });

      latestMock().serverClose();
      expect(onclose).toHaveBeenCalledTimes(1);
    });

    it("does not call onclose after explicit close", async () => {
      const conn = connect();
      const onclose = vi.fn();
      conn.onclose = onclose;
      await conn.send({ text: "ping" });

      conn.close();
      expect(onclose).not.toHaveBeenCalled();
    });

    it("calls onopen again after reconnection", async () => {
      vi.useFakeTimers();
      const conn = connect({ reconnect: { baseDelay: 100 } });
      const onopen = vi.fn();
      conn.onopen = onopen;

      await vi.runAllTimersAsync();
      expect(onopen).toHaveBeenCalledTimes(1);

      latestMock().serverClose();
      await vi.advanceTimersByTimeAsync(100);
      await vi.runAllTimersAsync();
      expect(onopen).toHaveBeenCalledTimes(2);
    });
  });
});
