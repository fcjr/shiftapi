import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import shiftapi from "@shiftapi/vite-plugin";

export default defineConfig({
  plugins: [react(), shiftapi()],
});
