import * as vscode from "vscode";
import type {
  LspCompletionItem,
  LspLocation,
  LspPosition,
  LspRange,
  LspTextEdit,
} from "./lspClient";

export function lspPositionToVs(p: LspPosition): vscode.Position {
  return new vscode.Position(
    Math.max(0, Math.floor(p.line)),
    Math.max(0, Math.floor(p.character))
  );
}

export function lspRangeToVs(r: LspRange): vscode.Range {
  return new vscode.Range(
    lspPositionToVs(r.start),
    lspPositionToVs(r.end)
  );
}

export function lspLocationToVs(loc: LspLocation): vscode.Location {
  return new vscode.Location(
    vscode.Uri.parse(loc.uri),
    lspRangeToVs(loc.range)
  );
}

export function lspDefinitionResultToVs(
  r: LspLocation | LspLocation[] | null | undefined
): vscode.Location | vscode.Location[] | undefined {
  if (r == null) {
    return undefined;
  }
  if (Array.isArray(r)) {
    return r.map(lspLocationToVs);
  }
  return lspLocationToVs(r);
}

export function lspTextEditsToVs(
  edits: LspTextEdit[] | null | undefined
): vscode.TextEdit[] {
  if (edits == null || edits.length === 0) {
    return [];
  }
  return edits.map(
    (e) => new vscode.TextEdit(lspRangeToVs(e.range), e.newText)
  );
}

/** Map LSP CompletionItemKind to vscode.CompletionItemKind (subset). */
export function lspCompletionKindToVs(kind: number | undefined): vscode.CompletionItemKind {
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

export function lspCompletionItemToVs(item: LspCompletionItem): vscode.CompletionItem {
  const ci = new vscode.CompletionItem(
    item.label,
    lspCompletionKindToVs(item.kind)
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

/** Best-effort: LSP SymbolKind → vscode.SymbolKind */
export function lspSymbolKindToVs(kind: number | undefined): vscode.SymbolKind {
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

export interface LspSymbolInformationLike {
  name: string;
  kind?: number;
  location: LspLocation;
  containerName?: string;
}

export function lspSymbolInformationToVs(raw: unknown): vscode.SymbolInformation | undefined {
  if (raw == null || typeof raw !== "object") {
    return undefined;
  }
  const o = raw as LspSymbolInformationLike;
  if (typeof o.name !== "string" || o.location == null) {
    return undefined;
  }
  return new vscode.SymbolInformation(
    o.name,
    lspSymbolKindToVs(o.kind),
    o.containerName ?? "",
    lspLocationToVs(o.location)
  );
}

/** Map document symbols: prefers flat SymbolInformation; hierarchical DocumentSymbol can be added when the LSP emits it. */
export function lspDocumentSymbolsToVs(
  raw: unknown[]
): vscode.DocumentSymbol[] | vscode.SymbolInformation[] {
  if (raw.length === 0) {
    return [];
  }
  const infos: vscode.SymbolInformation[] = [];
  for (const item of raw) {
    const si = lspSymbolInformationToVs(item);
    if (si) {
      infos.push(si);
    }
  }
  return infos;
}

export interface LspFoldingRangeLike {
  startLine: number;
  endLine: number;
  kind?: string;
}

export function lspFoldingRangeToVs(raw: unknown): vscode.FoldingRange | undefined {
  if (raw == null || typeof raw !== "object") {
    return undefined;
  }
  const o = raw as LspFoldingRangeLike;
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

export interface LspCodeLensLike {
  range: LspRange;
  command?: { title: string; command: string; arguments?: unknown[] };
}

export function lspCodeLensToVs(raw: unknown): vscode.CodeLens | undefined {
  if (raw == null || typeof raw !== "object") {
    return undefined;
  }
  const o = raw as LspCodeLensLike;
  if (o.range == null) {
    return undefined;
  }
  const lens = new vscode.CodeLens(lspRangeToVs(o.range));
  if (o.command != null && typeof o.command.title === "string") {
    lens.command = {
      title: o.command.title,
      command: o.command.command,
      arguments: o.command.arguments,
    };
  }
  return lens;
}

export interface LspCodeActionLike {
  title: string;
  kind?: string;
  command?: { title: string; command: string; arguments?: unknown[] };
}

export function lspCodeActionToVs(raw: unknown): vscode.CodeAction | undefined {
  if (raw == null || typeof raw !== "object") {
    return undefined;
  }
  const o = raw as LspCodeActionLike;
  if (typeof o.title !== "string") {
    return undefined;
  }
  const a = new vscode.CodeAction(o.title);
  if (typeof o.kind === "string") {
    a.kind = vscode.CodeActionKind.Empty.append(o.kind);
  }
  if (o.command != null && typeof o.command.title === "string") {
    a.command = {
      title: o.command.title,
      command: o.command.command,
      arguments: o.command.arguments,
    };
  }
  return a;
}
