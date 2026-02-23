import type { Plugin } from "vite";
import { resolve, relative } from "node:path";
import { createRequire } from "node:module";
import {
  MODULE_ID,
  RESOLVED_MODULE_ID,
  DEV_API_PREFIX,
  regenerateTypes as _regenerateTypes,
  writeGeneratedFiles,
  patchTsConfig,
  loadConfig,
} from "shiftapi/internal";
import { findFreePort } from "./ports";
import { GoServerManager } from "./goServer";
import type { ShiftAPIPluginOptions } from "./types";

export default function shiftapiPlugin(opts?: ShiftAPIPluginOptions): Plugin {
  let serverEntry = "";
  let baseUrl = "/";
  let goRoot = "";
  let url = "http://localhost:8080";

  let goPort = 8080;

  let virtualModuleSource = "";
  let generatedDts = "";
  let debounceTimer: ReturnType<typeof setTimeout> | null = null;
  let projectRoot = process.cwd();
  let configDir = "";
  let isDev = false;

  let goServer: GoServerManager | undefined;

  function regenerateTypes() {
    return _regenerateTypes(serverEntry, goRoot, baseUrl, isDev, generatedDts);
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

      patchTsConfig(projectRoot, configDir);
    },

    async config(_, env) {
      if (env.command === "serve") {
        isDev = true;

        // Parse url to get port — we might not have loaded config yet,
        // so use the default. The proxy target will be set with the actual port.
        const parsedUrl = new URL(url);
        const basePort = parseInt(parsedUrl.port || "8080");
        goPort = await findFreePort(basePort);
        if (goPort !== basePort) {
          console.log(`[shiftapi] Port ${basePort} is in use, using ${goPort}`);
        }

        const targetUrl = `${parsedUrl.protocol}//${parsedUrl.hostname}:${goPort}`;
        return {
          server: {
            proxy: {
              [DEV_API_PREFIX]: {
                target: targetUrl,
                rewrite: (path: string) =>
                  path.replace(new RegExp(`^${DEV_API_PREFIX}`), "") || "/",
              },
            },
          },
        };
      }
    },

    configureServer(server) {
      if (!goServer) return;
      const gm = goServer;

      server.watcher.add(resolve(goRoot));

      gm
        .start(goPort)
        .then(() => {
          const parsedUrl = new URL(url);
          console.log(
            `[shiftapi] API docs available at ${parsedUrl.protocol}//${parsedUrl.hostname}:${goPort}/docs`,
          );
        })
        .catch((err) => {
          console.error("[shiftapi] Go server failed to start:", err);
        });

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
      const result = await regenerateTypes();
      generatedDts = result.types;
      virtualModuleSource = result.virtualModuleSource;
      writeGeneratedFiles(configDir, generatedDts, baseUrl);
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
          await gm.start(goPort);

          const result = await regenerateTypes();
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
