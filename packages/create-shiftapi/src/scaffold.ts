import { execFile } from "node:child_process";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

export type Framework = "react" | "svelte";

export interface ScaffoldOptions {
  name: string;
  modulePath: string;
  port: string;
  framework: Framework;
  targetDir: string;
}

const renameFiles: Record<string, string> = {
  _gitignore: ".gitignore",
};

const templatesDir = path.resolve(
  fileURLToPath(import.meta.url),
  "../..",
  "templates",
);

function replaceplaceholders(
  content: string,
  opts: ScaffoldOptions,
): string {
  return content
    .replaceAll("{{name}}", opts.name)
    .replaceAll("{{modulePath}}", opts.modulePath)
    .replaceAll("{{port}}", opts.port);
}

function renamePath(filePath: string, opts: ScaffoldOptions): string {
  return filePath.replaceAll("__name__", opts.name);
}

function copyDir(srcDir: string, destDir: string, opts: ScaffoldOptions) {
  fs.mkdirSync(destDir, { recursive: true });
  for (const entry of fs.readdirSync(srcDir)) {
    const srcPath = path.join(srcDir, entry);
    const destName = renameFiles[entry] ?? entry;
    const destPath = path.join(destDir, renamePath(destName, opts));
    const stat = fs.statSync(srcPath);
    if (stat.isDirectory()) {
      copyDir(srcPath, destPath, opts);
    } else {
      const content = fs.readFileSync(srcPath, "utf-8");
      fs.mkdirSync(path.dirname(destPath), { recursive: true });
      fs.writeFileSync(destPath, replaceplaceholders(content, opts));
    }
  }
}

function gitInit(cwd: string): Promise<void> {
  return new Promise((resolve, reject) => {
    execFile("git", ["init"], { cwd }, (err) => {
      if (err) {
        reject(err);
        return;
      }
      resolve();
    });
  });
}

export async function scaffold(opts: ScaffoldOptions): Promise<void> {
  copyDir(path.join(templatesDir, "base"), opts.targetDir, opts);
  copyDir(path.join(templatesDir, opts.framework), opts.targetDir, opts);
  await gitInit(opts.targetDir);
}

/** Returns the list of file paths that would be generated (for testing). */
export function getFiles(opts: ScaffoldOptions): string[] {
  const files: string[] = [];

  function collect(srcDir: string, destPrefix: string) {
    for (const entry of fs.readdirSync(srcDir)) {
      const srcPath = path.join(srcDir, entry);
      const destName = renameFiles[entry] ?? entry;
      const destPath = path.join(destPrefix, renamePath(destName, opts));
      const stat = fs.statSync(srcPath);
      if (stat.isDirectory()) {
        collect(srcPath, destPath);
      } else {
        files.push(destPath);
      }
    }
  }

  collect(path.join(templatesDir, "base"), "");
  collect(path.join(templatesDir, opts.framework), "");
  return files.map((f) => f.replace(/^\//, ""));
}
