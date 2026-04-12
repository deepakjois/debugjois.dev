import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import { tanstackRouter } from "@tanstack/router-plugin/vite";
import { readFile } from "node:fs/promises";
import { resolve } from "node:path";
import { fileURLToPath } from "node:url";

const rootDir = fileURLToPath(new URL(".", import.meta.url));

function spaDevFallback() {
  const spaIndex = resolve(rootDir, "apps/spa/index.html");

  function shouldServeSpaEntry(pathname: string) {
    if (
      pathname !== "/apps/spa/" &&
      pathname !== "/apps/spa/index.html" &&
      pathname.startsWith("/apps/spa/") &&
      !/\.[^/]+$/.test(pathname)
    ) {
      return true;
    }

    return false;
  }

  return {
    name: "spa-dev-fallback",
    configureServer(server: import("vite").ViteDevServer) {
      server.middlewares.use((req, res, next) => {
        const rawUrl = req.url;
        if (!rawUrl || (req.method !== "GET" && req.method !== "HEAD")) {
          next();
          return;
        }

        const pathname = rawUrl.split("?", 1)[0];

        if (!shouldServeSpaEntry(pathname)) {
          next();
          return;
        }

        void readFile(spaIndex, "utf8")
          .then((html) => server.transformIndexHtml("/apps/spa/index.html", html, req.originalUrl))
          .then((transformed) => {
            res.statusCode = 200;
            res.setHeader("Content-Type", "text/html");
            if (req.method === "HEAD") {
              res.end();
              return;
            }
            res.end(transformed);
          })
          .catch((error: unknown) => {
            next(error as Error);
          });
      });
    },
  };
}

export default defineConfig({
  root: "./apps",
  base: "/apps/",
  appType: "mpa",
  plugins: [
    spaDevFallback(),
    tanstackRouter({
      target: "react",
      autoCodeSplitting: true,
      routesDirectory: resolve(rootDir, "src/spa/routes"),
      generatedRouteTree: resolve(rootDir, "src/spa/routeTree.gen.ts"),
    }),
    react(),
  ],
  build: {
    outDir: "../../site/build/apps",
    assetsDir: "assets",
    emptyOutDir: true,
    rollupOptions: {
      input: {
        spa: resolve(rootDir, "apps/spa/index.html"),
        transcriptReader: resolve(rootDir, "apps/transcript-reader/index.html"),
      },
    },
  },
  server: {
    fs: {
      allow: [rootDir],
    },
  },
});
