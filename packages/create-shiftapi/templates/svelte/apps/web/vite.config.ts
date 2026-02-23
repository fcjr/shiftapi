import { defineConfig } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";
import shiftapi from "@shiftapi/vite-plugin";

export default defineConfig({
  plugins: [svelte(), shiftapi()],
});
