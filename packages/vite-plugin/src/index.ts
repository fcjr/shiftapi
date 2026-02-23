import type { Plugin } from "vite";
import { resolve, relative } from "node:path";
import { createRequire } from "node:module";
import { MODULE_ID, RESOLVED_MODULE_ID, DEV_API_PREFIX } from "./constants";
import { findFreePort } from "./ports";
import { GoServerManager } from "./goServer";
import { regenerateTypes as _regenerateTypes, writeDtsFile, patchTsConfig } from "./codegen";
import type { ShiftAPIPluginOptions } from "./types";

export default function shiftapiPlugin({
  server: serverEntry,
  baseUrl = "/",
  goRoot = process.cwd(),
  url = "http://localhost:8080",
}: ShiftAPIPluginOptions): Plugin {
  const parsedUrl = new URL(url);
  const basePort = parseInt(parsedUrl.port || "8080");
  let goPort = basePort;

  let virtualModuleSource = "";
  let generatedDts = "";
  let debounceTimer: ReturnType<typeof setTimeout> | null = null;
  let projectRoot = process.cwd();
  let isDev = false;

  const goServer = new GoServerManager(serverEntry, goRoot);

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

    configResolved(config) {
      projectRoot = config.root;
      patchTsConfig(projectRoot);
    },

    async config(_, env) {
      if (env.command === "serve") {
        isDev = true;
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
      server.watcher.add(resolve(goRoot));

      goServer
        .start(goPort)
        .then(() => {
          console.log(
            `[shiftapi] API docs available at http://localhost:${goPort}/docs`,
          );
        })
        .catch((err) => {
          console.error("[shiftapi] Go server failed to start:", err);
        });

      server.httpServer?.on("close", () => {
        clearDebounce();
        goServer.stop().catch((err) => {
          console.error("[shiftapi] Failed to stop Go server on close:", err);
        });
      });
      process.on("exit", () => {
        clearDebounce();
        goServer.forceKill();
      });
      process.on("SIGINT", async () => {
        clearDebounce();
        await goServer.stop();
        process.exit();
      });
      process.on("SIGTERM", async () => {
        clearDebounce();
        await goServer.stop();
        process.exit();
      });
    },

    async buildStart() {
      const result = await regenerateTypes();
      generatedDts = result.types;
      virtualModuleSource = result.virtualModuleSource;
      writeDtsFile(projectRoot, generatedDts);
    },

    resolveId(id, importer) {
      // Map bare "@shiftapi/client" imports to our virtual module
      if (id === MODULE_ID) {
        return RESOLVED_MODULE_ID;
      }
      // Resolve "openapi-fetch" from the plugin's node_modules so
      // consuming projects don't need to install it directly
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
          await goServer.stop();
          await goServer.start(goPort);

          const result = await regenerateTypes();
          if (result.changed) {
            generatedDts = result.types;
            virtualModuleSource = result.virtualModuleSource;
            writeDtsFile(projectRoot, generatedDts);
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

export type { ShiftAPIPluginOptions };
