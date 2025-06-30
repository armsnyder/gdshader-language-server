import js from "@eslint/js";
import { defineConfig, globalIgnores } from "eslint/config";
import globals from "globals";

/** @type {import('eslint').Linter.Config[]} */
export default defineConfig([
  globalIgnores([".vscode-test/**", "node_modules/**"]),
  js.configs.recommended,
  {
    files: ["**/*.js"],
    languageOptions: {
      globals: {
        ...globals.commonjs,
        ...globals.node,
        ...globals.mocha,
      },
      ecmaVersion: 2022,
      sourceType: "commonjs",
    },
  },
]);
