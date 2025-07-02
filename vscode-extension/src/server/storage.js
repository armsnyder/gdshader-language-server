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
const { logger } = require("../log");
const { getConfiguration } = require("../config");
const { join } = require("node:path");
const { platform } = require("node:os");
const { rm, stat } = require("node:fs/promises");

/** @param {import('vscode').ExtensionContext} context @returns {string} */
function getBinCacheDir(context) {
  return join(context.globalStorageUri.fsPath, "bin");
}

/** @param {import('vscode').ExtensionContext} context @returns {string} */
function getBinDir(context) {
  return join(getBinCacheDir(context), context.extension.packageJSON.version);
}

/** @param {import('vscode').ExtensionContext} context @returns {Promise<string>} */
async function getBinPath(context) {
  const { serverPathOverride } = getConfiguration();

  if (serverPathOverride) {
    if (!(await fileExists(serverPathOverride))) {
      throw new Error(
        `Server path override does not exist: ${serverPathOverride}`,
      );
    }
    logger().warn(`Using server path override: ${serverPathOverride}`);
    return serverPathOverride;
  }

  return join(
    getBinDir(context),
    `gdshader-language-server${platform() === "win32" ? ".exe" : ""}`,
  );
}

/** @param {import('vscode').ExtensionContext} context @returns {Promise<string>} */
async function getSignaturePath(context) {
  return (await getBinPath(context)) + ".sig";
}

/** @param {import('vscode').ExtensionContext} context @returns {Promise<void>} */
async function cleanUpBin(context) {
  await rm(getBinCacheDir(context), { recursive: true, force: true }).catch(
    (error) => {
      logger().error("Failed to clean up bin directory:", error);
    },
  );
}

/** @param {string} path @returns {Promise<boolean>} */
async function fileExists(path) {
  try {
    return (await stat(path)).isFile();
  } catch (error) {
    if (error instanceof Error && "code" in error && error.code === "ENOENT") {
      return false;
    }
    throw error;
  }
}

module.exports = {
  getBinCacheDir,
  getBinDir,
  getBinPath,
  getSignaturePath,
  cleanUpBin,
  fileExists,
};
