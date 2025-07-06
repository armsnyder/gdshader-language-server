/**
 * Copyright (c) 2025 Adam Snyder <https://armsnyder.com> and contributors
 * SPDX-License-Identifier: MIT
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
      traceOutputChannel: logger(),
    };

    const client = new LanguageClient(
      "gdshader",
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
