import { describe, it, expect } from "vitest";
import { resolve } from "node:path";
import { extractSpec } from "../extract";

const REPO_ROOT = resolve(__dirname, "../../../..");
const GREETER_ENTRY = "./examples/greeter";

describe("extractSpec", () => {
  it("extracts a valid OpenAPI spec from the greeter example", () => {
    const spec = extractSpec(GREETER_ENTRY, REPO_ROOT) as Record<string, unknown>;

    expect(spec).toBeDefined();
    expect(spec.openapi).toBe("3.1");

    const info = spec.info as Record<string, unknown>;
    expect(info.title).toBe("Greeter Demo API");

    const paths = spec.paths as Record<string, unknown>;
    expect(paths).toHaveProperty("/greet");
    expect(paths).toHaveProperty("/health");
  });

  it("includes component schemas", () => {
    const spec = extractSpec(GREETER_ENTRY, REPO_ROOT) as Record<string, unknown>;
    const components = spec.components as Record<string, unknown>;
    const schemas = components.schemas as Record<string, unknown>;

    expect(schemas).toHaveProperty("Person");
    expect(schemas).toHaveProperty("Greeting");
    expect(schemas).toHaveProperty("Status");
  });

  it("throws on invalid server entry", () => {
    expect(() => extractSpec("./nonexistent", REPO_ROOT)).toThrow(
      "shiftapi:"
    );
  });
});
