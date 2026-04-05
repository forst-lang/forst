import * as vscode from "vscode";
import type {
  LspCompletionItem,
  LspLocation,
  LspPosition,
  LspRange,
  LspTextEdit,
} from "./types";

/** Bridges LSP 0-based integer positions to VS Code’s `Position` without trusting negative inputs. */
export function positionToVs(p: LspPosition): vscode.Position {
  return new vscode.Position(
    Math.max(0, Math.floor(p.line)),
    Math.max(0, Math.floor(p.character))
  );
}

/** Converts an LSP range to a VS Code `Range` for selections, hovers, and edits. */
export function rangeToVs(r: LspRange): vscode.Range {
  return new vscode.Range(
    positionToVs(r.start),
    positionToVs(r.end)
  );
}

/** Turns a single LSP location into a navigable VS Code `Location` for go-to-definition. */
export function locationToVs(loc: LspLocation): vscode.Location {
  return new vscode.Location(
    vscode.Uri.parse(loc.uri),
    rangeToVs(loc.range)
  );
}

/**
 * Normalizes definition responses that may be a single location or a list—mirrors how VS Code
 * providers accept either shape from the language service.
 */
export function definitionResultToVs(
  r: LspLocation | LspLocation[] | null | undefined
): vscode.Location | vscode.Location[] | undefined {
  if (r == null) {
    return undefined;
  }
  if (Array.isArray(r)) {
    return r.map(locationToVs);
  }
  return locationToVs(r);
}

/** Applies LSP text edits in document order through VS Code’s edit API. */
export function textEditsToVs(
  edits: LspTextEdit[] | null | undefined
): vscode.TextEdit[] {
  if (edits == null || edits.length === 0) {
    return [];
  }
  return edits.map(
    (e) => new vscode.TextEdit(rangeToVs(e.range), e.newText)
  );
}

/** Maps LSP `CompletionItemKind` numeric codes to VS Code icons where both define overlapping enums. */
export function completionKindToVs(kind: number | undefined): vscode.CompletionItemKind {
  if (kind == null || !Number.isFinite(kind)) {
    return vscode.CompletionItemKind.Text;
  }
  const k = Math.floor(kind);
  const map: Record<number, vscode.CompletionItemKind> = {
    1: vscode.CompletionItemKind.Text,
    2: vscode.CompletionItemKind.Method,
    3: vscode.CompletionItemKind.Function,
    4: vscode.CompletionItemKind.Constructor,
    5: vscode.CompletionItemKind.Field,
    6: vscode.CompletionItemKind.Variable,
    7: vscode.CompletionItemKind.Class,
    8: vscode.CompletionItemKind.Interface,
    9: vscode.CompletionItemKind.Module,
    10: vscode.CompletionItemKind.Property,
    12: vscode.CompletionItemKind.Value,
    13: vscode.CompletionItemKind.Enum,
    14: vscode.CompletionItemKind.Keyword,
    15: vscode.CompletionItemKind.Snippet,
    22: vscode.CompletionItemKind.Struct,
  };
  return map[k] ?? vscode.CompletionItemKind.Text;
}

/** Builds a `CompletionItem` with trusted-documentation flags aligned with the extension’s security posture. */
export function completionItemToVs(item: LspCompletionItem): vscode.CompletionItem {
  const ci = new vscode.CompletionItem(
    item.label,
    completionKindToVs(item.kind)
  );
  ci.detail = item.detail;
  if (item.documentation !== undefined && item.documentation !== "") {
    ci.documentation = new vscode.MarkdownString(item.documentation);
    (ci.documentation as vscode.MarkdownString).isTrusted = false;
  }
  if (item.sortText !== undefined) {
    ci.sortText = item.sortText;
  }
  if (item.filterText !== undefined) {
    ci.filterText = item.filterText;
  }
  if (item.insertText !== undefined) {
    ci.insertText = item.insertText;
  }
  if (item.insertTextFormat === 2) {
    ci.insertText = new vscode.SnippetString(item.insertText ?? item.label);
  }
  return ci;
}

/** Aligns LSP `SymbolKind` with VS Code outline icons for a consistent breadcrumb and symbol view. */
export function symbolKindToVs(kind: number | undefined): vscode.SymbolKind {
  if (kind == null || !Number.isFinite(kind)) {
    return vscode.SymbolKind.Null;
  }
  const k = Math.floor(kind);
  const map: Record<number, vscode.SymbolKind> = {
    1: vscode.SymbolKind.File,
    2: vscode.SymbolKind.Module,
    3: vscode.SymbolKind.Namespace,
    4: vscode.SymbolKind.Package,
    5: vscode.SymbolKind.Class,
    6: vscode.SymbolKind.Method,
    7: vscode.SymbolKind.Property,
    8: vscode.SymbolKind.Field,
    9: vscode.SymbolKind.Constructor,
    10: vscode.SymbolKind.Enum,
    11: vscode.SymbolKind.Interface,
    12: vscode.SymbolKind.Function,
    13: vscode.SymbolKind.Variable,
    14: vscode.SymbolKind.Constant,
    15: vscode.SymbolKind.String,
    16: vscode.SymbolKind.Number,
    17: vscode.SymbolKind.Boolean,
    18: vscode.SymbolKind.Array,
    19: vscode.SymbolKind.Object,
    20: vscode.SymbolKind.Key,
    21: vscode.SymbolKind.Null,
    22: vscode.SymbolKind.EnumMember,
    23: vscode.SymbolKind.Struct,
    24: vscode.SymbolKind.Event,
    25: vscode.SymbolKind.Operator,
    26: vscode.SymbolKind.TypeParameter,
  };
  return map[k] ?? vscode.SymbolKind.Object;
}

