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
