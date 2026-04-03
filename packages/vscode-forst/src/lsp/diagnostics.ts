import type { LogOutputChannel } from "vscode";
import * as vscode from "vscode";
import type { LspDiagnostic, PublishDiagnosticsParams } from "./types";

/** Coordinates that are negative or non-finite are clamped so `vscode.Range` construction never throws. */
function clampNonNeg(n: number): number {
  if (!Number.isFinite(n) || n < 0) {
    return 0;
  }
  return Math.floor(n);
}

/**
 * Maps compiler/LSP severities into VS Code’s presentation so Problems panel sorting and glyphs
 * match user expectations from other languages.
 */
export function lspDiagnosticToVs(d: LspDiagnostic): vscode.Diagnostic {
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
 * Filters legacy “phase completed” lines that older forst builds incorrectly surfaced as
 * diagnostics—keeps the Problems list trustworthy until every deployment runs a fixed server.
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

/**
 * Applies a publish-diagnostics payload to a VS Code collection, applying the milestone filter
 * and normalizing ranges in one place for open/change/close handlers.
 */
export function applyPublishDiagnostics(
  collection: vscode.DiagnosticCollection,
  pub: PublishDiagnosticsParams,
  log?: LogOutputChannel
): void {
  const uri = vscode.Uri.parse(pub.uri);
  const raw = pub.diagnostics ?? [];
  const noise = raw.filter((d) => isPhaseMilestoneNoise(d));
  if (noise.length > 0) {
    log?.debug(
      `[diagnostics] dropped ${noise.length} phase-milestone message(s) for ${pub.uri}`
    );
  }
  const vs = raw
    .filter((d) => !isPhaseMilestoneNoise(d))
    .map(lspDiagnosticToVs);
  collection.set(uri, vs);
}
