import { describe, it, expect, vi, beforeEach } from "vitest";

// Minimal mock — @tanstack/svelte-query v6 uses Svelte 5 runes (.svelte.js)
// which can't run outside a Svelte compiler. We mock only to verify the
// calling convention (Accessor pattern) and capture the options our wrapper
// passes through to tanstack.
vi.mock("@tanstack/svelte-query", () => ({
  createQuery: vi.fn((optsFn: () => any, qcFn?: () => any) => ({
    _optsFn: optsFn,
    _qcFn: qcFn,
  })),
  createMutation: vi.fn((optsFn: () => any, qcFn?: () => any) => ({
    _optsFn: optsFn,
    _qcFn: qcFn,
  })),
  createInfiniteQuery: vi.fn((optsFn: () => any, qcFn?: () => any) => ({
    _optsFn: optsFn,
    _qcFn: qcFn,
  })),
}));

import {
  createQuery as tanstackCreateQuery,
  createMutation as tanstackCreateMutation,
  createInfiniteQuery as tanstackCreateInfiniteQuery,
} from "@tanstack/svelte-query";
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
  return {
    GET: vi.fn(),
    POST: vi.fn(),
    PUT: vi.fn(),
    DELETE: vi.fn(),
    PATCH: vi.fn(),
    HEAD: vi.fn(),
    OPTIONS: vi.fn(),
    TRACE: vi.fn(),
  } as unknown as FetchClient<MockPaths>;
}

