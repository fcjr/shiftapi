import openapiTS, { astToString } from "openapi-typescript";

/**
 * Generates TypeScript type definitions from an OpenAPI spec object
 * using the openapi-typescript programmatic API.
 */
export async function generateTypes(spec: object): Promise<string> {
  const ast = await openapiTS(spec as Parameters<typeof openapiTS>[0]);
  return astToString(ast);
}
