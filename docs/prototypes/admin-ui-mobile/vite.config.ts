import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import { loadEnv } from "vite";
import { defineConfig } from "vitest/config";

export default defineConfig(({ mode }) => {
  const environment = loadEnv(mode, ".", "");
  const apiTarget = environment.COMPASSO_DEV_API_TARGET || environment.VITE_COMPASSO_API_BASE_URL;
  return {
    plugins: [react(), tailwindcss()],
    server: apiTarget ? {
      proxy: { "/api": { target: apiTarget, changeOrigin: false } },
    } : undefined,
    test: {
      environment: "jsdom",
      globals: true,
      setupFiles: "./src/test/setup.ts",
    },
  };
});
