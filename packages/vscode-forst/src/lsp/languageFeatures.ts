import * as vscode from "vscode";
import type { LogOutputChannel } from "vscode";
import type { ForstHttpLspClient } from "./client";
import type { LspHoverContents, LspHoverResult } from "./types";
import {
  codeActionToVs,
  codeLensToVs,
  completionItemToVs,
  definitionResultToVs,
  documentSymbolsToVs,
  foldingRangeToVs,
  locationToVs,
  symbolInformationToVs,
  textEditsToVs,
} from "./converters";

/** Scopes registered providers to on-disk `.ft` buffers so other schemes do not hit the HTTP client. */
const FORST_SELECTOR: vscode.DocumentSelector = {
  language: "forst",
  scheme: "file",
};

/** Normalizes hover range coordinates when the server omits or shortens the extent. */
function clampNonNeg(n: number): number {
  if (!Number.isFinite(n) || n < 0) {
    return 0;
  }
  return Math.floor(n);
}

/** Collapses heterogeneous LSP hover content into a single markdown string for `vscode.Hover`. */
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

/** Strips VS Code hover markup to decide whether to show an empty hover at all. */
function markdownPlainTextFromHover(h: vscode.Hover): string {
  const raw = h.contents;
  const parts = Array.isArray(raw) ? raw : [raw];
  const chunks: string[] = [];
  for (const p of parts) {
    if (typeof p === "string") {
      chunks.push(p);
    } else {
      chunks.push(p.value);
    }
  }
  return chunks.join("\n");
}

/** Wraps LSP hover content as an untrusted `MarkdownString` for consistent preview behavior. */
function hoverToVs(h: LspHoverResult): vscode.Hover {
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

/**
 * Dependencies for registering providers—indirection keeps tests able to substitute clients and
 * avoids importing extension activation state from this module.
 */
export interface ForstLanguageFeaturesOptions {
  getClient: () => Promise<ForstHttpLspClient>;
  isTrackableForstDoc: (doc: vscode.TextDocument) => boolean;
  /** Receives warnings when a provider cannot obtain the LSP client or the request fails. */
  log: LogOutputChannel;
}

/**
 * Wires VS Code’s language service hooks to the subset of LSP methods the Forst server exposes over
 * HTTP so features light up incrementally as the compiler implements them.
 */
export function registerForstLanguageFeatures(
  context: vscode.ExtensionContext,
  options: ForstLanguageFeaturesOptions
): void {
  const { getClient, isTrackableForstDoc, log } = options;

  async function withDoc<T>(
    doc: vscode.TextDocument,
    empty: T,
    label: string,
    fn: (c: ForstHttpLspClient) => Promise<T>
  ): Promise<T> {
    if (!isTrackableForstDoc(doc)) {
      return empty;
    }
    try {
      const c = await getClient();
      return await fn(c);
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e);
      log.warn(`[${label}] ${doc.uri.fsPath}: ${msg}`);
      return empty;
    }
  }

  async function withClient<T>(
    empty: T,
    label: string,
    fn: (c: ForstHttpLspClient) => Promise<T>
  ): Promise<T> {
    try {
      const c = await getClient();
      return await fn(c);
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e);
      log.warn(`[${label}] ${msg}`);
      return empty;
    }
  }

  context.subscriptions.push(
    vscode.languages.registerHoverProvider(FORST_SELECTOR, {
      provideHover: async (doc, pos) =>
        withDoc(doc, null as vscode.Hover | null, "hover", async (c) => {
          const h = await c.hover({
            uri: doc.uri.toString(),
            line: pos.line,
            character: pos.character,
          });
          if (!h) {
            return null;
          }
          const vs = hoverToVs(h);
          if (markdownPlainTextFromHover(vs).trim() === "") {
            return null;
          }
          return vs;
        }),
    })
  );

  context.subscriptions.push(
    vscode.languages.registerCompletionItemProvider(
      FORST_SELECTOR,
      {
        provideCompletionItems: async (doc, pos, _token, ctx) =>
          withDoc(doc, [] as vscode.CompletionItem[], "completion", async (c) => {
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
            const items = list.items.map(completionItemToVs);
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
        withDoc(doc, undefined, "definition", async (c) => {
          const def = await c.definition({
            uri: doc.uri.toString(),
            line: pos.line,
            character: pos.character,
          });
          return definitionResultToVs(def);
        }),
    })
  );

  context.subscriptions.push(
    vscode.languages.registerReferenceProvider(FORST_SELECTOR, {
      provideReferences: async (doc, pos, ctx) =>
        withDoc(doc, [] as vscode.Location[], "references", async (c) => {
          const refs = await c.references({
            uri: doc.uri.toString(),
            line: pos.line,
            character: pos.character,
            includeDeclaration: ctx.includeDeclaration,
          });
          return refs.map(locationToVs);
        }),
    })
  );

  context.subscriptions.push(
    vscode.languages.registerDocumentSymbolProvider(FORST_SELECTOR, {
      provideDocumentSymbols: async (doc) =>
        withDoc(doc, [] as vscode.SymbolInformation[], "documentSymbol", async (c) => {
          const raw = await c.documentSymbol({ uri: doc.uri.toString() });
          return documentSymbolsToVs(raw);
        }),
    })
  );

  context.subscriptions.push(
    vscode.languages.registerWorkspaceSymbolProvider({
      provideWorkspaceSymbols: async (query) =>
        withClient([] as vscode.SymbolInformation[], "workspaceSymbol", async (c) => {
          const raw = await c.workspaceSymbol(query);
          const out: vscode.SymbolInformation[] = [];
          for (const item of raw) {
            const si = symbolInformationToVs(item);
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
        withDoc(doc, [] as vscode.TextEdit[], "formatting", async (c) => {
          const edits = await c.formatting({
            uri: doc.uri.toString(),
            tabSize: opt.tabSize,
            insertSpaces: opt.insertSpaces,
          });
          return textEditsToVs(edits);
        }),
    })
  );

  context.subscriptions.push(
    vscode.languages.registerCodeActionsProvider(FORST_SELECTOR, {
      provideCodeActions: async (doc, range, _ctx) =>
        withDoc(doc, [] as vscode.CodeAction[], "codeAction", async (c) => {
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
            const a = codeActionToVs(item);
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
        withDoc(doc, [] as vscode.CodeLens[], "codeLens", async (c) => {
          const raw = await c.codeLens({ uri: doc.uri.toString() });
          const out: vscode.CodeLens[] = [];
          for (const item of raw) {
            const lens = codeLensToVs(item);
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
        withDoc(doc, [] as vscode.FoldingRange[], "foldingRange", async (c) => {
          const raw = await c.foldingRange({ uri: doc.uri.toString() });
          const out: vscode.FoldingRange[] = [];
          for (const item of raw) {
            const fr = foldingRangeToVs(item);
            if (fr) {
              out.push(fr);
            }
          }
          return out;
        }),
    })
  );

}
