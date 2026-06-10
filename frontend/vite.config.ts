import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      "/predictions.json": "http://127.0.0.1:8765",
      "/api": "http://127.0.0.1:8765",
    },
  },
  test: {
    globals: true,
    environment: "node",
  },
});
