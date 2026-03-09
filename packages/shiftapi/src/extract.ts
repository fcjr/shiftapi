import { execFileSync } from "node:child_process";
import { readFileSync, existsSync, rmSync, mkdtempSync } from "node:fs";
import { join } from "node:path";
import { tmpdir } from "node:os";

export interface ExtractedSpecs {
  openapi: object;
  asyncapi: object | null;
}

/**
 * Extracts the OpenAPI and AsyncAPI specs from a Go shiftapi server by
 * running it with the SHIFTAPI_EXPORT_SPEC (and optionally
 * SHIFTAPI_EXPORT_ASYNCAPI) environment variables set. The Go binary writes
 * the specs to the given paths and exits immediately.
 */
export function extractSpec(serverEntry: string, goRoot: string): object {
  const result = extractSpecs(serverEntry, goRoot);
  return result.openapi;
}

export function extractSpecs(serverEntry: string, goRoot: string): ExtractedSpecs {
  const tempDir = mkdtempSync(join(tmpdir(), "shiftapi-"));
  const specPath = join(tempDir, "openapi.json");
  const asyncSpecPath = join(tempDir, "asyncapi.json");

  try {
    execFileSync("go", ["run", "-tags", "shiftapidev", serverEntry], {
      cwd: goRoot,
      env: {
        ...process.env,
        SHIFTAPI_EXPORT_SPEC: specPath,
        SHIFTAPI_EXPORT_ASYNCAPI: asyncSpecPath,
      },
      stdio: ["ignore", "pipe", "pipe"],
      timeout: 30_000,
    });
  } catch (err: unknown) {
    // os.Exit(0) means exit code 0, so execFileSync does NOT throw.
    // If we're here, the Go code failed to compile or panicked.
    const stderr =
      err instanceof Error && "stderr" in err
        ? String((err as { stderr: unknown }).stderr)
        : "";
    throw new Error(
      `shiftapi: Failed to extract specs.\n` +
        `  Command: go run ${serverEntry}\n` +
        `  CWD: ${goRoot}\n` +
        `  Error: ${stderr || String(err)}`,
    );
  }

  let openapi: object;
  try {
    openapi = JSON.parse(readFileSync(specPath, "utf-8"));
  } catch {
    throw new Error(
      `shiftapi: Spec file was not created at ${specPath}.\n` +
        `  Make sure your Go server calls shiftapi.ListenAndServe().`,
    );
  }

  let asyncapi: object | null = null;
  try {
    if (existsSync(asyncSpecPath)) {
      asyncapi = JSON.parse(readFileSync(asyncSpecPath, "utf-8"));
    }
  } catch {
    // AsyncAPI spec is optional — ignore parse errors.
  }

  // Cleanup temp dir
  try {
    rmSync(tempDir, { recursive: true });
  } catch {
    // ignore cleanup errors
  }

  return { openapi, asyncapi };
}
