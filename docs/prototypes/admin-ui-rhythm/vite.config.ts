import react from "@vitejs/plugin-react";
import { defineConfig, loadEnv } from "vite";

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, ".", "");
  const target = env.COMPASSO_DEV_API_TARGET || env.VITE_COMPASSO_API_BASE_URL;
  return { plugins: [react()], server: target ? { proxy: { "/api": { target, changeOrigin: false } } } : undefined };
});
