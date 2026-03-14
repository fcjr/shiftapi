import {
  type CreateInfiniteQueryOptions,
  type CreateInfiniteQueryResult,
  type CreateMutationOptions,
  type CreateMutationResult,
  type CreateQueryOptions,
  type CreateQueryResult,
  type FetchQueryOptions,
  type GetNextPageParamFunction,
  type InfiniteData,
  type InitialPageParam,
  type QueryClient,
  type QueryFunctionContext,
  type SkipToken,
  createInfiniteQuery,
  createMutation,
  createQuery,
} from "@tanstack/svelte-query";
import type {
  ClientMethod,
  DefaultParamsOption,
  Client as FetchClient,
  FetchResponse,
  MaybeOptionalInit,
} from "openapi-fetch";
import type {
  HttpMethod,
  MediaType,
  PathsWithMethod,
  RequiredKeysOf,
} from "openapi-typescript-helpers";

type InferSelectReturnType<TData, TSelect> = TSelect extends (
  data: TData,
) => infer R
  ? R
  : TData;

type InitWithUnknowns<Init> = Init & { [key: string]: unknown };

export type QueryKey<
  Paths extends Record<string, Record<HttpMethod, {}>>,
  Method extends HttpMethod,
  Path extends PathsWithMethod<Paths, Method>,
  Init = MaybeOptionalInit<Paths[Path], Method>,
> = Init extends undefined
  ? readonly [Method, Path]
  : readonly [Method, Path, Init];

export type QueryOptionsFunction<
  Paths extends Record<string, Record<HttpMethod, {}>>,
  Media extends MediaType,
> = <
  Method extends HttpMethod,
  Path extends PathsWithMethod<Paths, Method>,
  Init extends MaybeOptionalInit<Paths[Path], Method>,
  Response extends Required<FetchResponse<Paths[Path][Method], Init, Media>>,
  Options extends Omit<
    CreateQueryOptions<
      Response["data"],
      Response["error"],
      InferSelectReturnType<Response["data"], Options["select"]>,
      QueryKey<Paths, Method, Path>
    >,
    "queryKey" | "queryFn"
  >,
>(
  method: Method,
  path: Path,
  ...[init, options]: RequiredKeysOf<Init> extends never
    ? [InitWithUnknowns<Init>?, Options?]
    : [InitWithUnknowns<Init>, Options?]
) => NoInfer<
  Omit<
    CreateQueryOptions<
      Response["data"],
      Response["error"],
      InferSelectReturnType<Response["data"], Options["select"]>,
      QueryKey<Paths, Method, Path>
    >,
    "queryFn"
  > & {
    queryFn: Exclude<
      CreateQueryOptions<
        Response["data"],
        Response["error"],
        InferSelectReturnType<Response["data"], Options["select"]>,
        QueryKey<Paths, Method, Path>
      >["queryFn"],
      SkipToken | undefined
    >;
  }
>;

export type CreateQueryMethod<
  Paths extends Record<string, Record<HttpMethod, {}>>,
  Media extends MediaType,
> = <
  Method extends HttpMethod,
  Path extends PathsWithMethod<Paths, Method>,
  Init extends MaybeOptionalInit<Paths[Path], Method>,
  Response extends Required<FetchResponse<Paths[Path][Method], Init, Media>>,
  Options extends Omit<
    CreateQueryOptions<
      Response["data"],
      Response["error"],
      InferSelectReturnType<Response["data"], Options["select"]>,
      QueryKey<Paths, Method, Path>
    >,
    "queryKey" | "queryFn"
  >,
>(
  method: Method,
  url: Path,
  ...[init, options, queryClient]: RequiredKeysOf<Init> extends never
    ? [InitWithUnknowns<Init>?, Options?, QueryClient?]
    : [InitWithUnknowns<Init>, Options?, QueryClient?]
) => CreateQueryResult<
  InferSelectReturnType<Response["data"], Options["select"]>,
  Response["error"]
>;

export type CreateInfiniteQueryMethod<
  Paths extends Record<string, Record<HttpMethod, {}>>,
  Media extends MediaType,
