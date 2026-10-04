/// <reference types="vitest/config" />
import path from "node:path";
import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: { "@": path.resolve(import.meta.dirname, "./src") },
  },
  server: {
    port: 5173,
    // Same-origin in dev too: the browser talks to Vite, Vite forwards /api to Go.
    proxy: { "/api": "http://localhost:8080" },
  },
  test: {
    environment: "node",
  },
});
