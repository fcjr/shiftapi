import { describe, it, expect, vi, beforeEach } from "vitest";
import {
  createQuery as mockCreateQueryFn,
  createMutation as mockCreateMutationFn,
  createInfiniteQuery as mockCreateInfiniteQueryFn,
} from "@tanstack/svelte-query";

// Mock @tanstack/svelte-query before importing the module under test
vi.mock("@tanstack/svelte-query", () => ({
  createQuery: vi.fn((...args: unknown[]) => {
    // Svelte query takes a function that returns options
    const optsFn = args[0] as () => Record<string, unknown>;
    return { _opts: optsFn() };
  }),
  createMutation: vi.fn((...args: unknown[]) => {
    const optsFn = args[0] as () => Record<string, unknown>;
    return { _opts: optsFn() };
  }),
  createInfiniteQuery: vi.fn((...args: unknown[]) => {
    const optsFn = args[0] as () => Record<string, unknown>;
    return { _opts: optsFn() };
  }),
}));

import createClient from "../svelte-query";
import type { Client as FetchClient } from "openapi-fetch";

type MockPaths = {
  "/health": {
    get: {
      responses: { 200: { content: { "application/json": { status: string } } } };
    };
  };
  "/echo": {
    post: {
      requestBody: { content: { "application/json": { message: string } } };
      responses: { 200: { content: { "application/json": { message: string } } } };
    };
  };
  "/items": {
    get: {
      parameters: { query: { cursor?: number } };
      responses: { 200: { content: { "application/json": { items: string[]; next: number } } } };
    };
  };
};

function createMockFetchClient() {
  const mockClient = {
    GET: vi.fn(),
    POST: vi.fn(),
    PUT: vi.fn(),
    DELETE: vi.fn(),
    PATCH: vi.fn(),
    HEAD: vi.fn(),
    OPTIONS: vi.fn(),
    TRACE: vi.fn(),
  };
  return mockClient as unknown as FetchClient<MockPaths>;
}

