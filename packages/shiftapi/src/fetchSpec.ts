/**
 * Fetches the OpenAPI spec from a running Go server by polling GET /openapi.json
 * until a successful response is received.
 */
export async function fetchSpec(
  baseUrl: string,
  options?: { timeout?: number; interval?: number },
): Promise<object> {
  const timeout = options?.timeout ?? 60_000;
  const interval = options?.interval ?? 200;
  const specUrl = baseUrl.replace(/\/$/, "") + "/openapi.json";
  const deadline = Date.now() + timeout;

  let lastError: unknown;

  while (Date.now() < deadline) {
    try {
      const resp = await fetch(specUrl, { signal: AbortSignal.timeout(2000) });
      if (resp.ok) {
        return (await resp.json()) as object;
      }
      lastError = new Error(`GET ${specUrl} returned ${resp.status}`);
    } catch (err) {
      lastError = err;
    }

    await new Promise((resolve) => setTimeout(resolve, interval));
  }

  throw new Error(
    `shiftapi: Timed out waiting for OpenAPI spec at ${specUrl} after ${timeout}ms.\n` +
      `  Last error: ${lastError instanceof Error ? lastError.message : String(lastError)}`,
  );
}
