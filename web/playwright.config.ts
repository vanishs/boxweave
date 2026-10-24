import { defineConfig } from "@playwright/test";
export default defineConfig({
  testDir: "./browser",
  workers: 1,
  use: {
    baseURL: "http://127.0.0.1:18555",
    headless: true,
    viewport: { width: 1440, height: 1000 },
  },
  webServer: {
    command: "../.venv/bin/python ../tests/web_fixture.py",
    url: "http://127.0.0.1:18555/healthz",
    reuseExistingServer: false,
    timeout: 30000,
  },
  reporter: "list",
});
