/**
 * Copyright (c) 2025 Adam Snyder <https://armsnyder.com> and contributors
 * SPDX-License-Identifier: MIT
 */
const vscode = require("vscode");

/**
 * @typedef Configuration
 * @property {string} serverPathOverride
 * @property {boolean} disableSafetyCheck
 */

/** @returns {Configuration} */
function getConfiguration() {
  const config = vscode.workspace.getConfiguration("gdshader.danger");
  return {
    serverPathOverride: config.get("serverPathOverride") ?? "",
    disableSafetyCheck: config.get("disableSafetyCheck") ?? false,
  };
}

module.exports = {
  getConfiguration,
};
