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
