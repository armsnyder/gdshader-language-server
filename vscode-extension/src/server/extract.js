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
