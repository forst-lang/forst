import { execFile } from "node:child_process";
import { promisify } from "node:util";

import type { ForstExtensionConfig } from "./config";

const execFileAsync = promisify(execFile);

/** Plain fields gathered for bug reports and support channels. */
export interface ForstDebugInfoFields {
  extensionVersion: string;
  vscodeVersion: string;
  platform: string;
  forstPathSetting: string;
  resolvedCompilerPath: string;
  compilerVersionOutput: string;
  downloadCompiler: boolean;
  lspPort: number;
  lspAutoStart: boolean;
  lspLogLevel: string;
}

/**
 * Markdown-friendly block for pasting into GitHub issues (clipboard / output channel).
 */
export function formatForstDebugInfo(f: ForstDebugInfoFields): string {
  return [
    "### Forst (VS Code) debug info",
    "",
    "| | |",
    "|-|-|",
    `| Extension | \`${f.extensionVersion}\` |`,
    `| VS Code | \`${f.vscodeVersion}\` |`,
    `| Platform | \`${f.platform}\` |`,
    `| \`forst.path\` | \`${f.forstPathSetting}\` |`,
    `| Resolved compiler | \`${f.resolvedCompilerPath}\` |`,
    `| \`forst.downloadCompiler\` | \`${f.downloadCompiler}\` |`,
    `| \`forst.lsp.port\` | \`${f.lspPort}\` |`,
    `| \`forst.lsp.autoStart\` | \`${f.lspAutoStart}\` |`,
    `| \`forst.lsp.logLevel\` | \`${f.lspLogLevel}\` |`,
    "",
    "Compiler `forst version`:",
    "",
    "```",
    f.compilerVersionOutput.trimEnd(),
    "```",
    "",
  ].join("\n");
}

/**
 * Resolves the compiler path and runs `forst version` for a complete support bundle.
 */
export async function gatherForstDebugInfo(
  extensionVersion: string,
  vscodeVersion: string,
  cfg: ForstExtensionConfig,
  resolvedCompilerPath: string
): Promise<ForstDebugInfoFields> {
  let compilerVersionOutput = "";
  try {
    const { stdout, stderr } = await execFileAsync(
      resolvedCompilerPath,
      ["version"],
      { maxBuffer: 512 * 1024 }
    );
    compilerVersionOutput = [stdout, stderr].filter(Boolean).join("");
  } catch (e) {
    compilerVersionOutput =
      e instanceof Error ? e.message : `Error: ${String(e)}`;
  }

  return {
    extensionVersion,
    vscodeVersion,
    platform: process.platform,
    forstPathSetting: cfg.forstPath || "(empty — PATH or workspace bin/forst)",
    resolvedCompilerPath,
    compilerVersionOutput,
    downloadCompiler: cfg.downloadCompiler,
    lspPort: cfg.port,
    lspAutoStart: cfg.autoStart,
    lspLogLevel: cfg.logLevel,
  };
}
