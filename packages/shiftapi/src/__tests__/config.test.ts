import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { mkdtempSync, writeFileSync, rmSync, mkdirSync } from "node:fs";
import { join } from "node:path";
import { tmpdir } from "node:os";
import { loadConfig, defineConfig } from "../config";

describe("defineConfig", () => {
  it("returns the same object (identity)", () => {
    const config = { server: "./cmd/myapp" };
    expect(defineConfig(config)).toBe(config);
  });
});

describe("loadConfig", () => {
  let tempDir: string;

  beforeEach(() => {
    tempDir = mkdtempSync(join(tmpdir(), "shiftapi-config-test-"));
  });

  afterEach(() => {
    rmSync(tempDir, { recursive: true });
  });

  it("finds shiftapi.config.ts in current directory", async () => {
    writeFileSync(
      join(tempDir, "shiftapi.config.ts"),
      `export default { server: "./cmd/myapp" };`,
    );

    const result = await loadConfig(tempDir);
    expect(result.config.server).toBe("./cmd/myapp");
    expect(result.configDir).toBe(tempDir);
  });

  it("finds shiftapi.config.js in current directory", async () => {
    writeFileSync(
      join(tempDir, "shiftapi.config.js"),
      `export default { server: "./cmd/myapp" };`,
    );

    const result = await loadConfig(tempDir);
    expect(result.config.server).toBe("./cmd/myapp");
    expect(result.configDir).toBe(tempDir);
  });

  it("walks up to find config in parent directory", async () => {
    writeFileSync(
      join(tempDir, "shiftapi.config.ts"),
      `export default { server: "./cmd/myapp", baseUrl: "/api" };`,
    );

    const childDir = join(tempDir, "apps", "web");
    mkdirSync(childDir, { recursive: true });

    const result = await loadConfig(childDir);
    expect(result.config.server).toBe("./cmd/myapp");
    expect(result.config.baseUrl).toBe("/api");
    expect(result.configDir).toBe(tempDir);
  });

  it("throws if no config found", async () => {
    await expect(loadConfig(tempDir)).rejects.toThrow(
      /Could not find shiftapi\.config\.ts/,
    );
  });

  it("throws if server field is missing", async () => {
    writeFileSync(
      join(tempDir, "shiftapi.config.ts"),
      `export default { baseUrl: "/" };`,
    );

    await expect(loadConfig(tempDir)).rejects.toThrow(
      /must specify a "server" field/,
    );
  });

  it("uses explicit configPath when provided", async () => {
    const configPath = join(tempDir, "custom.config.ts");
    writeFileSync(
      configPath,
      `export default { server: "./cmd/custom" };`,
    );

    const result = await loadConfig(tempDir, configPath);
    expect(result.config.server).toBe("./cmd/custom");
    expect(result.configDir).toBe(tempDir);
  });

  it("throws if explicit configPath does not exist", async () => {
    await expect(
      loadConfig(tempDir, join(tempDir, "nonexistent.config.ts")),
    ).rejects.toThrow(/Config file not found/);
  });

  it("loads config with defineConfig wrapper", async () => {
    writeFileSync(
      join(tempDir, "shiftapi.config.ts"),
      `
function defineConfig(c) { return c; }
export default defineConfig({ server: "./cmd/myapp", url: "http://localhost:9090" });
`,
    );

    const result = await loadConfig(tempDir);
    expect(result.config.server).toBe("./cmd/myapp");
    expect(result.config.url).toBe("http://localhost:9090");
  });
});
