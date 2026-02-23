#!/usr/bin/env node
import { resolve } from "node:path";
import { loadConfig } from "./config";
import { extractSpec } from "./extract";
import { generateTypes } from "./generate";
import { writeGeneratedFiles, patchTsConfig } from "./codegen";

const command = process.argv[2];
if (command !== "prepare") {
  console.error(`Usage: shiftapi prepare`);
  process.exit(1);
}

async function main() {
  const cwd = process.cwd();

  console.log("[shiftapi] Preparing...");

  const { config, configDir } = await loadConfig(cwd);
  const serverEntry = config.server;
  const baseUrl = config.baseUrl ?? "/";
  const goRoot = configDir;

  const spec = extractSpec(serverEntry, resolve(goRoot)) as Record<
    string,
    unknown
  >;
  const types = await generateTypes(spec);

  writeGeneratedFiles(configDir, types, baseUrl);
  patchTsConfig(cwd, configDir);

  console.log("[shiftapi] Done. Generated .shiftapi/client.d.ts and .shiftapi/client.js");
}

main().catch((err) => {
  console.error(err instanceof Error ? err.message : String(err));
  process.exit(1);
});
