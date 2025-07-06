/**
 * Copyright (c) 2025 Adam Snyder <https://armsnyder.com> and contributors
 * SPDX-License-Identifier: MIT
 * Please retain this header in any redistributions of this code.
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