describe("svelte-query createClient", () => {
  let fetchClient: FetchClient<MockPaths>;
  let api: ReturnType<typeof createClient<MockPaths>>;

  beforeEach(() => {
    vi.clearAllMocks();
    fetchClient = createMockFetchClient();
    api = createClient(fetchClient);
  });

  it("returns all expected methods", () => {
    expect(api).toHaveProperty("queryOptions");
    expect(api).toHaveProperty("createQuery");
    expect(api).toHaveProperty("createInfiniteQuery");
    expect(api).toHaveProperty("createMutation");
    expect(api).toHaveProperty("prefetchQuery");
    expect(api).toHaveProperty("prefetchInfiniteQuery");
  });

  // -------------------------------------------------------------------
  // queryOptions — pure logic, no tanstack dependency
  // -------------------------------------------------------------------
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

    it("spreads additional options through", () => {
      const opts = api.queryOptions("get", "/health", undefined, {
        enabled: false,
      } as any);
      expect(opts.enabled).toBe(false);
    });
  });

  // -------------------------------------------------------------------
  // queryFn — pure logic, exercises the openapi-fetch client wrapper
  // -------------------------------------------------------------------
  describe("queryFn", () => {
    const queryFnContext = (queryKey: any) => ({
      queryKey,
      signal: new AbortController().signal,
      meta: undefined,
      client: null as any,
    });

    it("calls GET on the fetch client and returns data", async () => {
      (fetchClient.GET as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { status: "ok" },
        error: undefined,
      });

      const opts = api.queryOptions("get", "/health");
      const result = await opts.queryFn(queryFnContext(["get", "/health"]));

      expect(fetchClient.GET).toHaveBeenCalledWith("/health", {
        signal: expect.any(AbortSignal),
      });
      expect(result).toEqual({ status: "ok" });
    });

    it("calls POST on the fetch client for post methods", async () => {
      (fetchClient.POST as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { message: "hi" },
        error: undefined,
      });

      const init = { body: { message: "hi" } };
      const opts = api.queryOptions("post", "/echo" as any, init);
      await opts.queryFn(queryFnContext(["post", "/echo", init]));

      expect(fetchClient.POST).toHaveBeenCalledWith("/echo", {
        signal: expect.any(AbortSignal),
        body: { message: "hi" },
      });
    });

    it("forwards init params to the fetch client", async () => {
      (fetchClient.GET as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { items: ["a"], next: 1 },
        error: undefined,
      });

      const init = { params: { query: { cursor: 10 } } };
      const opts = api.queryOptions("get", "/items", init);
      await opts.queryFn(queryFnContext(["get", "/items", init]));

      expect(fetchClient.GET).toHaveBeenCalledWith("/items", {
        signal: expect.any(AbortSignal),
        params: { query: { cursor: 10 } },
      });
    });

    it("throws the error when the fetch client returns an error", async () => {
      (fetchClient.GET as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: undefined,
        error: { message: "not found" },
      });

      const opts = api.queryOptions("get", "/health");
      await expect(
        opts.queryFn(queryFnContext(["get", "/health"])),
      ).rejects.toEqual({ message: "not found" });
    });
  });

  // -------------------------------------------------------------------
  // v6 Accessor contract — verify options/queryClient are passed as
  // functions, not raw objects (the v5→v6 breaking change)
  // -------------------------------------------------------------------
  describe("v6 Accessor contract", () => {
    describe("createQuery", () => {
      it("passes options as a function to tanstack createQuery", () => {
        api.createQuery("get", "/health");
        expect(tanstackCreateQuery).toHaveBeenCalledTimes(1);
        const [optsFn] = (tanstackCreateQuery as ReturnType<typeof vi.fn>).mock.calls[0];
        expect(optsFn).toBeTypeOf("function");
      });

      it("accessor returns correct queryKey and queryFn", () => {
        const result = api.createQuery("get", "/health") as any;
        const opts = result._optsFn();
        expect(opts.queryKey).toEqual(["get", "/health"]);
        expect(opts.queryFn).toBeTypeOf("function");
      });

      it("accessor includes init in queryKey", () => {
        const result = api.createQuery("get", "/items", {
          params: { query: { cursor: 3 } },
        }) as any;
        const opts = result._optsFn();
        expect(opts.queryKey).toEqual([
          "get",
          "/items",
          { params: { query: { cursor: 3 } } },
        ]);
      });

      it("wraps queryClient in an Accessor when provided", () => {
        const qc = {} as any;
        const result = api.createQuery("get", "/health", undefined, undefined, qc) as any;
        expect(result._qcFn).toBeTypeOf("function");
        expect(result._qcFn()).toBe(qc);
      });

      it("passes undefined queryClient Accessor when not provided", () => {
        const result = api.createQuery("get", "/health") as any;
        expect(result._qcFn).toBeUndefined();
      });
    });

    describe("createMutation", () => {
      it("passes options as a function to tanstack createMutation", () => {
        api.createMutation("post", "/echo");
        expect(tanstackCreateMutation).toHaveBeenCalledTimes(1);
        const [optsFn] = (tanstackCreateMutation as ReturnType<typeof vi.fn>).mock.calls[0];
        expect(optsFn).toBeTypeOf("function");
      });

      it("accessor returns correct mutationKey and mutationFn", () => {
        const result = api.createMutation("post", "/echo") as any;
        const opts = result._optsFn();
        expect(opts.mutationKey).toEqual(["post", "/echo"]);
        expect(opts.mutationFn).toBeTypeOf("function");
      });

      it("mutationFn calls the correct fetch client method", async () => {
        (fetchClient.POST as ReturnType<typeof vi.fn>).mockResolvedValue({
          data: { message: "hello" },
          error: undefined,
        });

        const result = api.createMutation("post", "/echo") as any;
        const opts = result._optsFn();
        const data = await opts.mutationFn({ body: { message: "hello" } });

        expect(fetchClient.POST).toHaveBeenCalledWith("/echo", {
          body: { message: "hello" },
        });
        expect(data).toEqual({ message: "hello" });
      });

      it("mutationFn throws on error response", async () => {
        (fetchClient.POST as ReturnType<typeof vi.fn>).mockResolvedValue({
          data: undefined,
          error: { code: 400, message: "bad request" },
        });

        const result = api.createMutation("post", "/echo") as any;
        const opts = result._optsFn();
        await expect(
          opts.mutationFn({ body: { message: "hello" } }),
        ).rejects.toEqual({ code: 400, message: "bad request" });
      });

      it("wraps queryClient in an Accessor when provided", () => {
        const qc = {} as any;
        const result = api.createMutation("post", "/echo", undefined, qc) as any;
        expect(result._qcFn).toBeTypeOf("function");
        expect(result._qcFn()).toBe(qc);
      });

      it("passes undefined queryClient Accessor when not provided", () => {
        const result = api.createMutation("post", "/echo") as any;
        expect(result._qcFn).toBeUndefined();
      });
    });

    describe("createInfiniteQuery", () => {
      const infiniteOpts = {
        initialPageParam: 0,
        getNextPageParam: (lastPage: any) => lastPage.next,
      } as any;

      it("passes options as a function to tanstack createInfiniteQuery", () => {
        api.createInfiniteQuery(
          "get", "/items",
          { params: { query: { cursor: 0 } } },
          infiniteOpts,
        );
        expect(tanstackCreateInfiniteQuery).toHaveBeenCalledTimes(1);
        const [optsFn] = (tanstackCreateInfiniteQuery as ReturnType<typeof vi.fn>).mock.calls[0];
        expect(optsFn).toBeTypeOf("function");
      });

      it("accessor returns correct queryKey and queryFn", () => {
        const result = api.createInfiniteQuery(
          "get", "/items",
          { params: { query: { cursor: 0 } } },
          infiniteOpts,
        ) as any;
        const opts = result._optsFn();
        expect(opts.queryKey).toEqual([
          "get", "/items",
          { params: { query: { cursor: 0 } } },
        ]);
        expect(opts.queryFn).toBeTypeOf("function");
      });

      it("strips pageParamName from options passed to tanstack", () => {
        const result = api.createInfiniteQuery(
          "get", "/items",
          { params: { query: { cursor: 0 } } },
          { ...infiniteOpts, pageParamName: "offset" },
        ) as any;
        const opts = result._optsFn();
        expect(opts).not.toHaveProperty("pageParamName");
      });

      it("infinite queryFn merges pageParam into query using default cursor key", async () => {
        (fetchClient.GET as ReturnType<typeof vi.fn>).mockResolvedValue({
          data: { items: ["a"], next: 1 },
          error: undefined,
        });

        const result = api.createInfiniteQuery(
          "get", "/items",
          { params: { query: { cursor: 0 } } },
          infiniteOpts,
        ) as any;
        const opts = result._optsFn();

        await opts.queryFn({
          queryKey: ["get", "/items", { params: { query: { cursor: 0 } } }],
          pageParam: 42,
          signal: new AbortController().signal,
        });

        expect(fetchClient.GET).toHaveBeenCalledWith("/items", expect.objectContaining({
          params: { query: { cursor: 42 } },
        }));
      });

      it("infinite queryFn uses custom pageParamName", async () => {
        (fetchClient.GET as ReturnType<typeof vi.fn>).mockResolvedValue({
          data: { items: ["a"], next: 1 },
          error: undefined,
        });

        const result = api.createInfiniteQuery(
          "get", "/items",
          { params: { query: { cursor: 0 } } },
          { ...infiniteOpts, pageParamName: "offset" },
        ) as any;
        const opts = result._optsFn();

        await opts.queryFn({
          queryKey: ["get", "/items", { params: { query: { cursor: 0 } } }],
          pageParam: 7,
          signal: new AbortController().signal,
        });

        expect(fetchClient.GET).toHaveBeenCalledWith("/items", expect.objectContaining({
          params: { query: { cursor: 0, offset: 7 } },
        }));
      });

      it("wraps queryClient in an Accessor when provided", () => {
        const qc = {} as any;
        const result = api.createInfiniteQuery(
          "get", "/items",
          { params: { query: { cursor: 0 } } },
          infiniteOpts,
          qc,
        ) as any;
        expect(result._qcFn).toBeTypeOf("function");
        expect(result._qcFn()).toBe(qc);
      });
    });
  });

  // -------------------------------------------------------------------
  // prefetchQuery / prefetchInfiniteQuery — pass-through to QueryClient
  // -------------------------------------------------------------------
  describe("prefetchQuery", () => {
    it("calls queryClient.prefetchQuery with queryKey and queryFn", async () => {
      const mockQueryClient = {
        prefetchQuery: vi.fn().mockResolvedValue(undefined),
      };

      await api.prefetchQuery(mockQueryClient as any, "get", "/health");

      expect(mockQueryClient.prefetchQuery).toHaveBeenCalledWith(
        expect.objectContaining({
          queryKey: ["get", "/health"],
          queryFn: expect.any(Function),
        }),
      );
    });

    it("includes init in prefetch queryKey", async () => {
      const mockQueryClient = {
        prefetchQuery: vi.fn().mockResolvedValue(undefined),
      };

      await api.prefetchQuery(
        mockQueryClient as any,
        "get", "/items",
        { params: { query: { cursor: 5 } } },
      );

      expect(mockQueryClient.prefetchQuery).toHaveBeenCalledWith(
        expect.objectContaining({
          queryKey: ["get", "/items", { params: { query: { cursor: 5 } } }],
        }),
      );
    });
  });

  describe("prefetchInfiniteQuery", () => {
    it("calls queryClient.prefetchInfiniteQuery with queryKey and queryFn", async () => {
      const mockQueryClient = {
        prefetchInfiniteQuery: vi.fn().mockResolvedValue(undefined),
      };

      await api.prefetchInfiniteQuery(
        mockQueryClient as any,
        "get", "/items",
        { params: { query: { cursor: 0 } } },
        {
          initialPageParam: 0,
          getNextPageParam: (lastPage: any) => lastPage.next,
        } as any,
      );

      expect(mockQueryClient.prefetchInfiniteQuery).toHaveBeenCalledWith(
        expect.objectContaining({
          queryKey: ["get", "/items", { params: { query: { cursor: 0 } } }],
          queryFn: expect.any(Function),
        }),
      );
    });

    it("prefetchInfiniteQuery queryFn merges pageParam", async () => {
      (fetchClient.GET as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { items: ["a"], next: 1 },
        error: undefined,
      });

      const mockQueryClient = {
        prefetchInfiniteQuery: vi.fn().mockResolvedValue(undefined),
      };

      await api.prefetchInfiniteQuery(
        mockQueryClient as any,
        "get", "/items",
        { params: { query: { cursor: 0 } } },
        {
          initialPageParam: 0,
          getNextPageParam: (lastPage: any) => lastPage.next,
        } as any,
      );

      // Extract the queryFn and call it with a pageParam
      const passedOpts = mockQueryClient.prefetchInfiniteQuery.mock.calls[0][0];
      await passedOpts.queryFn({
        queryKey: ["get", "/items", { params: { query: { cursor: 0 } } }],
        pageParam: 99,
        signal: new AbortController().signal,
      });

      expect(fetchClient.GET).toHaveBeenCalledWith("/items", expect.objectContaining({
        params: { query: { cursor: 99 } },
      }));
    });
  });
});