> = <
  Method extends HttpMethod,
  Path extends PathsWithMethod<Paths, Method>,
  Init extends MaybeOptionalInit<Paths[Path], Method>,
  Response extends Required<FetchResponse<Paths[Path][Method], Init, Media>>,
  Options extends Omit<
    CreateInfiniteQueryOptions<
      Response["data"],
      Response["error"],
      InfiniteData<Response["data"]>,
      QueryKey<Paths, Method, Path>,
      unknown
    >,
    "queryKey" | "queryFn"
  > & {
    pageParamName?: string;
  },
>(
  method: Method,
  url: Path,
  init: InitWithUnknowns<Init>,
  options: Options,
  queryClient?: QueryClient,
) => CreateInfiniteQueryResult<
  InfiniteData<Response["data"]>,
  Response["error"]
>;

export type CreateMutationMethod<
  Paths extends Record<string, Record<HttpMethod, {}>>,
  Media extends MediaType,
> = <
  Method extends HttpMethod,
  Path extends PathsWithMethod<Paths, Method>,
  Init extends MaybeOptionalInit<Paths[Path], Method>,
  Response extends Required<FetchResponse<Paths[Path][Method], Init, Media>>,
  Options extends Omit<
    CreateMutationOptions<Response["data"], Response["error"], Init>,
    "mutationKey" | "mutationFn"
  >,
>(
  method: Method,
  url: Path,
  options?: Options,
  queryClient?: QueryClient,
) => CreateMutationResult<Response["data"], Response["error"], Init>;

export type PrefetchQueryMethod<
  Paths extends Record<string, Record<HttpMethod, {}>>,
  Media extends MediaType,
> = <
  Method extends HttpMethod,
  Path extends PathsWithMethod<Paths, Method>,
  Init extends MaybeOptionalInit<Paths[Path], Method>,
  Response extends Required<FetchResponse<Paths[Path][Method], Init, Media>>,
  Options extends Omit<
    FetchQueryOptions<
      Response["data"],
      Response["error"],
      Response["data"],
      QueryKey<Paths, Method, Path>
    >,
    "queryKey" | "queryFn"
  >,
>(
  queryClient: QueryClient,
  method: Method,
  url: Path,
  ...[init, options]: RequiredKeysOf<Init> extends never
    ? [InitWithUnknowns<Init>?, Options?]
    : [InitWithUnknowns<Init>, Options?]
) => Promise<void>;

type FetchInfiniteQueryPages<TQueryFnData = unknown, TPageParam = unknown> =
  | {
      pages?: never;
    }
  | {
      pages: number;
      getNextPageParam: GetNextPageParamFunction<TPageParam, TQueryFnData>;
    };

export type PrefetchInfiniteQueryMethod<
  Paths extends Record<string, Record<HttpMethod, {}>>,
  Media extends MediaType,
> = <
  Method extends HttpMethod,
  Path extends PathsWithMethod<Paths, Method>,
  Init extends MaybeOptionalInit<Paths[Path], Method>,
  Response extends Required<FetchResponse<Paths[Path][Method], Init, Media>>,
  Options extends Omit<
    FetchQueryOptions<
      Response["data"],
      Response["error"],
      InfiniteData<Response["data"]>,
      QueryKey<Paths, Method, Path>,
      unknown
    >,
    "queryKey" | "queryFn" | "initialPageParam"
  > &
    InitialPageParam<unknown> &
    FetchInfiniteQueryPages<Response["data"], unknown> & {
      pageParamName?: string;
    },
>(
  queryClient: QueryClient,
  method: Method,
  url: Path,
  init: InitWithUnknowns<Init>,
  options: Options,
) => Promise<void>;

export interface OpenapiQueryClient<
  Paths extends {},
  Media extends MediaType = MediaType,
> {
  queryOptions: QueryOptionsFunction<Paths, Media>;
  createQuery: CreateQueryMethod<Paths, Media>;
  createInfiniteQuery: CreateInfiniteQueryMethod<Paths, Media>;
  createMutation: CreateMutationMethod<Paths, Media>;
  prefetchQuery: PrefetchQueryMethod<Paths, Media>;
  prefetchInfiniteQuery: PrefetchInfiniteQueryMethod<Paths, Media>;
}

