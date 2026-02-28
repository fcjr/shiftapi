import { defineConfig } from "vite";
import shiftapi from "@shiftapi/vite-plugin";

export default defineConfig({
  plugins: [shiftapi()],
});
