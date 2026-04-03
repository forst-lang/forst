import * as cp from "child_process";
import * as vscode from "vscode";
import {
  ForstHttpLspClient,
  LspDiagnostic,
  LspHealthTimeoutError,
  PublishDiagnosticsParams,
} from "./lspClient";
import { registerForstLanguageFeatures } from "./languageFeatures";
import {
  ForstExtensionConfig,
  lspBaseUrl,
  readForstConfig,
  resolveForstExecutable,
} from "./config";

let diagnosticsCollection: vscode.DiagnosticCollection;
let output: vscode.OutputChannel;
let client: ForstHttpLspClient | undefined;
let serverProcess: cp.ChildProcess | undefined;
let initialized = false;
let restartToken = 0;

function isTrackableForstDoc(doc: vscode.TextDocument): boolean {
  return doc.languageId === "forst" && doc.uri.scheme === "file";
}

function clampNonNeg(n: number): number {
  if (!Number.isFinite(n) || n < 0) {
    return 0;
  }
  return Math.floor(n);
}

function toVsDiagnostic(d: LspDiagnostic): vscode.Diagnostic {
  let startLine = clampNonNeg(d.range.start.line);
  let startChar = clampNonNeg(d.range.start.character);
  let endLine = clampNonNeg(d.range.end.line);
  let endChar = clampNonNeg(d.range.end.character);
  if (
    endLine < startLine ||
    (endLine === startLine && endChar < startChar)
  ) {
    endLine = startLine;
    endChar = startChar;
  }
  const range = new vscode.Range(startLine, startChar, endLine, endChar);
  let severity = vscode.DiagnosticSeverity.Hint;
  switch (d.severity) {
    case 1:
      severity = vscode.DiagnosticSeverity.Error;
      break;
    case 2:
      severity = vscode.DiagnosticSeverity.Warning;
      break;
    case 3:
      severity = vscode.DiagnosticSeverity.Information;
      break;
    case 4:
      severity = vscode.DiagnosticSeverity.Hint;
      break;
    default:
      severity = vscode.DiagnosticSeverity.Error;
  }
  const diag = new vscode.Diagnostic(range, d.message, severity);
  diag.source = d.source ?? "forst";
  if (d.code !== undefined) {
    diag.code = d.code;
  }
  return diag;
}

/**
 * Optional last line of defense for older forst LSP builds. The authoritative fix is on the
 * server: only real diagnostics belong in publishDiagnostics / didOpen results (LSP practice).
 */
function isPhaseMilestoneNoise(d: LspDiagnostic): boolean {
  const msg = (d.message ?? "").trim().toLowerCase();
  if (!msg.endsWith("completed")) {
    return false;
  }
  const src = (d.source ?? "").toLowerCase();
  return [
    "lexer",
    "parser",
    "typechecker",
    "transformer",
    "discovery",
    "executor",
  ].includes(src);
}

function applyPublishDiagnostics(
  collection: vscode.DiagnosticCollection,
  pub: PublishDiagnosticsParams
): void {
  const uri = vscode.Uri.parse(pub.uri);
  const vs = (pub.diagnostics ?? [])
    .filter((d) => !isPhaseMilestoneNoise(d))
    .map(toVsDiagnostic);
  collection.set(uri, vs);
}

async function waitForHealth(baseUrl: string, timeoutMs: number): Promise<void> {
  const healthUrl = `${baseUrl.replace(/\/$/, "")}/health`;
  const deadline = Date.now() + timeoutMs;
  let lastErr: string | undefined;
  while (Date.now() < deadline) {
    try {
      const r = await fetch(healthUrl);
      if (r.ok) {
        return;
      }
      lastErr = `HTTP ${r.status}`;
    } catch (e) {
      lastErr = e instanceof Error ? e.message : String(e);
    }
    await new Promise((r) => setTimeout(r, 100));
  }
  throw new LspHealthTimeoutError(healthUrl, lastErr);
}

function waitForChildSpawn(proc: cp.ChildProcess): Promise<void> {
  return new Promise((resolve, reject) => {
    proc.once("error", reject);
    proc.once("spawn", () => resolve());
  });
}

function stopServerProcess(): void {
  if (serverProcess) {
    try {
      serverProcess.kill();
    } catch {
      /* ignore */
    }
    serverProcess = undefined;
  }
}

async function ensureServerRunning(
  cfg: ForstExtensionConfig,
  log: vscode.OutputChannel
): Promise<void> {
  const base = lspBaseUrl(cfg.port);
  if (!cfg.autoStart) {
    await waitForHealth(base, 8000);
    log.appendLine(`Using existing Forst LSP at ${base}`);
    return;
  }
  if (serverProcess) {
    await waitForHealth(base, 2000);
    return;
  }
  const exe = resolveForstExecutable(cfg.forstPath);
  if (exe !== cfg.forstPath) {
    log.appendLine(`Resolved forst executable: ${exe}`);
  }
  log.appendLine(
    `Starting: ${exe} lsp -port ${cfg.port} -log-level ${cfg.logLevel}`
  );
  serverProcess = cp.spawn(
    exe,
    ["lsp", "-port", String(cfg.port), "-log-level", cfg.logLevel],
    {
      stdio: ["ignore", "pipe", "pipe"],
      env: process.env,
    }
  );
  serverProcess.stdout?.on("data", (b: Buffer) => {
    log.append(b.toString());
  });
  serverProcess.stderr?.on("data", (b: Buffer) => {
    log.append(b.toString());
  });
  serverProcess.on("error", (err: NodeJS.ErrnoException) => {
    log.appendLine(`forst process error: ${err.message}`);
  });
  serverProcess.on("exit", (code) => {
    log.appendLine(`forst lsp exited with code ${code ?? "null"}`);
    serverProcess = undefined;
  });
  try {
    await waitForChildSpawn(serverProcess);
  } catch (err) {
    serverProcess = undefined;
    const msg = err instanceof Error ? err.message : String(err);
    log.appendLine(
      `Could not spawn forst (${exe}): ${msg}. Set **forst.path**, put forst on PATH, or open a workspace that contains **bin/forst** after \`task build\`.`
    );
    throw err;
  }
  await waitForHealth(base, 15000);
  log.appendLine(`Forst LSP ready at ${base}`);
}

