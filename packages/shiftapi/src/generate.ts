import openapiTS, {
  astToString,
  stringToAST,
} from "openapi-typescript";
import type { SchemaObject } from "openapi-typescript";

// Build the union type node once from a string to avoid importing typescript directly.
// Importing typescript causes "Dynamic require of fs is not supported" when tsup bundles it into ESM.
const BINARY = (stringToAST("type T = File | Blob | Uint8Array") as any)[0]
  .type as import("typescript").TypeNode;

/**
 * Generates TypeScript type definitions from an OpenAPI spec object
 * using the openapi-typescript programmatic API.
 */
export async function generateTypes(spec: object): Promise<string> {
  const ast = await openapiTS(spec as Parameters<typeof openapiTS>[0], {
    transform(schemaObject: SchemaObject) {
      if (
        "format" in schemaObject &&
        schemaObject.format === "binary" &&
        schemaObject.type === "string"
      ) {
        return BINARY;
      }
    },
  });
  return astToString(ast);
}