describe("createClient", () => {
  let fetchClient: FetchClient<MockPaths>;
  let api: ReturnType<typeof createClient<MockPaths>>;

  beforeEach(() => {
    vi.clearAllMocks();
    fetchClient = createMockFetchClient();
    api = createClient(fetchClient);
  });

  it("returns an object with all expected methods", () => {
    expect(api).toHaveProperty("queryOptions");
    expect(api).toHaveProperty("createQuery");
    expect(api).toHaveProperty("createInfiniteQuery");
    expect(api).toHaveProperty("createMutation");
    expect(api).toHaveProperty("prefetchQuery");
    expect(api).toHaveProperty("prefetchInfiniteQuery");
  });

  describe("queryOptions", () => {
    it("builds query key without init", () => {
      const opts = api.queryOptions("get", "/health");
      expect(opts.queryKey).toEqual(["get", "/health"]);
      expect(opts.queryFn).toBeTypeOf("function");
    });

    it("builds query key with init", () => {
      const opts = api.queryOptions("get", "/items", {
        params: { query: { cursor: 5 } },
      });
      expect(opts.queryKey).toEqual([
        "get",
        "/items",
        { params: { query: { cursor: 5 } } },
      ]);
    });

    it("spreads additional options", () => {
      const opts = api.queryOptions("get", "/health", undefined, {
        enabled: false,
      } as any);
      expect(opts.enabled).toBe(false);
    });
  });

  describe("queryFn", () => {
    it("calls the correct client method and returns data", async () => {
      const mockGet = fetchClient.GET as ReturnType<typeof vi.fn>;
      mockGet.mockResolvedValue({
        data: { status: "ok" },
        error: undefined,
      });

      const opts = api.queryOptions("get", "/health");
      const result = await opts.queryFn({
        queryKey: ["get", "/health"] as any,
        signal: new AbortController().signal,
        meta: undefined,
        client: null as any,
      });

      expect(mockGet).toHaveBeenCalledWith("/health", {
        signal: expect.any(AbortSignal),
      });
      expect(result).toEqual({ status: "ok" });
    });

    it("passes init params to the client", async () => {
      const mockGet = fetchClient.GET as ReturnType<typeof vi.fn>;
      mockGet.mockResolvedValue({
        data: { items: ["a"], next: 1 },
        error: undefined,
      });

      const init = { params: { query: { cursor: 10 } } };
      const opts = api.queryOptions("get", "/items", init);
      await opts.queryFn({
        queryKey: ["get", "/items", init] as any,
        signal: new AbortController().signal,
        meta: undefined,
        client: null as any,
      });

      expect(mockGet).toHaveBeenCalledWith("/items", {
        signal: expect.any(AbortSignal),
        params: { query: { cursor: 10 } },
      });
    });

    it("throws on error response", async () => {
      const mockGet = fetchClient.GET as ReturnType<typeof vi.fn>;
      mockGet.mockResolvedValue({
        data: undefined,
        error: { message: "not found" },
      });

      const opts = api.queryOptions("get", "/health");
      await expect(
        opts.queryFn({
          queryKey: ["get", "/health"] as any,
          signal: new AbortController().signal,
          meta: undefined,
          client: null as any,
        }),
      ).rejects.toEqual({ message: "not found" });
    });
  });

  describe("createQuery", () => {
    it("delegates to @tanstack/svelte-query createQuery", () => {
      api.createQuery("get", "/health");
      expect(mockCreateQueryFn).toHaveBeenCalledTimes(1);
    });

    it("passes the correct query key and queryFn", () => {
      const result = api.createQuery("get", "/health") as any;
      expect(result._opts.queryKey).toEqual(["get", "/health"]);
      expect(result._opts.queryFn).toBeTypeOf("function");
    });

    it("includes init in query key when provided", () => {
      const result = api.createQuery("get", "/items", {
        params: { query: { cursor: 3 } },
      }) as any;
      expect(result._opts.queryKey).toEqual([
        "get",
        "/items",
        { params: { query: { cursor: 3 } } },
      ]);
    });
  });

  describe("createMutation", () => {
    it("delegates to @tanstack/svelte-query createMutation", () => {
      api.createMutation("post", "/echo");
      expect(mockCreateMutationFn).toHaveBeenCalledTimes(1);
    });

    it("sets the correct mutation key", () => {
      const result = api.createMutation("post", "/echo") as any;
      expect(result._opts.mutationKey).toEqual(["post", "/echo"]);
    });

    it("mutationFn calls the correct client method", async () => {
      const mockPost = fetchClient.POST as ReturnType<typeof vi.fn>;
      mockPost.mockResolvedValue({
        data: { message: "hello" },
        error: undefined,
      });

      const result = api.createMutation("post", "/echo") as any;
      const data = await result._opts.mutationFn({
        body: { message: "hello" },
      });

      expect(mockPost).toHaveBeenCalledWith("/echo", {
        body: { message: "hello" },
      });
      expect(data).toEqual({ message: "hello" });
    });

    it("mutationFn throws on error", async () => {
      const mockPost = fetchClient.POST as ReturnType<typeof vi.fn>;
      mockPost.mockResolvedValue({
        data: undefined,
        error: { code: 400, message: "bad request" },
      });

      const result = api.createMutation("post", "/echo") as any;
      await expect(
        result._opts.mutationFn({ body: { message: "hello" } }),
      ).rejects.toEqual({ code: 400, message: "bad request" });
    });
  });

  describe("createInfiniteQuery", () => {
    it("delegates to @tanstack/svelte-query createInfiniteQuery", () => {
      api.createInfiniteQuery(
        "get",
        "/items",
        { params: { query: { cursor: 0 } } },
        {
          initialPageParam: 0,
          getNextPageParam: (lastPage: any) => lastPage.next,
        } as any,
      );
      expect(mockCreateInfiniteQueryFn).toHaveBeenCalledTimes(1);
    });

    it("uses custom pageParamName", () => {
      const mockGet = fetchClient.GET as ReturnType<typeof vi.fn>;
      mockGet.mockResolvedValue({
        data: { items: ["a"], next: 1 },
        error: undefined,
      });

      const result = api.createInfiniteQuery(
        "get",
        "/items",
        { params: { query: { cursor: 0 } } },
        {
          initialPageParam: 0,
          getNextPageParam: (lastPage: any) => lastPage.next,
          pageParamName: "offset",
        } as any,
      ) as any;

      // The pageParamName should not leak into rest options
      expect(result._opts).not.toHaveProperty("pageParamName");
    });
  });

  describe("prefetchQuery", () => {
    it("calls queryClient.prefetchQuery with correct options", async () => {
      const mockQueryClient = {
        prefetchQuery: vi.fn().mockResolvedValue(undefined),
      };

      await api.prefetchQuery(
        mockQueryClient as any,
        "get",
        "/health",
      );

      expect(mockQueryClient.prefetchQuery).toHaveBeenCalledWith(
        expect.objectContaining({
          queryKey: ["get", "/health"],
          queryFn: expect.any(Function),
        }),
      );
    });
  });
});
