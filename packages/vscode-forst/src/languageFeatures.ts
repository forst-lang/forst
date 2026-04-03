import * as vscode from "vscode";
import type { ForstHttpLspClient } from "./lspClient";
import type { LspHoverContents, LspHoverResult } from "./lspClient";
import {
  lspCodeActionToVs,
  lspCodeLensToVs,
  lspCompletionItemToVs,
  lspDefinitionResultToVs,
  lspDocumentSymbolsToVs,
  lspFoldingRangeToVs,
  lspLocationToVs,
  lspSymbolInformationToVs,
  lspTextEditsToVs,
} from "./lspConverters";

const FORST_SELECTOR: vscode.DocumentSelector = {
  language: "forst",
  scheme: "file",
};

function clampNonNeg(n: number): number {
  if (!Number.isFinite(n) || n < 0) {
    return 0;
  }
  return Math.floor(n);
}

function markdownFromHoverContents(c: LspHoverContents): string {
  if (typeof c === "string") {
    return c;
  }
  if (Array.isArray(c)) {
    return c
      .map((part) =>
        typeof part === "string" ? part : part.value ?? ""
      )
      .join("\n\n");
  }
  if (
    "kind" in c &&
    typeof (c as { kind?: string }).kind === "string" &&
    (c as { kind: string }).kind === "markdown" &&
    "value" in c
  ) {
    return String((c as { value: string }).value);
  }
  if ("value" in c && typeof (c as { value?: string }).value === "string") {
    return (c as { value: string }).value;
  }
  return "";
}

function lspHoverToVs(h: LspHoverResult): vscode.Hover {
  const md = new vscode.MarkdownString(markdownFromHoverContents(h.contents));
  md.isTrusted = false;
  const hover = new vscode.Hover(md);
  if (h.range) {
    const s = h.range.start;
    const e = h.range.end;
    hover.range = new vscode.Range(
      clampNonNeg(s.line),
      clampNonNeg(s.character),
      clampNonNeg(e.line),
      clampNonNeg(e.character)
    );
  }
  return hover;
}

export interface ForstLanguageFeaturesOptions {
  getClient: () => Promise<ForstHttpLspClient>;
  isTrackableForstDoc: (doc: vscode.TextDocument) => boolean;
}

/**
 * Registers VS Code language features backed by the Forst HTTP LSP methods the server already implements.
 * Methods that still return empty / null on the server behave as no-ops until implemented in forst LSP.
 */
