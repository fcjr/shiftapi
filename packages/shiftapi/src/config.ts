import { existsSync } from "node:fs";
import { resolve, dirname } from "node:path";
import { createRequire } from "node:module";

export interface ShiftAPIConfig {
  /** Path to the Go server entry point, relative to the config file (e.g., "./cmd/server") */
  server: string;

  /**
   * Fallback base URL for the API client (default: "/").
   * Can be overridden at build time via the `VITE_SHIFTAPI_BASE_URL` env var.
   */
  baseUrl?: string;

  /**
   * Address the Go server listens on (default: "http://localhost:8080").
   * Used to auto-configure the Vite proxy in dev mode.
   */
  url?: string;
}

/** Identity helper for IntelliSense in `shiftapi.config.ts` files. */
export function defineConfig(config: ShiftAPIConfig): ShiftAPIConfig {
  return config;
}

const CONFIG_FILENAMES = [
  "shiftapi.config.ts",
  "shiftapi.config.js",
  "shiftapi.config.mjs",
];

/**
 * Walk up from `startDir` looking for a shiftapi config file.
 * Returns the parsed config and the directory containing it.
 */
export async function loadConfig(
  startDir: string,
  configPath?: string,
): Promise<{ config: ShiftAPIConfig; configDir: string }> {
  let resolvedPath: string | undefined;

  if (configPath) {
    resolvedPath = resolve(startDir, configPath);
    if (!existsSync(resolvedPath)) {
      throw new Error(`[shiftapi] Config file not found: ${resolvedPath}`);
    }
  } else {
    resolvedPath = findConfigFile(startDir);
    if (!resolvedPath) {
      throw new Error(
        `[shiftapi] Could not find shiftapi.config.ts (searched upward from ${startDir}). ` +
          `Create one with: import { defineConfig } from "shiftapi"`,
      );
    }
  }

  const require = createRequire(import.meta.url);
  const { createJiti } = require("jiti") as typeof import("jiti");
  const jiti = createJiti(resolvedPath);
  const mod = await jiti.import(resolvedPath);
  const config = (mod as { default?: ShiftAPIConfig }).default ?? (mod as ShiftAPIConfig);

  if (!config.server) {
    throw new Error(
      `[shiftapi] Config at ${resolvedPath} must specify a "server" field (path to Go entry point)`,
    );
  }

  return { config, configDir: dirname(resolvedPath) };
}

export interface ShiftAPIPluginOptions {
  /** Explicit path to shiftapi.config.ts (for advanced use only) */
  configPath?: string;
}

/**
 * Synchronously walk up from `startDir` to find the directory containing
 * a shiftapi config file. Returns the directory path, or undefined.
 */
export function findConfigDir(startDir: string, configPath?: string): string | undefined {
  if (configPath) {
    const resolved = resolve(startDir, configPath);
    return existsSync(resolved) ? dirname(resolved) : undefined;
  }

  const file = findConfigFile(startDir);
  return file ? dirname(file) : undefined;
}

function findConfigFile(startDir: string): string | undefined {
  let dir = resolve(startDir);
  const root = resolve("/");

  while (true) {
    for (const name of CONFIG_FILENAMES) {
      const candidate = resolve(dir, name);
      if (existsSync(candidate)) {
        return candidate;
      }
    }
    const parent = dirname(dir);
    if (parent === dir || dir === root) {
      return undefined;
    }
    dir = parent;
  }
}
