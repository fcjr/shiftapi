import { describe, it, expect, beforeEach, afterEach } from "vitest";
import fs from "node:fs/promises";
import path from "node:path";
import os from "node:os";
import { scaffold, getFiles } from "../scaffold.js";

const defaultOpts = {
  name: "test-app",
  modulePath: "github.com/user/test-app",
  port: "3000",
  framework: "react" as const,
  targetDir: "",
};

describe("getFiles", () => {
  it("returns all expected files for react", () => {
    const files = getFiles(defaultOpts);

    expect(files).toContain("cmd/test-app/main.go");
    expect(files).toContain("internal/server/server.go");
    expect(files).toContain("go.mod");
    expect(files).toContain(".env");
    expect(files).toContain(".gitignore");
    expect(files).toContain("package.json");
    expect(files).toContain("shiftapi.config.ts");
    expect(files).toContain("apps/web/package.json");
    expect(files).toContain("apps/web/vite.config.ts");
    expect(files).toContain("apps/web/tsconfig.json");
    expect(files).toContain("apps/web/index.html");
    expect(files).toContain("apps/web/src/main.tsx");
    expect(files).toContain("apps/web/src/App.tsx");
    expect(files).toContain("packages/api/package.json");
    expect(files).toContain("packages/api/tsconfig.json");
    expect(files).toContain("packages/api/src/index.ts");
    expect(files).toContain("README.md");
    expect(files).toHaveLength(17);
  });

  it("returns all expected files for next", () => {
    const files = getFiles({ ...defaultOpts, framework: "next" });

    expect(files).toContain("apps/web/package.json");
    expect(files).toContain("apps/web/next.config.ts");
    expect(files).toContain("apps/web/tsconfig.json");
    expect(files).toContain("apps/web/app/layout.tsx");
    expect(files).toContain("apps/web/app/page.tsx");
    expect(files).toContain("apps/web/app/providers.tsx");
    expect(files).toContain("apps/web/app/api.ts");
    expect(files).toContain("shiftapi.config.ts");
    expect(files).toHaveLength(15);
  });

  it("returns all expected files for svelte", () => {
    const files = getFiles({ ...defaultOpts, framework: "svelte" });

    expect(files).toContain("apps/web/src/main.ts");
    expect(files).toContain("apps/web/src/App.svelte");
    expect(files).toContain("apps/web/src/Home.svelte");
    expect(files).toContain("packages/api/package.json");
    expect(files).toContain("packages/api/src/index.ts");
    expect(files).toContain("shiftapi.config.ts");
    expect(files).toHaveLength(18);
  });
});

