import type { Plugin } from "vite";
import { resolve, relative } from "node:path";
import { createRequire } from "node:module";
import {
  MODULE_ID,
  RESOLVED_MODULE_ID,
  DEV_API_PREFIX,
  regenerateTypes,
  regenerateTypesFromServer,
  writeGeneratedFiles,
  patchTsConfigPaths,
  loadConfig,
  GoServerManager,
} from "shiftapi/internal";
import type { ShiftAPIPluginOptions } from "shiftapi/internal";

export default function shiftapiPlugin(opts?: ShiftAPIPluginOptions): Plugin {
  let serverEntry = "";
  let baseUrl = "/";
  let goRoot = "";
  let url = "http://localhost:8080";

  let virtualModuleSource = "";
  let generatedDts = "";
  let debounceTimer: ReturnType<typeof setTimeout> | null = null;
  let projectRoot = process.cwd();
  let configDir = "";
  let isDev = false;

  let goServer: GoServerManager | undefined;
  let devPort: number | null = null;

  async function regenerateTypesFromDev() {
    if (devPort === null) throw new Error("dev port not available");
    return regenerateTypesFromServer(
      `http://localhost:${devPort}`,
      baseUrl,
      isDev,
      generatedDts,
    );
  }

  function clearDebounce() {
    if (debounceTimer) {
      clearTimeout(debounceTimer);
      debounceTimer = null;
    }
  }

  return {
    name: "@shiftapi/vite-plugin",

    async configResolved(config) {
      projectRoot = config.root;

      const result = await loadConfig(projectRoot, opts?.configPath);
      configDir = result.configDir;

      serverEntry = result.config.server;
      baseUrl = result.config.baseUrl ?? "/";
      url = result.config.url ?? "http://localhost:8080";
      goRoot = configDir;

      goServer = new GoServerManager(serverEntry, goRoot);

      patchTsConfigPaths(projectRoot, configDir);
    },

    async config(_, env) {
      if (env.command === "serve") {
        isDev = true;

        // In dev mode the proxy target is set dynamically once the dev port
        // is known (via configureServer). Return a placeholder proxy that
        // will be replaced.
        return {
          server: {
            proxy: {
              [DEV_API_PREFIX]: {
                target: "http://localhost:0",
                rewrite: (path: string) =>
                  path.replace(new RegExp(`^${DEV_API_PREFIX}`), "") || "/",
              },
            },
          },
        };
      }
    },

    async configureServer(server) {
      if (!goServer) return;
      const gm = goServer;

      server.watcher.add(resolve(goRoot));

      // Start Go server and wait for the dev port.
      try {
        await gm.start();
        devPort = await gm.waitForDevPort();
      } catch (err) {
        console.error("[shiftapi] Go server failed to start:", err);
        return;
      }

      // Update proxy target to the actual dev port.
      const proxyConfig = server.config.server.proxy;
      if (proxyConfig && proxyConfig[DEV_API_PREFIX]) {
        const entry = proxyConfig[DEV_API_PREFIX];
        if (typeof entry === "object" && entry !== null) {
          (entry as Record<string, unknown>).target = `http://localhost:${devPort}`;
        }
      }

      const parsedUrl = new URL(url);
      console.log(
        `[shiftapi] API docs available at ${parsedUrl.protocol}//${parsedUrl.hostname}:${devPort}/docs`,
      );

      // Wake up the dev listener by sending a request to the user's Go server
      // port. This triggers ServeHTTP → sync.Once → devServe, which starts
      // serving on the dev port.
      const userUrl = url.replace(/\/$/, "");
      try {
        const resp = await fetch(`${userUrl}/openapi.json`, {
          signal: AbortSignal.timeout(30_000),
        });
        if (!resp.ok) {
          console.warn(`[shiftapi] Wake-up request returned ${resp.status}`);
        }
      } catch {
        // The user's server may not be reachable yet if http.ListenAndServe
        // hasn't been called. The plugin's polling in regenerateTypesFromDev
        // will retry via fetchSpec.
      }

      // Generate initial types from the dev server.
      try {
        const result = await regenerateTypesFromDev();
        generatedDts = result.types;
        virtualModuleSource = result.virtualModuleSource;
        writeGeneratedFiles(configDir, generatedDts, baseUrl);
      } catch (err) {
        console.error("[shiftapi] Failed to generate initial types:", err);
      }

      server.httpServer?.on("close", () => {
        clearDebounce();
        gm.stop().catch((err) => {
          console.error("[shiftapi] Failed to stop Go server on close:", err);
        });
      });
      process.on("exit", () => {
        clearDebounce();
        gm.forceKill();
      });
      process.on("SIGINT", async () => {
        clearDebounce();
        await gm.stop();
        process.exit();
      });
      process.on("SIGTERM", async () => {
        clearDebounce();
        await gm.stop();
        process.exit();
      });
    },

    async buildStart() {
      // In dev mode, types are generated in configureServer after the
      // Go server starts. For production builds, use the extract-and-exit flow.
      if (!isDev) {
        const result = await regenerateTypes(serverEntry, goRoot, baseUrl, false, generatedDts);
        generatedDts = result.types;
        virtualModuleSource = result.virtualModuleSource;
        writeGeneratedFiles(configDir, generatedDts, baseUrl);
      }
    },

    resolveId(id, importer) {
      if (id === MODULE_ID) {
        return RESOLVED_MODULE_ID;
      }
      if (id === "openapi-fetch" && importer === RESOLVED_MODULE_ID) {
        return createRequire(import.meta.url).resolve("openapi-fetch");
      }
    },

    load(id) {
      if (id === RESOLVED_MODULE_ID) {
        return virtualModuleSource;
      }
    },

    async handleHotUpdate({ file, server }) {
      if (!goServer) return;
      const gm = goServer;
      const resolvedGoRoot = resolve(goRoot);
      if (!file.endsWith(".go") || !file.startsWith(resolvedGoRoot)) {
        return;
      }

      console.log(
        `[shiftapi] Go file changed: ${relative(resolvedGoRoot, file)}`,
      );

      if (debounceTimer) clearTimeout(debounceTimer);
      debounceTimer = setTimeout(async () => {
        try {
          await gm.stop();
          await gm.start();
          devPort = await gm.waitForDevPort();

          // Wake up the dev listener.
          const userUrl = url.replace(/\/$/, "");
          try {
            await fetch(`${userUrl}/openapi.json`, {
              signal: AbortSignal.timeout(10_000),
            });
          } catch {
            // will retry via fetchSpec polling
          }

          const result = await regenerateTypesFromDev();
          if (result.changed) {
            generatedDts = result.types;
            virtualModuleSource = result.virtualModuleSource;
            writeGeneratedFiles(configDir, generatedDts, baseUrl);
            const mod = server.moduleGraph.getModuleById(RESOLVED_MODULE_ID);
            if (mod) {
              server.moduleGraph.invalidateModule(mod);
              server.ws.send({ type: "full-reload" });
            }
            console.log("[shiftapi] Types regenerated.");
          }
        } catch (err) {
          console.error("[shiftapi] Failed to regenerate:", err);
        }
      }, 500);
    },
  };
}

export { defineConfig } from "shiftapi";
export type { ShiftAPIConfig } from "shiftapi";
export type { ShiftAPIPluginOptions };
