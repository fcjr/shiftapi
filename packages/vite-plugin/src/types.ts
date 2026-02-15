export interface ShiftAPIPluginOptions {
  /** Path to the Go server entry point (e.g., "./cmd/server" or "./cmd/server/main.go") */
  server: string;

  /** Base URL for the API client at runtime (default: "/") */
  baseUrl?: string;

  /** Working directory for `go run` (default: process.cwd()) */
  goRoot?: string;

  /**
   * Address the Go server listens on (default: "http://localhost:8080").
   * Used to auto-configure the Vite proxy in dev mode.
   */
  url?: string;
}
