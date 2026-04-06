import * as vscode from "vscode";
import type { LogOutputChannel } from "vscode";
import { readForstConfig, resolveForstExecutableWithCli } from "./config";
import { formatForstDebugInfo, gatherForstDebugInfo } from "./debugInfo";
import { applyPublishDiagnostics } from "./lsp/diagnostics";
import { registerForstLanguageFeatures } from "./lsp/languageFeatures";
import type { ForstLspChildState } from "./lsp/process";
import { stopForstLspProcess } from "./lsp/process";
import type { LspSessionState } from "./lsp/session";
import { getOrCreateLspClient } from "./lsp/session";
import { refreshForstStatusBar, registerForstStatusBar } from "./statusBar";

/** Problems collection and log channel—module scope matches VS Code’s single extension instance lifecycle. */
let diagnosticsCollection: vscode.DiagnosticCollection;
let output: LogOutputChannel;

/** Bumps invalidate in-flight `withClient` work after a restart so stale responses cannot apply. */
let restartToken = 0;

const childState: ForstLspChildState = { process: undefined };

/** Single source of truth for HTTP client + spawn state shared by activation, listeners, and restart. */
const session: LspSessionState = {
  client: undefined,
  initialized: false,
  child: childState,
};

/** Restricts document events to on-disk Forst sources—virtual and untitled docs skip LSP sync. */
function isTrackableForstDoc(doc: vscode.TextDocument): boolean {
  return doc.languageId === "forst" && doc.uri.scheme === "file";
}

