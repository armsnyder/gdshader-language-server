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
const { createWriteStream } = require("node:fs");
const { unlink } = require("node:fs/promises");
const { tmpdir } = require("node:os");
const { join } = require("node:path");
const { pipeline } = require("node:stream/promises");
const { execFile, spawn } = require("node:child_process");

/**
 * @param {ReadableStream<Uint8Array>} response
 * @param {string} targetDir
 * @returns {Promise<void>}
 */
async function extractZip(response, targetDir) {
  if (process.platform !== "win32") {
    throw new Error("ZIP extraction is only supported on Windows.");
  }

  const tmpFile = join(tmpdir(), `gdshader-language-server-${Date.now()}.zip`);
  const tmpStream = createWriteStream(tmpFile);
  await pipeline(response, tmpStream);

  await new Promise((/** @type {(_?: void) => void} */ resolve, reject) => {
    execFile(
      "powershell.exe",
      [
        "-Command",
        `Expand-Archive -Path "${tmpFile}" -DestinationPath "${targetDir}" -Force`,
      ],
      (error, _, stderr) => {
        if (error) {
          logger().error(`Expand-Archive error: ${stderr}`);
          reject(error);
        } else {
          resolve();
        }
      },
    );
  });

  await unlink(tmpFile).catch((error) => {
    logger().error("Failed to delete temporary zip file:", error);
  });
}

/**
 * @param {ReadableStream<Uint8Array>} response
 * @param {string} targetDir
 * @returns {Promise<void>}
 */
async function extractTarGz(response, targetDir) {
  const tar = spawn("tar", ["-xz", "-C", targetDir]);

  /** @type {Buffer[]} */
  const stderrBuffer = [];
  tar.stderr.on("data", (data) => {
    stderrBuffer.push(data);
  });

  await pipeline(response, tar.stdin);

  return new Promise((/** @type {(_?: void) => void} */ resolve, reject) => {
    tar.on("close", (code) => {
      if (code === 0) resolve();
      else reject(new Error(`tar exited with code ${code}`));
    });
    tar.on("error", reject);
  }).catch((error) => {
    logger().error("tar error:", Buffer.concat(stderrBuffer).toString());
    throw error;
  });
}

module.exports = {
  extractZip,
  extractTarGz,
};
