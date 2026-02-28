<p align="center">
	<picture>
		<source media="(prefers-color-scheme: dark)" srcset="https://raw.githubusercontent.com/fcjr/shiftapi/main/assets/logo-dark.svg">
		<img src="https://raw.githubusercontent.com/fcjr/shiftapi/main/assets/logo.svg" alt="ShiftAPI Logo">
	</picture>
</p>

# @shiftapi/next

Next.js integration that generates fully-typed TypeScript clients from [ShiftAPI](https://github.com/fcjr/shiftapi) Go servers. Get end-to-end type safety between your Go API and your Next.js app with zero manual type definitions.

## How it works

1. Reads your `shiftapi.config.ts` (powered by [`shiftapi`](https://www.npmjs.com/package/shiftapi))
2. Extracts the OpenAPI 3.1 spec from your Go server at build time
3. Generates TypeScript types using `openapi-typescript`
4. Writes a typed `@shiftapi/client` module with a pre-configured [openapi-fetch](https://github.com/openapi-ts/openapi-typescript/tree/main/packages/openapi-fetch) client
5. In dev mode, watches `.go` files and regenerates types on changes
6. Auto-configures Next.js rewrites to proxy API requests to your Go server

## Installation

```bash
npm install -D shiftapi @shiftapi/next
# or
pnpm add -D shiftapi @shiftapi/next
```

**Peer dependency:** `next` (v14+).

## Setup

### shiftapi.config.ts

Create a `shiftapi.config.ts` in your project root:

```ts
import { defineConfig } from "shiftapi";

export default defineConfig({
  server: "./cmd/server",
});
```

### next.config.ts

```ts
import type { NextConfig } from "next";
import { withShiftAPI } from "@shiftapi/next";

const nextConfig: NextConfig = {};

export default withShiftAPI(nextConfig);
```

### Config options

Options are set in `shiftapi.config.ts` (see [`shiftapi`](https://www.npmjs.com/package/shiftapi)):

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `server` | `string` | **(required)** | Path to the Go server entry point (e.g. `"./cmd/server"`) |
| `baseUrl` | `string` | `"/"` | Fallback base URL for the API client. Can be overridden via `NEXT_PUBLIC_SHIFTAPI_BASE_URL`. |
| `url` | `string` | `"http://localhost:8080"` | Address the Go server listens on. Used to auto-configure the dev proxy. |

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

- **Next.js rewrites** -- API requests are proxied to your Go server during development via `beforeFiles` rewrites.
- **tsconfig.json** -- A path mapping for `@shiftapi/client` is added so TypeScript resolves the generated types.
- **Go file watcher** -- When `.go` files change, the plugin restarts the Go server and regenerates types.
- **SSR support** -- The generated client handles both server-side and client-side rendering, using the correct base URL in each environment.

## License

MIT