/** Narrow structural type for loose `documentSymbol` JSON before validation. */
export interface SymbolInformationLike {
  name: string;
  kind?: number;
  location: LspLocation;
  containerName?: string;
}

/** Best-effort conversion of one raw symbol entry; skips entries that are not object-shaped. */
export function symbolInformationToVs(raw: unknown): vscode.SymbolInformation | undefined {
  if (raw == null || typeof raw !== "object") {
    return undefined;
  }
  const o = raw as SymbolInformationLike;
  if (typeof o.name !== "string" || o.location == null) {
    return undefined;
  }
  return new vscode.SymbolInformation(
    o.name,
    symbolKindToVs(o.kind),
    o.containerName ?? "",
    locationToVs(o.location)
  );
}

/**
 * Flattens symbol arrays returned by the server into `SymbolInformation` until hierarchical
 * `DocumentSymbol` support is warranted by the wire format.
 */
export function documentSymbolsToVs(
  raw: unknown[]
): vscode.DocumentSymbol[] | vscode.SymbolInformation[] {
  if (raw.length === 0) {
    return [];
  }
  const infos: vscode.SymbolInformation[] = [];
  for (const item of raw) {
    const si = symbolInformationToVs(item);
    if (si) {
      infos.push(si);
    }
  }
  return infos;
}

/** Narrow structural type for folding range payloads. */
export interface FoldingRangeLike {
  startLine: number;
  endLine: number;
  kind?: string;
}

/** Maps optional LSP folding kinds to VS Code’s built-in folding range kinds for gutter affordances. */
export function foldingRangeToVs(raw: unknown): vscode.FoldingRange | undefined {
  if (raw == null || typeof raw !== "object") {
    return undefined;
  }
  const o = raw as FoldingRangeLike;
  if (!Number.isFinite(o.startLine) || !Number.isFinite(o.endLine)) {
    return undefined;
  }
  const fr = new vscode.FoldingRange(
    Math.max(0, Math.floor(o.startLine)),
    Math.max(0, Math.floor(o.endLine))
  );
  if (o.kind === "comment") {
    fr.kind = vscode.FoldingRangeKind.Comment;
  } else if (o.kind === "imports") {
    fr.kind = vscode.FoldingRangeKind.Imports;
  } else if (o.kind === "region") {
    fr.kind = vscode.FoldingRangeKind.Region;
  }
  return fr;
}

/** Narrow structural type for code lens responses. */
export interface CodeLensLike {
  range: LspRange;
  command?: { title: string; command: string; arguments?: unknown[] };
}

/** Instantiates a `CodeLens` with an optional command for inline actions when the server provides them. */
export function codeLensToVs(raw: unknown): vscode.CodeLens | undefined {
  if (raw == null || typeof raw !== "object") {
    return undefined;
  }
  const o = raw as CodeLensLike;
  if (o.range == null) {
    return undefined;
  }
  const lens = new vscode.CodeLens(rangeToVs(o.range));
  if (o.command != null && typeof o.command.title === "string") {
    lens.command = {
      title: o.command.title,
      command: o.command.command,
      arguments: o.command.arguments,
    };
  }
  return lens;
}

/** Narrow structural type for code action payloads. */
export interface CodeActionLike {
  title: string;
  kind?: string;
  command?: { title: string; command: string; arguments?: unknown[] };
  edit?: { changes?: Record<string, LspTextEdit[]> };
}

/** Converts an LSP workspace edit (`changes` map) to VS Code’s `WorkspaceEdit`. */
export function workspaceEditFromLsp(raw: unknown): vscode.WorkspaceEdit | undefined {
  if (raw == null || typeof raw !== "object") {
    return undefined;
  }
  const o = raw as { changes?: Record<string, LspTextEdit[]> };
  if (o.changes == null || typeof o.changes !== "object") {
    return undefined;
  }
  const ws = new vscode.WorkspaceEdit();
  for (const [uriStr, edits] of Object.entries(o.changes)) {
    const uri = vscode.Uri.parse(uriStr);
    for (const te of edits) {
      ws.replace(uri, rangeToVs(te.range), te.newText);
    }
  }
  return ws;
}

/** Produces a `CodeAction` suitable for the lightbulb menu, including kind and embedded commands. */
export function codeActionToVs(raw: unknown): vscode.CodeAction | undefined {
  if (raw == null || typeof raw !== "object") {
    return undefined;
  }
  const o = raw as CodeActionLike;
  if (typeof o.title !== "string") {
    return undefined;
  }
  const a = new vscode.CodeAction(o.title);
  if (typeof o.kind === "string") {
    if (o.kind === "source.formatDocument") {
      a.kind = vscode.CodeActionKind.Source.append("formatDocument");
    } else {
      a.kind = vscode.CodeActionKind.Empty.append(o.kind);
    }
  }
  if (o.command != null && typeof o.command.title === "string") {
    a.command = {
      title: o.command.title,
      command: o.command.command,
      arguments: o.command.arguments,
    };
  }
  if (o.edit?.changes != null) {
    const ws = workspaceEditFromLsp(o.edit);
    if (ws !== undefined) {
      a.edit = ws;
    }
  }
  return a;
}
