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

## Prompts

| Prompt | Default |
|---|---|
| Project name | `my-app` |
| Framework | React + Vite / Svelte + Vite / Next.js |
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

This starts the Go server and dev server together. The frontend gets fully typed API clients generated from your Go handlers — edit a struct in Go, get instant type errors in TypeScript.

API docs are served at `http://localhost:8080/docs`.
