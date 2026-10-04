import { defineConfig, devices } from "@playwright/test";
import { ADMIN_STATE } from "./e2e/helpers";

// Runs against the production build: Go serves frontend/dist on E2E_PORT with a throwaway database.
// `task e2e` resets the app_e2e database and builds the SPA first.
const PORT = Number(process.env.E2E_PORT ?? 8090);
const DB_PORT = process.env.DB_PORT ?? "5432";

export default defineConfig({
  testDir: "./e2e",
  fullyParallel: false,
  workers: 1,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  reporter: [["list"], ["html", { open: "never" }]],
  use: {
    baseURL: `http://localhost:${PORT}`,
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
  },
  projects: [
    { name: "setup", testMatch: /.*\.setup\.ts/ },
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"], storageState: ADMIN_STATE },
      dependencies: ["setup"],
    },
  ],
  webServer: {
    command: "go run ./cmd/server",
    cwd: "../backend",
    url: `http://localhost:${PORT}/healthz`,
    reuseExistingServer: false,
    timeout: 180_000,
    env: {
      PORT: String(PORT),
      APP_ENV: "development",
      ALLOW_SIGNUP: "true",
      STATIC_DIR: "../frontend/dist",
      JWT_SECRET: "e2e-only-secret-e2e-only-secret-0123",
      DATABASE_URL: process.env.E2E_DATABASE_URL ?? `postgres://app:app@localhost:${DB_PORT}/app_e2e?sslmode=disable`,
    },
  },
});
