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
const { setLogger, logger } = require("./log");
const { loadServer } = require("./server");
const vscode = require("vscode");
const { LanguageClient } = require("vscode-languageclient/node");

/** @param {import('vscode').ExtensionContext} context */
async function activate(context) {
  {
    const logger = vscode.window.createOutputChannel(
      "GDShader Language Server",
      { log: true },
    );
    context.subscriptions.push(logger);
    setLogger(logger);
  }

  logger().info("Loading the server binary...");
  const loader = loadServer(context);
  const loadStatus = vscode.window.createStatusBarItem(
    "loader",
    vscode.StatusBarAlignment.Left,
    100,
  );
  context.subscriptions.push(loadStatus);

  try {
    const binPath = await new Promise((resolve, reject) => {
      loader.on("downloadStarted", () => {
        loadStatus.text = "$(sync~spin) Downloading...";
        loadStatus.show();
      });

      loader.on("error", (error) => {
        loadStatus.dispose();
        reject(error);
      });

      loader.on("ready", (binPath) => {
        loadStatus.dispose();
        resolve(binPath);
      });
    });

    logger().info(`Server binary loaded from: ${binPath}`);

    /** @type {import('vscode-languageclient/node').ServerOptions} */
    const serverOptions = {
      command: binPath,
    };

    /** @type {import('vscode-languageclient/node').LanguageClientOptions} */
    const clientOptions = {
      documentSelector: [{ scheme: "file", language: "gdshader" }],
      synchronize: {
        fileEvents: vscode.workspace.createFileSystemWatcher("**/.clientrc"),
      },
      outputChannel: logger(),
    };

    const client = new LanguageClient(
      "gdshaderLanguageServer",
      "GDShader Language Server",
      serverOptions,
      clientOptions,
    );

    context.subscriptions.push({
      dispose: () => {
        client.stop().catch((error) => {
          vscode.window.showErrorMessage(
            `Error stopping gdshader-language-server: ${error.message}`,
          );
        });
      },
    });

    logger().info("Starting GDShader Language Server...");
    await client.start();
    logger().info("GDShader Language Server started successfully.");
  } catch (error) {
    logger().error("Error starting gdshader-language-server:", error);
    if (error instanceof Error) {
      vscode.window.showErrorMessage(
        `Error starting gdshader-language-server: ${error.message}`,
      );
    } else {
      vscode.window.showErrorMessage(
        "An unexpected error occurred while starting the gdshader-language-server.",
      );
    }
  }
}

module.exports = { activate };
