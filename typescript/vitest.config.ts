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
        "src/cli/index.ts",
        "src/demo/index.ts",
        "src/server/index.ts",
        "src/modules/http/makeHttpServer.ts"
      ]
    }
  }
});
