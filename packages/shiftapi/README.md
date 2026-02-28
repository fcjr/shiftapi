<p align="center">
	<picture>
		<source media="(prefers-color-scheme: dark)" srcset="https://raw.githubusercontent.com/fcjr/shiftapi/main/assets/logo-dark.svg">
		<img src="https://raw.githubusercontent.com/fcjr/shiftapi/main/assets/logo.svg" alt="ShiftAPI Logo">
	</picture>
</p>

# shiftapi

CLI and codegen core for [ShiftAPI](https://github.com/fcjr/shiftapi). Extracts the OpenAPI spec from your Go server and generates a fully-typed TypeScript client — no Vite required.

## Installation

```bash
npm install -D shiftapi
# or
pnpm add -D shiftapi
```

## Setup

Create a `shiftapi.config.ts` in your project root:

```ts
import { defineConfig } from "shiftapi";

export default defineConfig({
  server: "./cmd/server",
});
```

## CLI Usage

```bash
shiftapi prepare
```

This will:

1. Find your `shiftapi.config.ts` (searching upward from cwd)
2. Run your Go server to extract the OpenAPI 3.1 spec
3. Generate TypeScript types via `openapi-typescript`
4. Write `.shiftapi/client.d.ts` and `.shiftapi/client.js`
5. Patch `tsconfig.json` with the required path mapping

Typically used in a `postinstall` script:

```json
{
  "scripts": {
    "postinstall": "shiftapi prepare"
  }
}
```

## Config Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `server` | `string` | **(required)** | Path to the Go server entry point (e.g. `"./cmd/server"`) |
| `baseUrl` | `string` | `"/"` | Base URL for the generated API client |
| `url` | `string` | `"http://localhost:8080"` | Go server address (used by the Vite/Next.js plugins for dev proxy) |

## License

MIT
