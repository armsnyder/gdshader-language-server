/**
 * Copyright (c) 2025 Adam Snyder <https://armsnyder.com> and contributors
 * SPDX-License-Identifier: MIT
 * Please retain this header in any redistributions of this code.
 */
const { logger } = require("../log");
const { platform, arch } = require("node:os");
const { mkdir } = require("node:fs/promises");
const { getBinDir } = require("./storage");
const { extractZip, extractTarGz } = require("./extract");

/**
 * Get the name of the GitHub release asset containing the server archive.
 * @returns {string}
 */
function getAssetName() {
  let osName;
  let archiveExt;

  switch (platform()) {
    case "win32":
      osName = "Windows";
      archiveExt = ".zip";
      break;
    case "darwin":
      osName = "Darwin";
      archiveExt = ".tar.gz";
      break;
    case "linux":
      osName = "Linux";
      archiveExt = ".tar.gz";
      break;
    default:
      throw new Error(`Unsupported platform: ${platform()}`);
  }

  let archName;

  switch (arch()) {
    case "x64":
      archName = "x86_64";
      break;
    case "arm64":
      archName = "arm64";
      break;
    default:
      throw new Error(`Unsupported architecture: ${arch()}`);
  }

  return `gdshader-language-server_${osName}_${archName}${archiveExt}`;
}

/** @param {import('vscode').ExtensionContext} context  @returns {string} */
function getAssetUrl(context) {
  return `https://github.com/armsnyder/gdshader-language-server/releases/download/v${context.extension.packageJSON.version}/${getAssetName()}`;
}

/** @param {string} url @returns {Promise<ReadableStream<Uint8Array>>} */
async function startDownload(url) {
  const response = await fetch(url);
  if (!response.ok) {
    throw new Error(`Failed to download from ${url}: ${response.statusText}`);
  }
  if (!response.body) {
    throw new Error(`No response body for ${url}`);
  }
  return response.body;
}

/** @param {import('vscode').ExtensionContext} context @returns {Promise<void>} */
async function downloadAndExtractServerArchive(context) {
  const binDir = getBinDir(context);
  await mkdir(binDir, { recursive: true });

  const assetName = getAssetName();
  const assetUrl = getAssetUrl(context);

  logger().info(`Downloading server archive from: ${assetUrl}`);

  const response = await startDownload(assetUrl);

  logger().info(`Extracting server archive to: ${binDir}`);

  if (assetName.endsWith(".zip")) {
    return extractZip(response, binDir);
  }

  if (assetName.endsWith(".tar.gz")) {
    return extractTarGz(response, binDir);
  }

  throw new Error(`Unsupported asset type: ${assetName}`);
}

module.exports = { downloadAndExtractServerArchive };
