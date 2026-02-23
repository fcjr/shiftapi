import { resolve } from "node:path";
import { writeFileSync, readFileSync, mkdirSync, existsSync } from "node:fs";
import { extractSpec } from "./extract";
import { generateTypes } from "./generate";
import { MODULE_ID, DEV_API_PREFIX } from "./constants";
import { dtsTemplate, virtualModuleTemplate } from "./templates";

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

export function writeDtsFile(projectRoot: string, generatedDts: string): void {
  const outDir = resolve(projectRoot, ".shiftapi");
  if (!existsSync(outDir)) {
    mkdirSync(outDir, { recursive: true });
  }

  const dtsContent = dtsTemplate(generatedDts);
  writeFileSync(resolve(outDir, "client.d.ts"), dtsContent);
}

export function patchTsConfig(projectRoot: string): void {
  const tsconfigPath = resolve(projectRoot, "tsconfig.json");
  if (!existsSync(tsconfigPath)) return;

  const raw = readFileSync(tsconfigPath, "utf-8");
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  let tsconfig: any;
  try {
    tsconfig = JSON.parse(raw);
  } catch (err) {
    console.warn(
      `[shiftapi] Failed to parse tsconfig.json: ${err instanceof Error ? err.message : String(err)}`,
    );
    return;
  }

  if (tsconfig?.compilerOptions?.paths?.[MODULE_ID]) return;

  if (!tsconfig.compilerOptions) tsconfig.compilerOptions = {};
  if (!tsconfig.compilerOptions.paths) tsconfig.compilerOptions.paths = {};
  tsconfig.compilerOptions.paths[MODULE_ID] = ["./.shiftapi/client.d.ts"];

  const detectedIndent = raw.match(/^[ \t]+/m)?.[0] ?? "  ";
  writeFileSync(tsconfigPath, JSON.stringify(tsconfig, null, detectedIndent) + "\n");
  console.log(
    "[shiftapi] Updated tsconfig.json with @shiftapi/client path mapping",
  );
}
