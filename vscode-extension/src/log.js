/**
 * Copyright (c) 2025 Adam Snyder <https://armsnyder.com> and contributors
 * SPDX-License-Identifier: MIT
 */

/**
 * @type {import("vscode").LogOutputChannel?}
 */
let instance = null;

/** @param {import("vscode").LogOutputChannel} outputChannel */
function setLogger(outputChannel) {
  instance = outputChannel;
}

function logger() {
  if (!instance) {
    throw new Error("Logger not initialized");
  }
  return instance;
}

module.exports = {
  setLogger,
  logger,
};