/** Subscribes document sync, diagnostics, language features, and the restart command to the shared LSP session. */
export async function activate(context: vscode.ExtensionContext): Promise<void> {
  diagnosticsCollection = vscode.languages.createDiagnosticCollection("forst");
  output = vscode.window.createOutputChannel("Forst", { log: true });
  context.subscriptions.push(diagnosticsCollection, output);
  registerForstStatusBar(context, output);

  context.subscriptions.push(
    vscode.commands.registerCommand("forst.copyDebugInfo", async () => {
      const cfg = readForstConfig();
      const vscodeExtVer = String(
        (context.extension.packageJSON as { version?: string }).version ?? "?"
      );
      try {
        const resolved = await resolveForstExecutableWithCli(cfg, output);
        const fields = await gatherForstDebugInfo(
          vscodeExtVer,
          vscode.version,
          cfg,
          resolved
        );
        const text = formatForstDebugInfo(fields);
        await vscode.env.clipboard.writeText(text);
        output.info("Forst debug info copied to clipboard.");
        void vscode.window.showInformationMessage(
          "Forst: debug info copied to clipboard."
        );
      } catch (e) {
        const msg = e instanceof Error ? e.message : String(e);
        output.error(`Copy debug info failed: ${msg}`);
        void vscode.window.showErrorMessage(`Forst: ${msg}`);
      }
    })
  );

  const vscodeExtVer = String(
    (context.extension.packageJSON as { version?: string }).version ?? "?"
  );
  output.info(`Forst VS Code extension ${vscodeExtVer} — activated. Open a .ft file (language mode “Forst”) or a folder that contains .ft files to run diagnostics.`);
  output.show(true);

  registerForstLanguageFeatures(context, {
    getClient: () => getOrCreateLspClient(readForstConfig(), output, session),
    isTrackableForstDoc,
    log: output,
  });

  const withClient = async <T>(
    fn: (
      c: Awaited<ReturnType<typeof getOrCreateLspClient>>,
      token: number
    ) => Promise<T>
  ): Promise<void> => {
    const cfg = readForstConfig();
    const token = restartToken;
    try {
      const c = await getOrCreateLspClient(cfg, output, session);
      if (token !== restartToken) {
        output.debug(
          "Skipped document sync (restart superseded this request)."
        );
        return;
      }
      await fn(c, token);
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e);
      output.warn(`Document sync / LSP: ${msg}`);
      refreshForstStatusBar("error", msg);
      void vscode.window.showWarningMessage(`Forst: ${msg}`);
    }
  };

  context.subscriptions.push(
    vscode.workspace.onDidOpenTextDocument((doc) => {
      if (!isTrackableForstDoc(doc)) {
        return;
      }
      void withClient(async (c, token) => {
        if (token !== restartToken) {
          return;
        }
        const pub = await c.didOpen({
          uri: doc.uri.toString(),
          version: doc.version,
          text: doc.getText(),
        });
        applyPublishDiagnostics(diagnosticsCollection, pub, output);
      });
    })
  );

  context.subscriptions.push(
    vscode.workspace.onDidChangeTextDocument((e) => {
      const doc = e.document;
      if (!isTrackableForstDoc(doc)) {
        return;
      }
      void withClient(async (c, token) => {
        if (token !== restartToken) {
          return;
        }
        const pub = await c.didChange({
          uri: doc.uri.toString(),
          version: doc.version,
          text: doc.getText(),
        });
        applyPublishDiagnostics(diagnosticsCollection, pub, output);
      });
    })
  );

  context.subscriptions.push(
    vscode.workspace.onDidCloseTextDocument((doc) => {
      if (!isTrackableForstDoc(doc)) {
        return;
      }
      diagnosticsCollection.delete(doc.uri);
      void withClient(async (c, token) => {
        if (token !== restartToken) {
          return;
        }
        const pub = await c.didClose(doc.uri.toString());
        applyPublishDiagnostics(diagnosticsCollection, pub, output);
      });
    })
  );

  context.subscriptions.push(
    vscode.commands.registerCommand(
      "forst.showReferences",
      async (uriStr: string, line: number, char: number) => {
        const uri = vscode.Uri.parse(uriStr);
        const pos = new vscode.Position(line, char);
        const refs = await vscode.commands.executeCommand<
          vscode.Location[] | undefined
        >("vscode.executeReferenceProvider", uri, pos);
        if (refs == null || refs.length === 0) {
          return;
        }
        await vscode.commands.executeCommand(
          "editor.action.showReferences",
          uri,
          pos,
          refs
        );
      }
    )
  );

    context.subscriptions.push(
    vscode.commands.registerCommand("forst.restartLanguageServer", async () => {
      restartToken++;
      session.initialized = false;
      session.client = undefined;
      diagnosticsCollection.clear();
      refreshForstStatusBar("idle");
      stopForstLspProcess(session.child, output);
      output.info("Forst language server restarted (local state cleared).");
      const cfg = readForstConfig();
      for (const doc of vscode.workspace.textDocuments) {
        if (!isTrackableForstDoc(doc)) {
          continue;
        }
        try {
          const c = await getOrCreateLspClient(cfg, output, session);
          const pub = await c.didOpen({
            uri: doc.uri.toString(),
            version: doc.version,
            text: doc.getText(),
          });
          applyPublishDiagnostics(diagnosticsCollection, pub, output);
        } catch (e) {
          const msg = e instanceof Error ? e.message : String(e);
          refreshForstStatusBar("error", msg);
          output.error(
            `After restart, failed to sync open document ${doc.uri.fsPath}: ${msg}`
          );
        }
      }
    })
  );

  for (const doc of vscode.workspace.textDocuments) {
    if (isTrackableForstDoc(doc)) {
      try {
        const cfg = readForstConfig();
        const c = await getOrCreateLspClient(cfg, output, session);
        const pub = await c.didOpen({
          uri: doc.uri.toString(),
          version: doc.version,
          text: doc.getText(),
        });
        applyPublishDiagnostics(diagnosticsCollection, pub, output);
      } catch (e) {
        const msg = e instanceof Error ? e.message : String(e);
        refreshForstStatusBar("error", msg);
        output.error(
          `Initial sync failed for ${doc.uri.fsPath}: ${msg}`
        );
      }
    }
  }
}

/** Resets client state and tears down the child process so reloading the window leaves no stray `forst lsp`. */
export function deactivate(): void {
  output?.debug("Forst extension deactivate.");
  session.initialized = false;
  session.client = undefined;
  diagnosticsCollection?.clear();
  stopForstLspProcess(session.child, output);
}
