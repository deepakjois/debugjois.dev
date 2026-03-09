import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import { tanstackRouter } from "@tanstack/router-plugin/vite";

export default defineConfig({
  base: "/app/",
  plugins: [tanstackRouter({ target: "react", autoCodeSplitting: true }), react()],
  build: {
    outDir: "../site/build/app",
    emptyOutDir: true,
  },
});
