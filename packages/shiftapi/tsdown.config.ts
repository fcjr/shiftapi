import { defineConfig } from "tsdown";

export default defineConfig({
  entry: [
    "src/index.ts",
    "src/internal.ts",
    "src/browser.ts",
    "src/prepare.ts",
  ],
});
