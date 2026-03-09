import { defineConfig } from "tsdown";

export default defineConfig({
  entry: [
    "src/index.ts",
    "src/internal.ts",
    "src/prepare.ts",
  ],
});
