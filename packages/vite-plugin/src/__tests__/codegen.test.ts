import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { mkdtempSync, writeFileSync, readFileSync, rmSync } from "node:fs";
import { join } from "node:path";
import { tmpdir } from "node:os";
import { patchTsConfig } from "../codegen";

describe("patchTsConfig", () => {
  let tempDir: string;

  beforeEach(() => {
    tempDir = mkdtempSync(join(tmpdir(), "shiftapi-test-"));
  });

  afterEach(() => {
    rmSync(tempDir, { recursive: true });
  });

  it("adds path mapping to a plain tsconfig", () => {
    writeFileSync(
      join(tempDir, "tsconfig.json"),
      JSON.stringify({ compilerOptions: { strict: true } }, null, 2),
    );

    patchTsConfig(tempDir);

    const result = JSON.parse(readFileSync(join(tempDir, "tsconfig.json"), "utf-8"));
    expect(result.compilerOptions.paths["@shiftapi/client"]).toEqual([
      "./.shiftapi/client.d.ts",
    ]);
    expect(result.compilerOptions.strict).toBe(true);
  });

  it("adds path mapping when compilerOptions.paths already exists", () => {
    writeFileSync(
      join(tempDir, "tsconfig.json"),
      JSON.stringify(
        { compilerOptions: { paths: { "@/*": ["./src/*"] } } },
        null,
        2,
      ),
    );

    patchTsConfig(tempDir);

    const result = JSON.parse(readFileSync(join(tempDir, "tsconfig.json"), "utf-8"));
    expect(result.compilerOptions.paths["@shiftapi/client"]).toEqual([
      "./.shiftapi/client.d.ts",
    ]);
    expect(result.compilerOptions.paths["@/*"]).toEqual(["./src/*"]);
  });

  it("adds compilerOptions when missing entirely", () => {
    writeFileSync(join(tempDir, "tsconfig.json"), JSON.stringify({ extends: "./base.json" }, null, 2));

    patchTsConfig(tempDir);

    const result = JSON.parse(readFileSync(join(tempDir, "tsconfig.json"), "utf-8"));
    expect(result.compilerOptions.paths["@shiftapi/client"]).toEqual([
      "./.shiftapi/client.d.ts",
    ]);
    expect(result.extends).toBe("./base.json");
  });

  it("does not overwrite if path mapping already exists", () => {
    const original = {
      compilerOptions: {
        paths: { "@shiftapi/client": ["./custom/path.d.ts"] },
      },
    };
    writeFileSync(join(tempDir, "tsconfig.json"), JSON.stringify(original, null, 2));

    patchTsConfig(tempDir);

    const result = JSON.parse(readFileSync(join(tempDir, "tsconfig.json"), "utf-8"));
    expect(result.compilerOptions.paths["@shiftapi/client"]).toEqual([
      "./custom/path.d.ts",
    ]);
  });

  it("preserves comments in JSONC tsconfig", () => {
    const jsonc = `{
  // This is a comment
  "compilerOptions": {
    "strict": true /* inline comment */
  }
}
`;
    writeFileSync(join(tempDir, "tsconfig.json"), jsonc);

    patchTsConfig(tempDir);

    const raw = readFileSync(join(tempDir, "tsconfig.json"), "utf-8");
    expect(raw).toContain("// This is a comment");
    expect(raw).toContain("/* inline comment */");
    expect(raw).toContain("@shiftapi/client");
  });

  it("handles trailing commas in JSONC tsconfig", () => {
    const jsonc = `{
  "compilerOptions": {
    "strict": true,
  },
}
`;
    writeFileSync(join(tempDir, "tsconfig.json"), jsonc);

    patchTsConfig(tempDir);

    const raw = readFileSync(join(tempDir, "tsconfig.json"), "utf-8");
    expect(raw).toContain("@shiftapi/client");
  });

  it("does nothing when tsconfig.json does not exist", () => {
    // Should not throw
    patchTsConfig(tempDir);
  });

  it("warns and skips on unparseable tsconfig", () => {
    writeFileSync(join(tempDir, "tsconfig.json"), "not valid json at all {{{");

    // Should not throw
    patchTsConfig(tempDir);

    const raw = readFileSync(join(tempDir, "tsconfig.json"), "utf-8");
    expect(raw).toBe("not valid json at all {{{");
  });
});
