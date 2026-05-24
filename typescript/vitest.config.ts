import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    globals: true,
    coverage: {
      reporter: ["text", "html"],
      include: ["src/**/*.ts"],
      exclude: [
        "src/cli/index.ts",
        "src/demo/index.ts",
        "src/server/index.ts",
        "src/adapters/http/makeHttpServer.ts"
      ]
    }
  }
});
