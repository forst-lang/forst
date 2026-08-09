import path from "node:path";
import { fileURLToPath } from "node:url";
import { vitePlugin as remix } from "@remix-run/dev";
import { defineConfig } from "vite";

const dir = path.dirname(fileURLToPath(import.meta.url));

export default defineConfig({
  plugins: [remix()],
  resolve: {
    alias: {
      // @forst/client: local ESM shim for Remix SSR + CJS interop.
      "@forst/client": path.resolve(dir, "app/lib/forst-client.ts"),
      // @forst/gen: resolve through node_modules/@forst/gen (forst generate link + exports map).
      // Do not alias to .forst/client — subpaths like @forst/gen/main map to dist/pkg/main.js.
    },
  },
  ssr: {
    // Bundle generated client + sidecar into the SSR graph; no resolve.alias needed for @forst/gen.
    noExternal: ["@forst/sidecar", "@forst/client", "@forst/gen"],
  },
});
