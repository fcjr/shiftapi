<p align="center">
	<picture>
		<source media="(prefers-color-scheme: dark)" srcset="https://raw.githubusercontent.com/fcjr/shiftapi/main/assets/logo-dark.svg">
		<img src="https://raw.githubusercontent.com/fcjr/shiftapi/main/assets/logo.svg" alt="ShiftAPI Logo">
	</picture>
</p>

# create-shiftapi

Scaffold a new [ShiftAPI](https://github.com/fcjr/shiftapi) fullstack app — Go server + typed TypeScript frontend.

## Usage

```bash
npm create shiftapi@latest
```

Or with pnpm / yarn:

```bash
pnpm create shiftapi@latest
yarn create shiftapi@latest
```

You can also pass the project name directly:

```bash
npm create shiftapi@latest my-app
```

## What You Get

```
my-app/
  cmd/my-app/main.go              # Go entry point with graceful shutdown
  internal/server/server.go        # API routes and handlers
  go.mod
  .env                             # PORT config
  .gitignore
  package.json                     # Monorepo root with workspaces
  apps/web/
    package.json                   # React or Svelte frontend
    vite.config.ts                 # ShiftAPI vite plugin configured
    tsconfig.json
    index.html
    src/
      main.tsx (or .ts)            # App entry
      App.tsx (or .svelte)         # Demo component with typed API calls
```

## Prompts

| Prompt | Default |
|---|---|
| Project name | `my-app` |
| Framework | React / Svelte |
| Directory | `./<project-name>` |
| Go module path | `github.com/<gh-user>/<project-name>` if logged into `gh`, otherwise `<project-name>` |
| Server port | `8080` |

## After Scaffolding

```bash
cd my-app
go mod tidy
npm install
npm run dev
```

This starts the Go server and Vite dev server together. The frontend gets fully typed API clients generated from your Go handlers — edit a struct in Go, get instant type errors in TypeScript.

API docs are served at `http://localhost:8080/docs`.
