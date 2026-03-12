import { defineConfig } from "astro/config";
import starlight from "@astrojs/starlight";
import react from "@astrojs/react";
import tailwindcss from "@tailwindcss/vite";

export default defineConfig({
  redirects: {
    "/docs": "/docs/getting-started/introduction",
  },
  integrations: [
    starlight({
      title: "ShiftAPI",
      logo: {
        src: "./src/assets/logo.svg",
        replacesTitle: false,
      },
      expressiveCode: {
        themes: ["starlight-dark"],
      },
      customCss: ["./src/docs.css"],
      components: {
        PageTitle: "./src/components/DocsPageTitle.astro",
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
          items: [
            { slug: "docs/core-concepts/handlers" },
            { slug: "docs/core-concepts/validation" },
            { slug: "docs/core-concepts/error-handling" },
            { slug: "docs/core-concepts/middleware" },
            { slug: "docs/core-concepts/options" },
            { slug: "docs/core-concepts/file-uploads" },
            { slug: "docs/core-concepts/raw-handlers" },
            {
              label: "Server-Sent Events",
              badge: { text: "Experimental", variant: "caution" },
              items: [
                { slug: "docs/core-concepts/server-sent-events/server" },
                { slug: "docs/core-concepts/server-sent-events/client" },
              ],
            },
            {
              label: "WebSockets",
              badge: { text: "Experimental", variant: "caution" },
              items: [
                { slug: "docs/core-concepts/websockets/server" },
                { slug: "docs/core-concepts/websockets/client" },
              ],
            },
          ],
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
