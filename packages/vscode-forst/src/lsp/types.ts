/**
 * Minimal LSP-shaped types matching what the Forst HTTP server returns, without pulling in
 * vscode-languageclient or a full protocol package—keeps the extension lightweight and aligned
 * with the compiler’s wire format.
 */

/** Diagnostic shape used when applying `publishDiagnostics`-style payloads from HTTP responses. */
export interface LspDiagnostic {
  range: {
    start: { line: number; character: number };
    end: { line: number; character: number };
  };
  severity?: number;
  message: string;
  source?: string;
  code?: string;
}

/** Batch notification payload for syncing editor diagnostics with server results. */
export interface PublishDiagnosticsParams {
  uri: string;
  diagnostics: LspDiagnostic[];
}

/** MarkedString-style fragment (`language` + `value`) or markup (`kind` + `value`). */
export interface LspMarkedString {
  language?: string;
  value: string;
}

/** Union of hover content forms the server may emit before VS Code normalization. */
export type LspHoverContents =
  | string
  | LspMarkedString
  | { kind: string; value: string }
  | (string | LspMarkedString)[];

/** Hover payload subset used when wiring `textDocument/hover` into `vscode.Hover`. */
export interface LspHoverResult {
  contents: LspHoverContents;
  range?: {
    start: { line: number; character: number };
    end: { line: number; character: number };
  };
}

/** Zero-based line/character pair as emitted by the LSP wire format (not VS Code offsets). */
export interface LspPosition {
  line: number;
  character: number;
}

/** Inclusive span in LSP coordinates—used for edits, hovers, and locations before conversion. */
export interface LspRange {
  start: LspPosition;
  end: LspPosition;
}

/** File URI plus range for definitions, references, and workspace symbols. */
export interface LspLocation {
  uri: string;
  range: LspRange;
}

/** Single textual replacement within a range for formatting and quick-fix application. */
export interface LspTextEdit {
  range: LspRange;
  newText: string;
}

/** Completion entry as returned by `textDocument/completion` before UI mapping. */
export interface LspCompletionItem {
  label: string;
  kind?: number;
  detail?: string;
  documentation?: string;
  sortText?: string;
  filterText?: string;
  insertText?: string;
  insertTextFormat?: number;
}

/** Batch result flagging whether more items exist on the server for incremental fetch scenarios. */
export interface LspCompletionList {
  isIncomplete?: boolean;
  items: LspCompletionItem[];
}
