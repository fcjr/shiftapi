// Fallback type declaration for the @shiftapi/client virtual module.
// Run `vite dev` or `vite build` to generate the real API-specific types.
declare module "@shiftapi/client" {
  import type createClient from "openapi-fetch";

  /** OpenAPI paths type — generated from your Go API */
  export type paths = Record<string, any>;
  /** OpenAPI components type — generated from your Go API */
  export type components = Record<string, any>;
  /** OpenAPI operations type — generated from your Go API */
  export type operations = Record<string, any>;

  /** Pre-configured, fully-typed API client */
  export const client: ReturnType<typeof createClient<paths>>;
  export { createClient };
}
