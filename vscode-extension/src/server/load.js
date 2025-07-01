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
  if (!(await fileExists(getBinPath(context)))) {
    return false;
  }
  if (shouldValidateBinarySignature()) {
    return await fileExists(getSignaturePath(context));
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
