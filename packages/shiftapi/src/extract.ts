import { spawn } from "node:child_process";
import { mkdtemp, readFile, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";

/**
 * Extracts the OpenAPI spec from a Go shiftapi server by running it with
 * the shiftapidev build tag and SHIFTAPI_EXPORT_SPEC=1. The Go process
 * writes the spec JSON to a temp file during route registration (via
 * devNotifyRoute), then a background goroutine exits the process cleanly.
 *
 * This works even when the user's configured port is in use: the spec file
 * is written synchronously on the main goroutine during route registration,
 * before http.ListenAndServe is ever called. If the port is unavailable
 * and log.Fatal exits the process, the file already contains the complete spec.
 */
export async function extractSpec(
  serverEntry: string,
  goRoot: string,
): Promise<object> {
  const tmpDir = await mkdtemp(join(tmpdir(), "shiftapi-"));
  const exportFile = join(tmpDir, "spec.json");

  try {
    return await runExtract(serverEntry, goRoot, exportFile);
  } finally {
    await rm(tmpDir, { recursive: true, force: true }).catch(() => {});
  }
}

async function runExtract(
  serverEntry: string,
  goRoot: string,
  exportFile: string,
): Promise<object> {
  return new Promise((resolve, reject) => {
    const proc = spawn(
      "go",
      ["run", "-tags", "shiftapidev", serverEntry],
      {
        cwd: goRoot,
        stdio: ["ignore", "ignore", "pipe"],
        env: {
          ...process.env,
          SHIFTAPI_EXPORT_SPEC: "1",
          SHIFTAPI_EXPORT_FILE: exportFile,
        },
      },
    );

    let stderrBuf = "";
    let settled = false;

    function finish(err: Error | null, result?: object) {
      if (settled) return;
      settled = true;
      if (err) reject(err);
      else resolve(result!);
    }

    const timeout = setTimeout(() => {
      proc.kill();
      finish(
        new Error(
          `shiftapi: Timed out waiting for OpenAPI spec.\n` +
            `  Command: go run -tags shiftapidev ${serverEntry}\n` +
            `  CWD: ${goRoot}`,
        ),
      );
    }, 30_000);

    proc.on("error", (err) => {
      clearTimeout(timeout);
      finish(
        new Error(
          `shiftapi: Failed to start Go process: ${err.message}`,
        ),
      );
    });

    proc.stderr!.on("data", (chunk: Buffer) => {
      stderrBuf += chunk.toString();
    });

    // Read the spec file after the process exits (any exit code is OK).
    proc.on("exit", async () => {
      clearTimeout(timeout);

      try {
        const data = await readFile(exportFile, "utf-8");
        const spec = JSON.parse(data) as object;
        finish(null, spec);
      } catch (readErr) {
        finish(
          new Error(
            `shiftapi: Failed to read spec from export file.\n` +
              `  File: ${exportFile}\n` +
              `  Error: ${readErr instanceof Error ? readErr.message : String(readErr)}\n` +
              `  Command: go run -tags shiftapidev ${serverEntry}\n` +
              `  CWD: ${goRoot}\n` +
              `  Stderr: ${stderrBuf}`,
          ),
        );
      }
    });
  });
}