describe("scaffold", () => {
  let tmpDir: string;

  beforeEach(async () => {
    tmpDir = await fs.mkdtemp(path.join(os.tmpdir(), "create-shiftapi-test-"));
  });

  afterEach(async () => {
    await fs.rm(tmpDir, { recursive: true, force: true });
  });

  it("creates all files on disk (react)", async () => {
    const targetDir = path.join(tmpDir, "my-app");
    await scaffold({ ...defaultOpts, targetDir });

    const exists = async (p: string) => {
      try {
        await fs.access(path.join(targetDir, p));
        return true;
      } catch {
        return false;
      }
    };

    expect(await exists("cmd/test-app/main.go")).toBe(true);
    expect(await exists("internal/server/server.go")).toBe(true);
    expect(await exists("go.mod")).toBe(true);
    expect(await exists(".env")).toBe(true);
    expect(await exists(".gitignore")).toBe(true);
    expect(await exists("package.json")).toBe(true);
    expect(await exists("shiftapi.config.ts")).toBe(true);
    expect(await exists("apps/web/package.json")).toBe(true);
    expect(await exists("apps/web/src/main.tsx")).toBe(true);
    expect(await exists("apps/web/src/App.tsx")).toBe(true);
    expect(await exists("packages/api/package.json")).toBe(true);
    expect(await exists("packages/api/src/index.ts")).toBe(true);
    expect(await exists(".git")).toBe(true);
  });

  it("creates next files on disk", async () => {
    const targetDir = path.join(tmpDir, "my-app");
    await scaffold({ ...defaultOpts, framework: "next", targetDir });

    const exists = async (p: string) => {
      try {
        await fs.access(path.join(targetDir, p));
        return true;
      } catch {
        return false;
      }
    };

    expect(await exists("apps/web/next.config.ts")).toBe(true);
    expect(await exists("apps/web/app/layout.tsx")).toBe(true);
    expect(await exists("apps/web/app/page.tsx")).toBe(true);
    expect(await exists("apps/web/app/providers.tsx")).toBe(true);
    expect(await exists("apps/web/app/api.ts")).toBe(true);
  });

  it("creates svelte files on disk", async () => {
    const targetDir = path.join(tmpDir, "my-app");
    await scaffold({ ...defaultOpts, framework: "svelte", targetDir });

    const exists = async (p: string) => {
      try {
        await fs.access(path.join(targetDir, p));
        return true;
      } catch {
        return false;
      }
    };

    expect(await exists("apps/web/src/main.ts")).toBe(true);
    expect(await exists("apps/web/src/App.svelte")).toBe(true);
    expect(await exists("apps/web/src/Home.svelte")).toBe(true);
    expect(await exists("packages/api/package.json")).toBe(true);
    expect(await exists("packages/api/src/index.ts")).toBe(true);
  });

  it("initializes a git repository", async () => {
    const targetDir = path.join(tmpDir, "my-app");
    await scaffold({ ...defaultOpts, targetDir });

    const stat = await fs.stat(path.join(targetDir, ".git"));
    expect(stat.isDirectory()).toBe(true);
  });

  it("replaces placeholders in files", async () => {
    const targetDir = path.join(tmpDir, "my-app");
    await scaffold({
      ...defaultOpts,
      name: "cool-project",
      modulePath: "github.com/user/cool-project",
      port: "4000",
      targetDir,
    });

    const mainGo = await fs.readFile(
      path.join(targetDir, "cmd/cool-project/main.go"),
      "utf-8",
    );
    expect(mainGo).toContain(`port = "4000"`);
    expect(mainGo).toContain(
      `"github.com/user/cool-project/internal/server"`,
    );
    expect(mainGo).not.toContain("{{");

    const serverGo = await fs.readFile(
      path.join(targetDir, "internal/server/server.go"),
      "utf-8",
    );
    expect(serverGo).toContain(`Title: "cool-project"`);

    const goMod = await fs.readFile(
      path.join(targetDir, "go.mod"),
      "utf-8",
    );
    expect(goMod).toContain("module github.com/user/cool-project");

    const html = await fs.readFile(
      path.join(targetDir, "apps/web/index.html"),
      "utf-8",
    );
    expect(html).toContain("<title>cool-project</title>");

    const rootPkg = JSON.parse(
      await fs.readFile(path.join(targetDir, "package.json"), "utf-8"),
    );
    expect(rootPkg.name).toBe("cool-project");
    expect(rootPkg.private).toBe(true);
    expect(rootPkg.workspaces).toContain("apps/*");
    expect(rootPkg.workspaces).toContain("packages/*");

    const shiftapiConfig = await fs.readFile(
      path.join(targetDir, "shiftapi.config.ts"),
      "utf-8",
    );
    expect(shiftapiConfig).toContain("./cmd/cool-project");

    const webPkg = JSON.parse(
      await fs.readFile(
        path.join(targetDir, "apps/web/package.json"),
        "utf-8",
      ),
    );
    expect(webPkg.dependencies["react"]).toBeDefined();
    expect(webPkg.devDependencies["@vitejs/plugin-react"]).toBeDefined();

    const viteConfig = await fs.readFile(
      path.join(targetDir, "apps/web/vite.config.ts"),
      "utf-8",
    );
    expect(viteConfig).toContain("shiftapi()");
  });

  it("renames _gitignore to .gitignore", async () => {
    const targetDir = path.join(tmpDir, "my-app");
    await scaffold({ ...defaultOpts, targetDir });

    const exists = async (p: string) => {
      try {
        await fs.access(path.join(targetDir, p));
        return true;
      } catch {
        return false;
      }
    };

    expect(await exists(".gitignore")).toBe(true);
    expect(await exists("_gitignore")).toBe(false);
  });
});
