import path from "node:path";
import { fileURLToPath } from "node:url";
import { vitePlugin as remix } from "@remix-run/dev";
import { defineConfig } from "vite";

const dir = path.dirname(fileURLToPath(import.meta.url));

export default defineConfig({
  plugins: [remix()],
  resolve: {
    alias: {
      "@forst/client": path.resolve(dir, "app/lib/forst-client.ts"),
    },
  },
  ssr: {
    noExternal: ["@forst/sidecar", "@forst/client"],
  },
});
