/**
 * Copyright (c) 2025 Adam Snyder <https://armsnyder.com> and contributors
 * SPDX-License-Identifier: MIT
 */
import js from "@eslint/js";
import { defineConfig, globalIgnores } from "eslint/config";
import headers from "eslint-plugin-headers";
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
    plugins: {
      headers,
    },
    rules: {
      "headers/header-format": [
        "error",
        {
          source: "string",
          content:
            "Copyright (c) 2025 Adam Snyder <https://armsnyder.com> and contributors\n" +
            "SPDX-License-Identifier: MIT",
          path: "../LICENSE",
        },
      ],
    },
  },
]);
