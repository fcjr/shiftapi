import { describe, it, expect } from "vitest";
import { resolve } from "node:path";
import { readFileSync, existsSync } from "node:fs";
import { extractSpec } from "../extract";
import { generateTypes } from "../generate";
import { virtualModuleTemplate } from "../templates";

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

  it("transforms format: binary to File | Blob | Uint8Array", async () => {
    const spec = {
      openapi: "3.1",
      info: { title: "Test", version: "1.0" },
      paths: {
        "/upload": {
          post: {
            operationId: "postUpload",
            requestBody: {
              required: true,
              content: {
                "multipart/form-data": {
                  schema: {
                    type: "object",
                    properties: {
                      file: { type: "string", format: "binary" },
                      title: { type: "string" },
                    },
                    required: ["file"],
                  },
                },
              },
            },
            responses: {
              "200": {
                description: "OK",
                content: {
                  "application/json": {
                    schema: {
                      type: "object",
                      properties: { filename: { type: "string" } },
                    },
                  },
                },
              },
            },
          },
        },
      },
    };
    const types = await generateTypes(spec);

    // The file field should be File | Blob | Uint8Array, not string
    expect(types).toContain("File | Blob | Uint8Array");
    expect(types).not.toMatch(/file\??\s*:\s*string/);
    // title should still be string
    expect(types).toContain("title");
  });

  it("transforms binary array items to File | Blob | Uint8Array", async () => {
    const spec = {
      openapi: "3.1",
      info: { title: "Test", version: "1.0" },
      paths: {
        "/upload-multi": {
          post: {
            operationId: "postUploadMulti",
            requestBody: {
              required: true,
              content: {
                "multipart/form-data": {
                  schema: {
                    type: "object",
                    properties: {
                      files: {
                        type: "array",
                        items: { type: "string", format: "binary" },
                      },
                    },
                  },
                },
              },
            },
            responses: {
              "200": { description: "OK" },
            },
          },
        },
      },
    };
    const types = await generateTypes(spec);

    // Array items should be File | Blob | Uint8Array
    expect(types).toContain("File | Blob | Uint8Array");
  });
});

describe("browser entry point", () => {
  const distDir = resolve(__dirname, "../../dist");

  /**
   * Recursively collect all local chunk imports starting from an entry file.
   */
  function collectChunks(entryFile: string): string[] {
    const seen = new Set<string>();
    const queue = [entryFile];
    while (queue.length > 0) {
      const file = queue.pop()!;
      if (seen.has(file)) continue;
      seen.add(file);
      if (!existsSync(file)) continue;
      const content = readFileSync(file, "utf-8");
      const importRe = /from\s+"\.\/([^"]+)"/g;
      let m;
      while ((m = importRe.exec(content)) !== null) {
        queue.push(resolve(distDir, m[1]));
      }
    }
    return [...seen];
  }

  it("does not depend on Node.js built-ins", () => {
    const entry = resolve(distDir, "browser.mjs");
    if (!existsSync(entry)) {
      // Skip if dist hasn't been built (CI may run tests before build).
      return;
    }
    const files = collectChunks(entry);
    for (const file of files) {
      const content = readFileSync(file, "utf-8");
      const nodeImports = content.match(/from\s+["']node:[^"']+["']/g);
      if (nodeImports) {
        throw new Error(
          `${file} imports Node.js built-ins (would break in browser):\n  ${nodeImports.join("\n  ")}`,
        );
      }
    }
  });
});

describe("virtualModuleTemplate", () => {
  it("produces a runtime JS module with client export", () => {
    const source = virtualModuleTemplate("/api");

    expect(source).toContain('import createClient from "openapi-fetch"');
    expect(source).toContain("export const client");
    expect(source).toContain("import.meta.env.VITE_SHIFTAPI_BASE_URL");
    expect(source).toContain("/api");
    expect(source).toContain("export { createClient }");
    expect(source).toContain('import { createSSE, createWebSocket } from "shiftapi/internal/browser"');
    expect(source).toContain("export const sse = createSSE(");
    expect(source).toContain("export const websocket = createWebSocket(");
    // Should NOT contain TypeScript syntax
    expect(source).not.toContain("interface");
    expect(source).not.toContain("type ");
  });
});
