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

export function writeGeneratedFiles(typesRoot: string, generatedDts: string, baseUrl: string): void {
  const outDir = resolve(typesRoot, ".shiftapi");
  if (!existsSync(outDir)) {
    mkdirSync(outDir, { recursive: true });
  }

  writeFileSync(resolve(outDir, "client.d.ts"), dtsTemplate(generatedDts));
  writeFileSync(resolve(outDir, "client.js"), clientJsTemplate(baseUrl));
  writeFileSync(
    resolve(outDir, "tsconfig.json"),
    JSON.stringify(
      {
        compilerOptions: {
          paths: {
            [MODULE_ID]: ["./client.d.ts"],
          },
        },
      },
      null,
      2,
    ) + "\n",
  );
}

export function patchTsConfig(tsconfigDir: string, typesRoot: string): void {
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

  const rel = relative(tsconfigDir, resolve(typesRoot, ".shiftapi", "tsconfig.json"));
  const extendsPath = rel.startsWith("..") ? rel : `./${rel}`;

  if (tsconfig?.extends === extendsPath) return;

  tsconfig.extends = extendsPath;

  const detectedIndent = raw.match(/^[ \t]+/m)?.[0] ?? "  ";
  writeFileSync(tsconfigPath, stringify(tsconfig, null, detectedIndent) + "\n");
  console.log(
    "[shiftapi] Updated tsconfig.json to extend .shiftapi/tsconfig.json",
  );
}
