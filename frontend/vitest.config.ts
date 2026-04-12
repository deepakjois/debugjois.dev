import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";

export default defineConfig({
  // Intentionally omits tanstackRouter plugin — it generates routeTree.gen.ts
  // and must never run during tests.
  plugins: [react()],
  define: {
    "import.meta.env.VITE_SITE_BACKEND_URL": JSON.stringify("http://localhost:3000"),
    "import.meta.env.VITE_GOOGLE_CLIENT_ID": JSON.stringify("test-google-client-id"),
    // Disable the auth bypass in tests so auth gate tests run normally.
    "import.meta.env.VITE_AUTH_BYPASS": JSON.stringify("false"),
  },
  test: {
    environment: "jsdom",
    globals: true,
    setupFiles: ["./src/spa/test/setup.ts"],
    include: ["src/**/*.{test,spec}.{ts,tsx}"],
    exclude: ["src/spa/routeTree.gen.ts", "node_modules"],
    coverage: {
      provider: "v8",
      reporter: ["text", "lcov"],
      exclude: [
        "src/spa/routeTree.gen.ts",
        "src/spa/test/**",
        "src/spa/__tests__/**",
        "*.config.ts",
      ],
    },
  },
});
