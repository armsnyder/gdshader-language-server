/**
 * Copyright (c) 2025 Adam Snyder <https://armsnyder.com> and contributors
 * SPDX-License-Identifier: MIT
 */
import { defineConfig } from "@vscode/test-cli";
import fs from "fs";
import url from "url";
import path from "path";

function detectMinimumVSCodeVersion() {
  const __filename = url.fileURLToPath(import.meta.url);
  const __dirname = path.dirname(__filename);
  const packageJsonPath = path.join(__dirname, "package.json");
  const packageJson = JSON.parse(fs.readFileSync(packageJsonPath, "utf8"));
  const vscodeVersion = packageJson.engines.vscode.replace(/^\^/, "");
  return vscodeVersion;
}

export default defineConfig([
  {
    version: "stable",
    files: "src/**/*.test.js",
  },
  {
    version: detectMinimumVSCodeVersion(),
    files: "src/**/*.test.js",
  },
]);
