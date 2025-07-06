/**
 * MIT License
 *
 * Copyright (c) 2025 Adam Snyder
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
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
