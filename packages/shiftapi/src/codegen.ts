import { resolve, relative } from "node:path";
import { writeFileSync, readFileSync, mkdirSync, existsSync } from "node:fs";
import { parse, stringify } from "comment-json";
import { extractSpec } from "./extract";
import { generateTypes } from "./generate";
import { MODULE_ID, DEV_API_PREFIX } from "./constants";
import { dtsTemplate, clientJsTemplate, virtualModuleTemplate } from "./templates";

export async function regenerateTypes(
  serverEntry: string,
  goRoot: string,
  baseUrl: string,
  isDev: boolean,
  previousTypes: string,
): Promise<{ types: string; virtualModuleSource: string; changed: boolean }> {
  const spec = extractSpec(serverEntry, resolve(goRoot)) as Record<
    string,
    unknown
  >;
  const types = await generateTypes(spec);
  const changed = types !== previousTypes;
  const virtualModuleSource = virtualModuleTemplate(
    baseUrl,
    isDev ? DEV_API_PREFIX : undefined,
  );
  return { types, virtualModuleSource, changed };
}

export function writeGeneratedFiles(
  typesRoot: string,
  generatedDts: string,
  baseUrl: string,
  options?: {
    clientJsContent?: string;
    openapiSource?: string;
  },
): void {
  const outDir = resolve(typesRoot, ".shiftapi");
  if (!existsSync(outDir)) {
    mkdirSync(outDir, { recursive: true });
  }

  writeFileSync(resolve(outDir, "client.d.ts"), dtsTemplate(generatedDts));
  writeFileSync(resolve(outDir, "client.js"), options?.clientJsContent ?? clientJsTemplate(baseUrl));
  if (options?.openapiSource) {
    writeFileSync(resolve(outDir, "openapi-fetch.js"), options.openapiSource);
  }
  writeFileSync(
    resolve(outDir, "tsconfig.json"),
    JSON.stringify(
      {
        compilerOptions: {
          paths: {
            [MODULE_ID]: ["./client"],
          },
        },
      },
      null,
      2,
    ) + "\n",
  );
}

export function patchTsConfigPaths(tsconfigDir: string, typesRoot: string): void {
  const tsconfigPath = resolve(tsconfigDir, "tsconfig.json");
  if (!existsSync(tsconfigPath)) return;

  const raw = readFileSync(tsconfigPath, "utf-8");
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  let tsconfig: any;
  try {
    tsconfig = parse(raw);
  } catch (err) {
    console.warn(
      `[shiftapi] Failed to parse tsconfig.json: ${err instanceof Error ? err.message : String(err)}`,
    );
    return;
  }

  // Use extensionless path so TypeScript finds client.d.ts for types
  // and bundlers (Turbopack) find client.js for runtime.
  const clientRel = relative(tsconfigDir, resolve(typesRoot, ".shiftapi", "client"));
  const clientPath = clientRel.startsWith("..") ? clientRel : `./${clientRel}`;

  tsconfig.compilerOptions = tsconfig.compilerOptions || {};
  tsconfig.compilerOptions.paths = tsconfig.compilerOptions.paths || {};

  const existing = tsconfig.compilerOptions.paths[MODULE_ID];
  if (Array.isArray(existing) && existing.length === 1 && existing[0] === clientPath) {
    return;
  }

  tsconfig.compilerOptions.paths[MODULE_ID] = [clientPath];

  const detectedIndent = raw.match(/^[ \t]+/m)?.[0] ?? "  ";
  writeFileSync(tsconfigPath, stringify(tsconfig, null, detectedIndent) + "\n");
  console.log(
    "[shiftapi] Updated tsconfig.json with @shiftapi/client path mapping.",
  );
}
