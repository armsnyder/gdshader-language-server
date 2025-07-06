/**
 * Copyright (c) 2025 Adam Snyder <https://armsnyder.com> and contributors
 * SPDX-License-Identifier: MIT
 */
const { createReadStream } = require("node:fs");
const { readFile } = require("node:fs/promises");
const { platform } = require("node:os");
const { join } = require("node:path");
const { createVerify } = require("node:crypto");
const { getBinPath, getSignaturePath } = require("./storage");
const { getConfiguration } = require("../config");
const { logger } = require("../log");

function shouldValidateBinarySignature() {
  // MacOS has validation built-in with Gatekeeper, and the MacOS archives do
  // not include signature files.
  if (platform() === "darwin") {
    return false;
  }
  if (getConfiguration().disableSafetyCheck) {
    logger().warn("Safety check is disabled. This is not recommended.");
    return false;
  }
  return true;
}

/** @param {import('vscode').ExtensionContext} context @returns {Promise<void>} */
async function validateBinarySignature(context) {
  const publicKeyPath = join(context.extension.extensionPath, "cosign.pub");
  const pubKey = await readFile(publicKeyPath, "utf8");

  const signature = Buffer.from(
    (await readFile(await getSignaturePath(context), "utf8")).trim(),
    "base64",
  );

  const verify = createVerify("sha256");
  const binPath = await getBinPath(context);

  await new Promise((/** @type {(_?: void) => void} */ resolve, reject) => {
    const stream = createReadStream(binPath);
    stream.on("data", (chunk) => verify.update(chunk));
    stream.on("error", reject);
    stream.on("end", () => {
      verify.end();
      resolve();
    });
  });

  if (!verify.verify(pubKey, signature)) {
    throw new Error(
      "The language server binary may be corrupted or can no longer be verified. Try updating the extension.",
    );
  }
}

module.exports = {
  shouldValidateBinarySignature,
  validateBinarySignature,
};
