/**
 * Copyright (c) 2025 Adam Snyder <https://armsnyder.com> and contributors
 * SPDX-License-Identifier: MIT
 */
const { logger } = require("../log");
const { EventEmitter } = require("node:events");
const {
  shouldValidateBinarySignature,
  validateBinarySignature,
} = require("./validate");
const {
  getBinPath,
  cleanUpBin,
  fileExists,
  getSignaturePath,
} = require("./storage");
const { downloadAndExtractServerArchive } = require("./download");

/**
 * @typedef {Object} LoadServerEvent
 * @property {[string]} ready
 * @property {[unknown]} error
 * @property {[void]} downloadStarted
 */

/** @param {import('vscode').ExtensionContext} context @returns {Promise<boolean>} */
async function hasRequiredFiles(context) {
  if (!(await fileExists(await getBinPath(context)))) {
    return false;
  }
  if (shouldValidateBinarySignature()) {
    return await fileExists(await getSignaturePath(context));
  }
  return true;
}

/**
 * @param {import('vscode').ExtensionContext} context
 * @param {EventEmitter<LoadServerEvent>} emitter
 * @returns {Promise<string>}
 */
async function loadServerAsync(context, emitter) {
  logger().info("Checking for required files...");

  if (!(await hasRequiredFiles(context))) {
    logger().info("Required files not found. Downloading server binary...");
    await cleanUpBin(context); // Clean out any previous downloads
    emitter.emit("downloadStarted");
    await downloadAndExtractServerArchive(context).catch((error) => {
      throw new Error(`Failed to download server binary: ${error.message}`);
    });
  }

  if (shouldValidateBinarySignature()) {
    logger().info("Validating server binary signature...");
    await validateBinarySignature(context); // Always validate the binary before running
  }

  const binPath = getBinPath(context);
  logger().info(`Server binary is ready at: ${binPath}`);
  return binPath;
}

/** @param {import('vscode').ExtensionContext} context @returns {EventEmitter<LoadServerEvent>} */
function loadServer(context) {
  /** @type {EventEmitter<LoadServerEvent>} */
  const emitter = new EventEmitter();

  process.nextTick(() => {
    try {
      loadServerAsync(context, emitter)
        .then((path) => {
          emitter.emit("ready", path);
        })
        .catch((error) => {
          emitter.emit("error", error);
        });
    } catch (error) {
      emitter.emit("error", error);
    }
  });

  return emitter;
}

module.exports = { loadServer };
