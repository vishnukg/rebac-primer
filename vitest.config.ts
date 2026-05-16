import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    globals: true,
    coverage: {
      reporter: ["text", "html"],
      include: ["src/**/*.ts"],
      exclude: [
        "src/main.ts",
        "src/server.ts",
        "src/client/tui.ts",
        "src/http/server.ts"
      ]
    }
  }
});
