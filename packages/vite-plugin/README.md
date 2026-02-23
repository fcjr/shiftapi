<p align="center">
	<picture>
		<source media="(prefers-color-scheme: dark)" srcset="https://raw.githubusercontent.com/fcjr/shiftapi/main/assets/logo-dark.svg">
		<img src="https://raw.githubusercontent.com/fcjr/shiftapi/main/assets/logo.svg" alt="ShiftAPI Logo">
	</picture>
</p>

# @shiftapi/vite-plugin

Vite plugin that generates fully-typed TypeScript clients from [ShiftAPI](https://github.com/fcjr/shiftapi) Go servers. Get end-to-end type safety between your Go API and your frontend with zero manual type definitions.

## How it works

1. Reads your `shiftapi.config.ts` (powered by [`shiftapi`](https://www.npmjs.com/package/shiftapi))
2. Extracts the OpenAPI 3.1 spec from your Go server at build time
3. Generates TypeScript types using `openapi-typescript`
4. Provides a virtual `@shiftapi/client` module with a pre-configured, fully-typed API client
5. In dev mode, watches `.go` files and regenerates types on changes
6. Auto-configures Vite's dev server proxy to forward API requests to your Go server

## Installation

```bash
npm install -D shiftapi @shiftapi/vite-plugin
# or
pnpm add -D shiftapi @shiftapi/vite-plugin
```

**Peer dependency:** `vite` (v6).

## Setup

### shiftapi.config.ts

Create a `shiftapi.config.ts` in your project root:

```ts
import { defineConfig } from "shiftapi";

export default defineConfig({
  server: "./cmd/server",
});
```

### vite.config.ts

```ts
import { defineConfig } from "vite";
import shiftapi from "@shiftapi/vite-plugin";

export default defineConfig({
  plugins: [shiftapi()],
});
```

### Config options

Options are set in `shiftapi.config.ts` (see [`shiftapi`](https://www.npmjs.com/package/shiftapi)):

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `server` | `string` | **(required)** | Path to the Go server entry point (e.g. `"./cmd/server"`) |
| `baseUrl` | `string` | `"/"` | Fallback base URL for the API client. Can be overridden via `VITE_SHIFTAPI_BASE_URL`. |
| `url` | `string` | `"http://localhost:8080"` | Address the Go server listens on. Used to auto-configure the Vite dev proxy. |

### Plugin options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `configPath` | `string` | auto-detected | Explicit path to `shiftapi.config.ts` (for advanced use only) |

## Usage

Import the typed client in your frontend code:

```ts
import { client } from "@shiftapi/client";

const { data, error } = await client.GET("/greet", {
  params: { query: { name: "World" } },
});
// `data` and `error` are fully typed based on your Go handler signatures
```

The `createClient` factory is also exported if you need a custom instance:

```ts
import { createClient } from "@shiftapi/client";

const api = createClient({ baseUrl: "https://api.example.com" });
```

## What it auto-configures

- **Vite proxy** -- API paths discovered from your OpenAPI spec are automatically proxied to your Go server during development.
- **tsconfig.json** -- A path mapping for `@shiftapi/client` is added so TypeScript resolves the generated types.
- **HMR** -- When `.go` files change, the plugin restarts the Go server, regenerates types, and triggers a full reload in the browser.

## License

MIT
