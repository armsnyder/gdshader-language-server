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
const { createReadStream } = require("node:fs");
const { readFile } = require("node:fs/promises");
const { platform } = require("node:os");
const { join } = require("node:path");
const { createVerify } = require("node:crypto");
const { getBinPath, getSignaturePath } = require("./storage");

function shouldValidateBinarySignature() {
  // MacOS has validation built-in with Gatekeeper, and the MacOS archives do
  // not include signature files.
  return platform() !== "darwin";
}

/** @param {import('vscode').ExtensionContext} context @returns {Promise<void>} */
async function validateBinarySignature(context) {
  const publicKeyPath = join(context.extension.extensionPath, "cosign.pub");
  const pubKey = await readFile(publicKeyPath, "utf8");

  const signature = Buffer.from(
    (await readFile(getSignaturePath(context), "utf8")).trim(),
    "base64",
  );

  const verify = createVerify("sha256");

  await new Promise((/** @type {(_?: void) => void} */ resolve, reject) => {
    const stream = createReadStream(getBinPath(context));
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
