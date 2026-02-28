import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { mkdtempSync, writeFileSync, readFileSync, rmSync, existsSync, mkdirSync } from "node:fs";
import { join } from "node:path";
import { tmpdir } from "node:os";
import { writeGeneratedFiles, patchTsConfigPaths } from "../codegen";

describe("writeGeneratedFiles", () => {
  let tempDir: string;

  beforeEach(() => {
    tempDir = mkdtempSync(join(tmpdir(), "shiftapi-test-"));
  });

  afterEach(() => {
    rmSync(tempDir, { recursive: true });
  });

  it("creates .shiftapi directory", () => {
    writeGeneratedFiles(tempDir, "// generated types", "/");

    expect(existsSync(join(tempDir, ".shiftapi"))).toBe(true);
  });

  it("writes client.d.ts", () => {
    writeGeneratedFiles(tempDir, "// generated types", "/");

    const dts = readFileSync(join(tempDir, ".shiftapi", "client.d.ts"), "utf-8");
    expect(dts).toContain("@shiftapi/client");
    expect(dts).toContain("// generated types");
  });

  it("writes tsconfig.json with path mapping", () => {
    writeGeneratedFiles(tempDir, "// generated types", "/");

    const tsconfig = JSON.parse(
      readFileSync(join(tempDir, ".shiftapi", "tsconfig.json"), "utf-8"),
    );
    expect(tsconfig.compilerOptions.paths["@shiftapi/client"]).toEqual([
      "./client",
    ]);
  });

  it("writes client.js with correct baseUrl", () => {
    writeGeneratedFiles(tempDir, "// generated types", "/api/v1");

    const js = readFileSync(join(tempDir, ".shiftapi", "client.js"), "utf-8");
    expect(js).toContain('"/api/v1"');
    expect(js).toContain("openapi-fetch");
    expect(js).toContain("export const client");
    expect(js).not.toContain("import.meta.env");
  });

  it("overwrites existing files", () => {
    writeGeneratedFiles(tempDir, "// first", "/");
    writeGeneratedFiles(tempDir, "// second", "/");

    const dts = readFileSync(join(tempDir, ".shiftapi", "client.d.ts"), "utf-8");
    expect(dts).toContain("// second");
    expect(dts).not.toContain("// first");
  });
});

describe("patchTsConfigPaths", () => {
  let tempDir: string;

  beforeEach(() => {
    tempDir = mkdtempSync(join(tmpdir(), "shiftapi-test-"));
  });

  afterEach(() => {
    rmSync(tempDir, { recursive: true });
  });

  it("adds @shiftapi/client path when typesRoot is same directory", () => {
    writeFileSync(
      join(tempDir, "tsconfig.json"),
      JSON.stringify({ compilerOptions: { strict: true } }, null, 2),
    );

    patchTsConfigPaths(tempDir, tempDir);

    const result = JSON.parse(readFileSync(join(tempDir, "tsconfig.json"), "utf-8"));
    expect(result.compilerOptions.paths["@shiftapi/client"]).toEqual(["./.shiftapi/client"]);
    expect(result.compilerOptions.strict).toBe(true);
  });

  it("computes relative path when typesRoot differs", () => {
    const appDir = join(tempDir, "apps", "web");
    mkdirSync(appDir, { recursive: true });
    writeFileSync(
      join(appDir, "tsconfig.json"),
      JSON.stringify({ compilerOptions: {} }, null, 2),
    );

    patchTsConfigPaths(appDir, tempDir);

    const result = JSON.parse(readFileSync(join(appDir, "tsconfig.json"), "utf-8"));
    expect(result.compilerOptions.paths["@shiftapi/client"]).toEqual(["../../.shiftapi/client"]);
  });

  it("is a no-op if path already set", () => {
    const original = JSON.stringify(
      { compilerOptions: { paths: { "@shiftapi/client": ["./.shiftapi/client"] } } },
      null,
      2,
    );
    writeFileSync(join(tempDir, "tsconfig.json"), original);

    patchTsConfigPaths(tempDir, tempDir);

    expect(readFileSync(join(tempDir, "tsconfig.json"), "utf-8")).toBe(original);
  });

  it("preserves existing paths entries", () => {
    writeFileSync(join(tempDir, "tsconfig.json"), JSON.stringify(
      { compilerOptions: { paths: { "@/*": ["./*"] } } },
      null,
      2,
    ));

    patchTsConfigPaths(tempDir, tempDir);

    const result = JSON.parse(readFileSync(join(tempDir, "tsconfig.json"), "utf-8"));
    expect(result.compilerOptions.paths["@/*"]).toEqual(["./*"]);
    expect(result.compilerOptions.paths["@shiftapi/client"]).toEqual(["./.shiftapi/client"]);
  });

  it("preserves comments in JSONC", () => {
    writeFileSync(join(tempDir, "tsconfig.json"), `{
  // This is a comment
  "compilerOptions": {
    "strict": true /* inline */
  }
}
`);

    patchTsConfigPaths(tempDir, tempDir);

    const raw = readFileSync(join(tempDir, "tsconfig.json"), "utf-8");
    expect(raw).toContain("// This is a comment");
    expect(raw).toContain("/* inline */");
    expect(raw).toContain("@shiftapi/client");
  });

  it("does nothing when tsconfig.json does not exist", () => {
    patchTsConfigPaths(tempDir, tempDir);
  });

  it("warns and skips on unparseable tsconfig", () => {
    writeFileSync(join(tempDir, "tsconfig.json"), "not valid json {{{");

    patchTsConfigPaths(tempDir, tempDir);

    expect(readFileSync(join(tempDir, "tsconfig.json"), "utf-8")).toBe("not valid json {{{");
  });
});
