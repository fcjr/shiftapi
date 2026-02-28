import { resolve } from "node:path";
import { watch, type FSWatcher } from "node:fs";
import { createRequire } from "node:module";
import {
  loadConfig,
  findConfigDir,
  regenerateTypes as _regenerateTypes,
  writeGeneratedFiles,
  patchTsConfigPaths,
  nextClientJsTemplate,
  DEV_API_PREFIX,
  GoServerManager,
  findFreePort,
} from "shiftapi/internal";
import type { ShiftAPIPluginOptions } from "shiftapi/internal";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type NextConfigObject = Record<string, any>;
type NextConfigFunction = (...args: unknown[]) => NextConfigObject | Promise<NextConfigObject>;
type NextConfig = NextConfigObject | NextConfigFunction;

type Rewrite = { source: string; destination: string };
type RewritesResult =
  | Rewrite[]
  | { beforeFiles: Rewrite[]; afterFiles: Rewrite[]; fallback: Rewrite[] };
type RewritesFn = () => Promise<RewritesResult>;

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type WebpackConfigObj = Record<string, any>;
type WebpackFn = (config: WebpackConfigObj, context: unknown) => WebpackConfigObj;

interface InitResult {
  port: number;
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function isThenable(value: unknown): value is PromiseLike<unknown> {
  return (
    typeof value === "object" &&
    value !== null &&
    typeof (value as Record<string, unknown>).then === "function"
  );
}

// ---------------------------------------------------------------------------
// withShiftAPI — synchronous, like Sentry's withSentryConfig
// ---------------------------------------------------------------------------

/**
 * Wraps a Next.js config with ShiftAPI integration (Go server management,
 * typed API client generation, dev proxy).
 *
 * Supports both object and function configs.
 *
 * @param nextConfig - The user's exported Next.js config
 * @param opts - ShiftAPI-specific options
 * @returns The wrapped Next.js config (same shape as the input)
 */
export function withShiftAPI<C extends NextConfig>(
  nextConfig?: C,
  opts?: ShiftAPIPluginOptions,
): C {
  const cast = (nextConfig ?? {}) as NextConfigObject | NextConfigFunction;

  if (typeof cast === "function") {
    return function (this: unknown, ...args: unknown[]) {
      const result = (cast as NextConfigFunction).apply(this, args);
      if (isThenable(result)) {
        return (result as Promise<NextConfigObject>).then((cfg) =>
          applyShiftAPI(cfg, opts),
        );
      }
      return applyShiftAPI(result as NextConfigObject, opts);
    } as unknown as C;
  }

  return applyShiftAPI(cast as NextConfigObject, opts) as unknown as C;
}

// ---------------------------------------------------------------------------
// Core config patching (sync return, async work deferred)
// ---------------------------------------------------------------------------

function applyShiftAPI(
  nextConfig: NextConfigObject,
  opts?: ShiftAPIPluginOptions,
): NextConfigObject {
  const projectRoot = process.cwd();
  const configDir = findConfigDir(projectRoot, opts?.configPath);

  if (!configDir) {
    console.warn(
      "[shiftapi] Could not find shiftapi.config.ts. Skipping ShiftAPI integration.",
    );
    return nextConfig;
  }

  const isDev = process.env.NODE_ENV !== "production";
  const shiftapiClientPath = resolve(configDir, ".shiftapi", "client.js");

  // Resolve to the ESM entry point (require.resolve would pick the CJS one)
  const require = createRequire(import.meta.url);
  const openapiPkgDir = resolve(require.resolve("openapi-fetch/package.json"), "..");
  const openapiPath = resolve(openapiPkgDir, "dist", "index.js");

  // Kick off async initialization (Go server, type generation) immediately.
  // The promise is awaited lazily by webpack (beforeCompile hook) and rewrites.
  const initPromise = initializeAsync(projectRoot, configDir, isDev, opts);

  const patched: NextConfigObject = { ...nextConfig };

  // -- Webpack: sync aliases + async plugin to block until types exist ------
  const existingWebpack = nextConfig.webpack as WebpackFn | undefined;

  patched.webpack = (config: WebpackConfigObj, context: unknown) => {
    const cfg = existingWebpack ? existingWebpack(config, context) : config;

    // Aliases
    cfg.resolve = cfg.resolve || {};
    cfg.resolve.alias = cfg.resolve.alias || {};
    cfg.resolve.alias["@shiftapi/client"] = shiftapiClientPath;
    cfg.resolve.alias["openapi-fetch"] = openapiPath;

    // Block compilation until async init is done
    cfg.plugins = cfg.plugins || [];
    cfg.plugins.push({
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      apply(compiler: any) {
        compiler.hooks.beforeCompile.tapPromise("ShiftAPI", async () => {
          await initPromise;
        });
      },
    });

    return cfg;
  };

  // -- Turbopack aliases (top-level in Next.js 15.3+) ----------------------
  const existingTurbopack = (nextConfig.turbopack ?? {}) as Record<string, unknown>;
  const existingResolveAlias = (existingTurbopack.resolveAlias ?? {}) as Record<string, string>;

  patched.turbopack = {
    ...existingTurbopack,
    resolveAlias: {
      ...existingResolveAlias,
      "@shiftapi/client": shiftapiClientPath,
      "openapi-fetch": openapiPath,
    },
  };

  // -- Rewrites (dev proxy) ------------------------------------------------
  if (isDev) {
    const existingRewrites = nextConfig.rewrites as RewritesFn | undefined;

    patched.rewrites = async () => {
      const { port } = await initPromise;
      const shiftapiRewrite: Rewrite = {
        source: `${DEV_API_PREFIX}/:path*`,
        destination: `http://localhost:${port}/:path*`,
      };

      if (!existingRewrites) {
        return {
          beforeFiles: [shiftapiRewrite],
          afterFiles: [],
          fallback: [],
        };
      }

      const existing = await existingRewrites();

      if (Array.isArray(existing)) {
        return {
          beforeFiles: [shiftapiRewrite],
          afterFiles: existing,
          fallback: [],
        };
      }

      return {
        beforeFiles: [shiftapiRewrite, ...(existing.beforeFiles ?? [])],
        afterFiles: existing.afterFiles ?? [],
        fallback: existing.fallback ?? [],
      };
    };
  }

  return patched;
}

// ---------------------------------------------------------------------------
// Async initialization (fire-and-forget, awaited lazily)
// ---------------------------------------------------------------------------

async function initializeAsync(
  projectRoot: string,
  configDir: string,
  isDev: boolean,
  opts?: ShiftAPIPluginOptions,
): Promise<InitResult> {
  const { config } = await loadConfig(projectRoot, opts?.configPath);

  const serverEntry = config.server;
  const baseUrl = config.baseUrl ?? "/";
  const goRoot = configDir;
  const parsedUrl = new URL(config.url ?? "http://localhost:8080");
  const basePort = parseInt(parsedUrl.port || "8080");

  if (isDev) {
    return initializeDev(
      projectRoot,
      configDir,
      serverEntry,
      baseUrl,
      goRoot,
      parsedUrl,
      basePort,
    );
  }

  return initializeBuild(projectRoot, configDir, serverEntry, baseUrl, goRoot, basePort);
}

async function initializeDev(
  projectRoot: string,
  configDir: string,
  serverEntry: string,
  baseUrl: string,
  goRoot: string,
  parsedUrl: URL,
  basePort: number,
): Promise<InitResult> {
  const goPort = await findFreePort(basePort);
  if (goPort !== basePort) {
    console.log(`[shiftapi] Port ${basePort} is in use, using ${goPort}`);
  }

  const goServer = new GoServerManager(serverEntry, goRoot);

  // Start Go server
  try {
    await goServer.start(goPort);
    console.log(
      `[shiftapi] API docs available at ${parsedUrl.protocol}//${parsedUrl.hostname}:${goPort}/docs`,
    );
  } catch (err) {
    console.error("[shiftapi] Go server failed to start:", err);
  }

  // Generate types
  let generatedDts = "";
  try {
    const result = await _regenerateTypes(serverEntry, goRoot, baseUrl, true, "");
    generatedDts = result.types;
    const clientJs = nextClientJsTemplate(goPort, baseUrl, DEV_API_PREFIX);
    writeGeneratedFiles(configDir, generatedDts, baseUrl, { clientJsContent: clientJs });
    patchTsConfigPaths(projectRoot, configDir);
    console.log("[shiftapi] Types generated.");
  } catch (err) {
    console.error("[shiftapi] Failed to generate types:", err);
  }

  // File watcher for .go changes
  let debounceTimer: ReturnType<typeof setTimeout> | null = null;
  let watcher: FSWatcher | undefined;

  try {
    watcher = watch(resolve(goRoot), { recursive: true }, (_event, filename) => {
      if (!filename || !filename.endsWith(".go")) return;

      console.log(`[shiftapi] Go file changed: ${filename}`);

      if (debounceTimer) clearTimeout(debounceTimer);
      debounceTimer = setTimeout(async () => {
        try {
          await goServer.stop();
          await goServer.start(goPort);

          const result = await _regenerateTypes(
            serverEntry,
            goRoot,
            baseUrl,
            true,
            generatedDts,
          );
          if (result.changed) {
            generatedDts = result.types;
            const clientJs = nextClientJsTemplate(goPort, baseUrl, DEV_API_PREFIX);
            writeGeneratedFiles(configDir, generatedDts, baseUrl, {
              clientJsContent: clientJs,
            });
            console.log("[shiftapi] Types regenerated.");
          }
        } catch (err) {
          console.error("[shiftapi] Failed to regenerate:", err);
        }
      }, 500);
    });
  } catch {
    console.warn("[shiftapi] Could not set up file watcher for Go files.");
  }

  // Cleanup handlers
  function cleanup() {
    if (debounceTimer) clearTimeout(debounceTimer);
    watcher?.close();
    goServer.forceKill();
  }

  process.on("exit", cleanup);
  process.on("SIGINT", async () => {
    if (debounceTimer) clearTimeout(debounceTimer);
    watcher?.close();
    await goServer.stop();
    process.exit();
  });
  process.on("SIGTERM", async () => {
    if (debounceTimer) clearTimeout(debounceTimer);
    watcher?.close();
    await goServer.stop();
    process.exit();
  });

  return { port: goPort };
}

async function initializeBuild(
  projectRoot: string,
  configDir: string,
  serverEntry: string,
  baseUrl: string,
  goRoot: string,
  basePort: number,
): Promise<InitResult> {
  try {
    const result = await _regenerateTypes(serverEntry, goRoot, baseUrl, false, "");
    const clientJs = nextClientJsTemplate(basePort, baseUrl);
    writeGeneratedFiles(configDir, result.types, baseUrl, { clientJsContent: clientJs });
    patchTsConfigPaths(projectRoot, configDir);
    console.log("[shiftapi] Types generated for build.");
  } catch (err) {
    console.error("[shiftapi] Failed to generate types for build:", err);
  }

  return { port: basePort };
}

export { defineConfig } from "shiftapi";
export type { ShiftAPIConfig } from "shiftapi";
export type { ShiftAPIPluginOptions };
