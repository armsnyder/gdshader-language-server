const { logger } = require("../log");
const { join } = require("node:path");
const { platform } = require("node:os");
const { rm, stat } = require("node:fs/promises");

function getBinName() {
  return `gdshader-language-server${platform() === "win32" ? ".exe" : ""}`;
}

/** @param {import('vscode').ExtensionContext} context @returns {string} */
function getBinCacheDir(context) {
  return join(context.globalStorageUri.fsPath, "bin");
}

/** @param {import('vscode').ExtensionContext} context @returns {string} */
function getBinDir(context) {
  return join(getBinCacheDir(context), context.extension.packageJSON.version);
}

/** @param {import('vscode').ExtensionContext} context @returns {string} */
function getBinPath(context) {
  return join(getBinDir(context), getBinName());
}

/** @param {import('vscode').ExtensionContext} context @returns {string} */
function getSignaturePath(context) {
  return getBinPath(context) + ".sig";
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
  getBinName,
  getBinCacheDir,
  getBinDir,
  getBinPath,
  getSignaturePath,
  cleanUpBin,
  fileExists,
};
