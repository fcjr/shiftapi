#!/usr/bin/env node

import fs from "node:fs";
import os from "node:os";
import * as p from "@clack/prompts";
import path from "node:path";
import { scaffold, type Framework } from "./scaffold.js";

function expandHome(filepath: string): string {
  if (filepath === "~" || filepath.startsWith("~/")) {
    return path.join(os.homedir(), filepath.slice(1));
  }
  return filepath;
}

function getGitHubUser(): string | null {
  const configDir = expandHome(
    process.env.GH_CONFIG_DIR ??
    process.env.XDG_CONFIG_HOME
      ? path.join(expandHome(process.env.XDG_CONFIG_HOME!), "gh")
      : path.join(os.homedir(), ".config", "gh"),
  );
  try {
    const hosts = fs.readFileSync(path.join(configDir, "hosts.yml"), "utf-8");
    const match = hosts.match(/github\.com:\s[\s\S]*?^\s+user:\s+(.+)$/m);
    return match?.[1]?.trim() || null;
  } catch {
    return null;
  }
}

async function main() {
  const positionalName = process.argv[2];
  const ghUser = getGitHubUser();

  p.intro("create-shiftapi");

  const project = await p.group(
    {
      name: () =>
        positionalName
          ? Promise.resolve(positionalName)
          : p.text({
              message: "Project name",
              placeholder: "my-app",
              defaultValue: "my-app",
            }),
      framework: () =>
        p.select({
          message: "Framework",
          options: [
            { label: "React", value: "react" as const },
            { label: "Svelte", value: "svelte" as const },
          ],
        }),
      directory: ({ results }) =>
        p.text({
          message: "Directory",
          placeholder: `./${results.name}`,
          defaultValue: `./${results.name}`,
        }),
      module: ({ results }) =>
        p.text({
          message: "Go module path",
          placeholder: ghUser
            ? `github.com/${ghUser}/${results.name}`
            : results.name!,
          defaultValue: ghUser
            ? `github.com/${ghUser}/${results.name}`
            : results.name!,
        }),
      port: () =>
        p.text({
          message: "Server port",
          placeholder: "8080",
          defaultValue: "8080",
        }),
    },
    {
      onCancel: () => {
        p.cancel("Cancelled.");
        process.exit(1);
      },
    },
  );

  const targetDir = path.resolve(process.cwd(), expandHome(project.directory as string));

  if (fs.existsSync(targetDir)) {
    p.cancel(`${targetDir} already exists.`);
    process.exit(1);
  }

  const s = p.spinner();
  s.start("Scaffolding project");

  await scaffold({
    name: project.name as string,
    modulePath: project.module as string,
    port: project.port as string,
    framework: project.framework as Framework,
    targetDir,
  });

  s.stop("Project scaffolded");

  const relDir = path.relative(process.cwd(), targetDir) || ".";
  p.note(
    [`cd ${relDir}`, "go mod tidy", "npm install", "npm run dev"].join("\n"),
    "Next steps",
  );

  p.outro("Happy hacking!");
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
