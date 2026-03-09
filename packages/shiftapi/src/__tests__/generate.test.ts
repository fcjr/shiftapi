import { describe, it, expect } from "vitest";
import { resolve } from "node:path";
import { extractSpec } from "../extract";
import { generateTypes } from "../generate";
import { virtualModuleTemplate } from "../templates";

const REPO_ROOT = resolve(__dirname, "../../../..");
const GREETER_ENTRY = "./examples/greeter";

describe("generateTypes", () => {
  it("generates TypeScript types from the greeter spec", async () => {
    const spec = await extractSpec(GREETER_ENTRY, REPO_ROOT);
    const types = await generateTypes(spec);

    expect(types).toContain("paths");
    expect(types).toContain("/greet");
    expect(types).toContain("/health");
  });

  it("includes component type definitions", async () => {
    const spec = await extractSpec(GREETER_ENTRY, REPO_ROOT);
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

describe("virtualModuleTemplate", () => {
  it("produces a runtime JS module with client export", () => {
    const source = virtualModuleTemplate("/api");

    expect(source).toContain('import createClient from "openapi-fetch"');
    expect(source).toContain("export const client");
    expect(source).toContain("import.meta.env.VITE_SHIFTAPI_BASE_URL");
    expect(source).toContain("/api");
    expect(source).toContain("export { createClient }");
    // Should NOT contain TypeScript syntax
    expect(source).not.toContain("interface");
    expect(source).not.toContain("type ");
  });
});