export function registerForstLanguageFeatures(
  context: vscode.ExtensionContext,
  options: ForstLanguageFeaturesOptions
): void {
  const { getClient, isTrackableForstDoc } = options;

  async function withDoc<T>(
    doc: vscode.TextDocument,
    empty: T,
    fn: (c: ForstHttpLspClient) => Promise<T>
  ): Promise<T> {
    if (!isTrackableForstDoc(doc)) {
      return empty;
    }
    try {
      const c = await getClient();
      return await fn(c);
    } catch {
      return empty;
    }
  }

  async function withClient<T>(empty: T, fn: (c: ForstHttpLspClient) => Promise<T>): Promise<T> {
    try {
      const c = await getClient();
      return await fn(c);
    } catch {
      return empty;
    }
  }

  context.subscriptions.push(
    vscode.languages.registerHoverProvider(FORST_SELECTOR, {
      provideHover: async (doc, pos) =>
        withDoc(doc, null as vscode.Hover | null, async (c) => {
          const h = await c.hover({
            uri: doc.uri.toString(),
            line: pos.line,
            character: pos.character,
          });
          if (!h) {
            return null;
          }
          return lspHoverToVs(h);
        }),
    })
  );

  context.subscriptions.push(
    vscode.languages.registerCompletionItemProvider(
      FORST_SELECTOR,
      {
        provideCompletionItems: async (doc, pos, _token, ctx) =>
          withDoc(doc, [] as vscode.CompletionItem[], async (c) => {
            const list = await c.completion({
              uri: doc.uri.toString(),
              line: pos.line,
              character: pos.character,
              triggerKind: ctx.triggerKind,
              triggerCharacter: ctx.triggerCharacter ?? null,
            });
            if (!list || list.items.length === 0) {
              return [];
            }
            const items = list.items.map(lspCompletionItemToVs);
            if (list.isIncomplete) {
              return new vscode.CompletionList(items, true);
            }
            return items;
          }),
      },
      ".",
      ":",
      "(",
      " "
    )
  );

  context.subscriptions.push(
    vscode.languages.registerDefinitionProvider(FORST_SELECTOR, {
      provideDefinition: async (doc, pos) =>
        withDoc(doc, undefined, async (c) => {
          const def = await c.definition({
            uri: doc.uri.toString(),
            line: pos.line,
            character: pos.character,
          });
          return lspDefinitionResultToVs(def);
        }),
    })
  );

  context.subscriptions.push(
    vscode.languages.registerReferenceProvider(FORST_SELECTOR, {
      provideReferences: async (doc, pos, ctx) =>
        withDoc(doc, [] as vscode.Location[], async (c) => {
          const refs = await c.references({
            uri: doc.uri.toString(),
            line: pos.line,
            character: pos.character,
            includeDeclaration: ctx.includeDeclaration,
          });
          return refs.map(lspLocationToVs);
        }),
    })
  );

  context.subscriptions.push(
    vscode.languages.registerDocumentSymbolProvider(FORST_SELECTOR, {
      provideDocumentSymbols: async (doc) =>
        withDoc(doc, [] as vscode.SymbolInformation[], async (c) => {
          const raw = await c.documentSymbol({ uri: doc.uri.toString() });
          return lspDocumentSymbolsToVs(raw);
        }),
    })
  );

  context.subscriptions.push(
    vscode.languages.registerWorkspaceSymbolProvider({
      provideWorkspaceSymbols: async (query) =>
        withClient([] as vscode.SymbolInformation[], async (c) => {
          const raw = await c.workspaceSymbol(query);
          const out: vscode.SymbolInformation[] = [];
          for (const item of raw) {
            const si = lspSymbolInformationToVs(item);
            if (si) {
              out.push(si);
            }
          }
          return out;
        }),
    })
  );

  context.subscriptions.push(
    vscode.languages.registerDocumentFormattingEditProvider(FORST_SELECTOR, {
      provideDocumentFormattingEdits: async (doc, opt) =>
        withDoc(doc, [] as vscode.TextEdit[], async (c) => {
          const edits = await c.formatting({
            uri: doc.uri.toString(),
            tabSize: opt.tabSize,
            insertSpaces: opt.insertSpaces,
          });
          return lspTextEditsToVs(edits);
        }),
    })
  );

  context.subscriptions.push(
    vscode.languages.registerCodeActionsProvider(FORST_SELECTOR, {
      provideCodeActions: async (doc, range, ctx) =>
        withDoc(doc, [] as vscode.CodeAction[], async (c) => {
          const raw = await c.codeAction({
            uri: doc.uri.toString(),
            range: {
              start: {
                line: range.start.line,
                character: range.start.character,
              },
              end: { line: range.end.line, character: range.end.character },
            },
            diagnostics: [],
          });
          const out: vscode.CodeAction[] = [];
          for (const item of raw) {
            const a = lspCodeActionToVs(item);
            if (a) {
              out.push(a);
            }
          }
          return out;
        }),
    })
  );

  context.subscriptions.push(
    vscode.languages.registerCodeLensProvider(FORST_SELECTOR, {
      provideCodeLenses: async (doc) =>
        withDoc(doc, [] as vscode.CodeLens[], async (c) => {
          const raw = await c.codeLens({ uri: doc.uri.toString() });
          const out: vscode.CodeLens[] = [];
          for (const item of raw) {
            const lens = lspCodeLensToVs(item);
            if (lens) {
              out.push(lens);
            }
          }
          return out;
        }),
    })
  );

  context.subscriptions.push(
    vscode.languages.registerFoldingRangeProvider(FORST_SELECTOR, {
      provideFoldingRanges: async (doc) =>
        withDoc(doc, [] as vscode.FoldingRange[], async (c) => {
          const raw = await c.foldingRange({ uri: doc.uri.toString() });
          const out: vscode.FoldingRange[] = [];
          for (const item of raw) {
            const fr = lspFoldingRangeToVs(item);
            if (fr) {
              out.push(fr);
            }
          }
          return out;
        }),
    })
  );

}
