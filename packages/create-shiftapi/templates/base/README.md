# {{name}}

Built with [ShiftAPI](https://github.com/fcjr/shiftapi) — Go server + typed TypeScript frontend.

## Getting Started

```bash
go mod tidy
npm install
npm run dev
```

This starts the Go server and Vite dev server together. The frontend gets fully typed API clients generated from your Go handlers.

- API docs: http://localhost:{{port}}/docs
- Frontend: http://localhost:5173

## Project Structure

```
cmd/{{name}}/main.go          # Go entry point
internal/server/server.go      # API routes and handlers
go.mod
.env                           # Environment variables (PORT)
shiftapi.config.ts             # ShiftAPI config (anchors project root)
packages/api/                  # Typed API client (shared)
apps/web/                      # Frontend (Vite + TypeScript)
```

## Scripts

```bash
npm run dev       # Start Go server + Vite dev server
```
