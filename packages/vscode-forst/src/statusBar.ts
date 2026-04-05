import * as vscode from "vscode";
import type { LogOutputChannel } from "vscode";
import { readForstConfig, resolveForstExecutable } from "./config";

export type ForstLspStatusKind = "ready" | "idle" | "error";

let item: vscode.StatusBarItem | undefined;

/**
 * Updates the right-side status bar: port, resolved `forst` binary, and a one-click link to the Forst output channel.
 */
export function refreshForstStatusBar(
  kind: ForstLspStatusKind,
  detail?: string
): void {
  if (!item) {
    return;
  }
  const cfg = readForstConfig();
  const exe = resolveForstExecutable(cfg.forstPath);
  const icon =
    kind === "ready"
      ? "$(check)"
      : kind === "error"
        ? "$(error)"
        : "$(circle-outline)";
  item.text = `${icon} Forst :${cfg.port}`;
  const state =
    kind === "ready"
      ? "connected"
      : kind === "error"
        ? "error"
        : "idle / starting";
  const md = new vscode.MarkdownString(
    `**Forst LSP** — *${state}*\n\n` +
      `- **Port:** ${cfg.port}\n` +
      `- **Executable:** \`${exe}\`\n` +
      `- **Auto-start:** ${cfg.autoStart}\n` +
      (detail ? `\n${detail}\n` : "") +
      `\n[Open Forst output](command:forst.showOutput)`
  );
  md.isTrusted = true;
  item.tooltip = md;
}

/** Registers the status bar item, the show-output command, and config-driven refresh. */
export function registerForstStatusBar(
  context: vscode.ExtensionContext,
  output: LogOutputChannel
): void {
  item = vscode.window.createStatusBarItem(vscode.StatusBarAlignment.Right, 100);
  item.name = "Forst LSP";
  item.command = "forst.showOutput";
  context.subscriptions.push(item);

  context.subscriptions.push(
    vscode.commands.registerCommand("forst.showOutput", () => {
      output.show(true);
    })
  );

  context.subscriptions.push(
    vscode.workspace.onDidChangeConfiguration((e) => {
      if (e.affectsConfiguration("forst")) {
        refreshForstStatusBar("idle");
      }
    })
  );

  refreshForstStatusBar("idle");
  item.show();
}