async function ensureClient(
  cfg: ForstExtensionConfig,
  log: vscode.OutputChannel
): Promise<ForstHttpLspClient> {
  await ensureServerRunning(cfg, log);
  const base = lspBaseUrl(cfg.port);
  if (!client || client.base !== base) {
    client = new ForstHttpLspClient(base);
    initialized = false;
  }
  if (!initialized) {
    const root =
      vscode.workspace.workspaceFolders?.[0]?.uri.toString() ?? null;
    await client.initialize(root);
    initialized = true;
    log.appendLine("LSP initialize completed");
  }
  return client;
}

export async function activate(context: vscode.ExtensionContext): Promise<void> {
  diagnosticsCollection = vscode.languages.createDiagnosticCollection("forst");
  output = vscode.window.createOutputChannel("Forst");
  context.subscriptions.push(diagnosticsCollection, output);

  output.appendLine("Forst extension activated.");
  output.appendLine(
    "Open a .ft file (language mode “Forst”) or a folder that contains .ft files to run diagnostics."
  );
  output.show(true);

  registerForstLanguageFeatures(context, {
    getClient: () => ensureClient(readForstConfig(), output),
    isTrackableForstDoc,
  });

  const withClient = async <T>(
    fn: (c: ForstHttpLspClient, cfg: ForstExtensionConfig, token: number) => Promise<T>
  ): Promise<void> => {
    const cfg = readForstConfig();
    const token = restartToken;
    try {
      const c = await ensureClient(cfg, output);
      if (token !== restartToken) {
        return;
      }
      await fn(c, cfg, token);
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e);
      output.appendLine(`Forst: ${msg}`);
      void vscode.window.showWarningMessage(`Forst: ${msg}`);
    }
  };

  context.subscriptions.push(
    vscode.workspace.onDidOpenTextDocument((doc) => {
      if (!isTrackableForstDoc(doc)) {
        return;
      }
      void withClient(async (c, _cfg, token) => {
        if (token !== restartToken) {
          return;
        }
        const pub = await c.didOpen({
          uri: doc.uri.toString(),
          version: doc.version,
          text: doc.getText(),
        });
        applyPublishDiagnostics(diagnosticsCollection, pub);
      });
    })
  );

  context.subscriptions.push(
    vscode.workspace.onDidChangeTextDocument((e) => {
      const doc = e.document;
      if (!isTrackableForstDoc(doc)) {
        return;
      }
      void withClient(async (c, _cfg, token) => {
        if (token !== restartToken) {
          return;
        }
        const pub = await c.didChange({
          uri: doc.uri.toString(),
          version: doc.version,
          text: doc.getText(),
        });
        applyPublishDiagnostics(diagnosticsCollection, pub);
      });
    })
  );

  context.subscriptions.push(
    vscode.workspace.onDidCloseTextDocument((doc) => {
      if (!isTrackableForstDoc(doc)) {
        return;
      }
      diagnosticsCollection.delete(doc.uri);
      void withClient(async (c, _cfg, token) => {
        if (token !== restartToken) {
          return;
        }
        const pub = await c.didClose(doc.uri.toString());
        applyPublishDiagnostics(diagnosticsCollection, pub);
      });
    })
  );

  context.subscriptions.push(
    vscode.commands.registerCommand("forst.restartLanguageServer", async () => {
      restartToken++;
      initialized = false;
      client = undefined;
      diagnosticsCollection.clear();
      stopServerProcess();
      output.appendLine("Forst language server restarted (local state cleared).");
      const cfg = readForstConfig();
      for (const doc of vscode.workspace.textDocuments) {
        if (!isTrackableForstDoc(doc)) {
          continue;
        }
        try {
          const c = await ensureClient(cfg, output);
          const pub = await c.didOpen({
            uri: doc.uri.toString(),
            version: doc.version,
            text: doc.getText(),
          });
          applyPublishDiagnostics(diagnosticsCollection, pub);
        } catch (e) {
          const msg = e instanceof Error ? e.message : String(e);
          output.appendLine(`Forst: ${msg}`);
        }
      }
    })
  );

  for (const doc of vscode.workspace.textDocuments) {
    if (isTrackableForstDoc(doc)) {
      try {
        const cfg = readForstConfig();
        const c = await ensureClient(cfg, output);
        const pub = await c.didOpen({
          uri: doc.uri.toString(),
          version: doc.version,
          text: doc.getText(),
        });
        applyPublishDiagnostics(diagnosticsCollection, pub);
      } catch (e) {
        const msg = e instanceof Error ? e.message : String(e);
        output.appendLine(`Forst: ${msg}`);
      }
    }
  }
}

export function deactivate(): void {
  initialized = false;
  client = undefined;
  diagnosticsCollection?.clear();
  stopServerProcess();
}
