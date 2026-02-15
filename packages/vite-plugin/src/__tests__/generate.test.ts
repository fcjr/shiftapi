import { describe, it, expect } from "vitest";
import { resolve } from "node:path";
import { extractSpec } from "../extract.js";
import { generateTypes } from "../generate.js";
import { buildVirtualModuleSource } from "../virtualModule.js";

const REPO_ROOT = resolve(__dirname, "../../../..");
const GREETER_ENTRY = "./examples/greeter";

describe("generateTypes", () => {
  it("generates TypeScript types from the greeter spec", async () => {
    const spec = extractSpec(GREETER_ENTRY, REPO_ROOT);
    const types = await generateTypes(spec);

    expect(types).toContain("paths");
    expect(types).toContain("/greet");
    expect(types).toContain("/health");
  });

  it("includes component type definitions", async () => {
    const spec = extractSpec(GREETER_ENTRY, REPO_ROOT);
    const types = await generateTypes(spec);

    expect(types).toContain("Person");
    expect(types).toContain("Greeting");
  });
});

describe("buildVirtualModuleSource", () => {
  it("produces a runtime JS module with client export", () => {
    const source = buildVirtualModuleSource("/api");

    expect(source).toContain('import createClient from "openapi-fetch"');
    expect(source).toContain("export const client");
    expect(source).toContain("/api");
    expect(source).toContain("export { createClient }");
    // Should NOT contain TypeScript syntax
    expect(source).not.toContain("interface");
    expect(source).not.toContain("type ");
  });
});
