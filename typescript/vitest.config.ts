import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    globals: true,
    coverage: {
      reporter: ["text", "html"],
      include: ["src/**/*.ts"],
      exclude: [
        "src/cli/index.ts",
        "src/authz-service/index.ts",
        "src/documents-service/index.ts",
        "src/authz-service/adapters/http/makeAuthzHttpServer.ts",
        "src/documents-service/adapters/http/makeDocumentsHttpServer.ts"
      ]
    }
  }
});
