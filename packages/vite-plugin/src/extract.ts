import { execFileSync } from "node:child_process";
import { readFileSync, unlinkSync, rmSync, mkdtempSync } from "node:fs";
import { join } from "node:path";
import { tmpdir } from "node:os";

/**
 * Extracts the OpenAPI spec from a Go shiftapi server by running it with
 * the SHIFTAPI_EXPORT_SPEC environment variable set. The Go binary writes
 * the spec to the given path and exits immediately.
 */
export function extractSpec(serverEntry: string, goRoot: string): object {
  const tempDir = mkdtempSync(join(tmpdir(), "shiftapi-"));
  const specPath = join(tempDir, "openapi.json");

  try {
    execFileSync("go", ["run", "-tags", "shiftapidev", serverEntry], {
      cwd: goRoot,
      env: {
        ...process.env,
        SHIFTAPI_EXPORT_SPEC: specPath,
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
      `@shiftapi/vite-plugin: Failed to extract OpenAPI spec.\n` +
        `  Command: go run ${serverEntry}\n` +
        `  CWD: ${goRoot}\n` +
        `  Error: ${stderr || String(err)}`,
    );
  }

  let raw: string;
  try {
    raw = readFileSync(specPath, "utf-8");
  } catch {
    throw new Error(
      `@shiftapi/vite-plugin: Spec file was not created at ${specPath}.\n` +
        `  Make sure your Go server calls shiftapi.ListenAndServe().`,
    );
  }

  // Cleanup temp dir
  try {
    unlinkSync(specPath);
    rmSync(tempDir, { recursive: true });
  } catch {
    // ignore cleanup errors
  }

  return JSON.parse(raw);
}
