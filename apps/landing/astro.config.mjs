import { defineConfig } from "astro/config";
import starlight from "@astrojs/starlight";
import react from "@astrojs/react";
import tailwindcss from "@tailwindcss/vite";

export default defineConfig({
  integrations: [
    starlight({
      title: "ShiftAPI",
      logo: {
        src: "./src/assets/logo.svg",
        replacesTitle: false,
      },
      customCss: ["./src/docs.css"],
      components: {
        ThemeSelect: "./src/components/ThemeSelect.astro",
      },
      social: [
        { icon: "github", label: "GitHub", href: "https://github.com/fcjr/shiftapi" },
      ],
      sidebar: [
        {
          label: "Getting Started",
          autogenerate: { directory: "docs/getting-started" },
        },
        {
          label: "Core Concepts",
          autogenerate: { directory: "docs/core-concepts" },
        },
        {
          label: "Frontend",
          autogenerate: { directory: "docs/frontend" },
        },
      ],
    }),
    react(),
  ],
  vite: {
    plugins: [tailwindcss()],
  },
});