export default function createClient<
  Paths extends {},
  Media extends MediaType = MediaType,
>(client: FetchClient<Paths, Media>): OpenapiQueryClient<Paths, Media> {
  const queryFn = async <
    Method extends HttpMethod,
    Path extends PathsWithMethod<Paths, Method>,
  >({
    queryKey: [method, path, init],
    signal,
  }: QueryFunctionContext<QueryKey<Paths, Method, Path>>) => {
    const mth = method.toUpperCase() as Uppercase<typeof method>;
    const fn = client[mth] as ClientMethod<Paths, typeof method, Media>;
    const { data, error } = await fn(path, { signal, ...(init as any) });
    if (error) {
      throw error;
    }
    return data;
  };

  const queryOptions: QueryOptionsFunction<Paths, Media> = (
    method,
    path,
    ...[init, options]
  ) => ({
    queryKey: (
      init === undefined
        ? ([method, path] as const)
        : ([method, path, init] as const)
    ) as QueryKey<Paths, typeof method, typeof path>,
    queryFn,
    ...options,
  });

  return {
    queryOptions,
    createQuery: (method, path, ...[init, options, queryClient]) =>
      createQuery(
        (() => ({
          queryKey: (
            init === undefined
              ? ([method, path] as const)
              : ([method, path, init] as const)
          ) as QueryKey<Paths, typeof method, typeof path>,
          queryFn,
          ...options,
        })) as any,
        queryClient,
      ),
    createInfiniteQuery: (method, path, init, options, queryClient) => {
      const { pageParamName = "cursor", ...restOptions } = options;
      const { queryKey } = queryOptions(method, path, init);
      return createInfiniteQuery(
        (() => ({
          queryKey,
          queryFn: async ({
            queryKey: [method, path, init],
            pageParam = 0,
            signal,
          }: any) => {
            const mth = (method as string).toUpperCase() as Uppercase<typeof method>;
            const fn = client[mth as keyof typeof client] as ClientMethod<
              Paths,
              typeof method,
              Media
            >;
            const mergedInit = {
              ...init,
              signal,
              params: {
                ...(init?.params || {}),
                query: {
                  ...(init?.params as { query?: DefaultParamsOption })?.query,
                  [pageParamName]: pageParam,
                },
              },
            };
            const { data, error } = await fn(path, mergedInit as any);
            if (error) {
              throw error;
            }
            return data;
          },
          ...restOptions,
        })) as any,
        queryClient,
      );
    },
    createMutation: (method, path, options, queryClient) =>
      createMutation(
        (() => ({
          mutationKey: [method, path],
          mutationFn: async (init: any) => {
            const mth = method.toUpperCase() as Uppercase<typeof method>;
            const fn = client[mth] as ClientMethod<
              Paths,
              typeof method,
              Media
            >;
            const { data, error } = await fn(
              path,
              init as InitWithUnknowns<typeof init>,
            );
            if (error) {
              throw error;
            }
            return data as Exclude<typeof data, undefined>;
          },
          ...options,
        })) as any,
        queryClient,
      ),
    prefetchQuery: (queryClient, method, path, ...[init, options]) => {
      return queryClient.prefetchQuery(
        queryOptions(
          method,
          path,
          init as InitWithUnknowns<typeof init>,
          options,
        ),
      );
    },
    prefetchInfiniteQuery: (queryClient, method, path, init, options) => {
      const { pageParamName = "cursor" } = options;
      const { queryKey } = queryOptions(method, path, init);
      return queryClient.prefetchInfiniteQuery({
        queryKey,
        queryFn: async ({
          queryKey: [method, path, init],
          pageParam = 0,
          signal,
        }) => {
          const mth = method.toUpperCase() as Uppercase<typeof method>;
          const fn = client[mth] as ClientMethod<Paths, typeof method, Media>;
          const mergedInit = {
            ...init,
            signal,
            params: {
              ...(init?.params || {}),
              query: {
                ...(init?.params as { query?: DefaultParamsOption })?.query,
                [pageParamName]: pageParam,
              },
            },
          };
          const { data, error } = await fn(path, mergedInit as any);
          if (error) {
            throw error;
          }
          return data;
        },
        ...options,
      });
    },
  };
}
