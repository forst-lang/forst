import type { LogOutputChannel } from "vscode";
import {
  LspHttpResponseError,
  LspJsonRpcResponseError,
  LspMalformedResponseError,
} from "./errors";
import type {
  LspCompletionItem,
  LspCompletionList,
  LspHoverResult,
  LspLocation,
  LspRange,
  LspTextEdit,
  PublishDiagnosticsParams,
} from "./types";

/** Wire-level JSON-RPC error object when `result` is absent. */
interface JsonRpcError {
  code: number;
  message: string;
}

/** Parsed POST body shape before branching on `result` vs `error`. */
interface JsonRpcResponse {
  jsonrpc: string;
  id?: number | string | null;
  result?: unknown;
  error?: JsonRpcError;
}

/**
 * Thin JSON-RPC over HTTP to the Forst compiler’s LSP mode—lets the extension stay in-process
 * while reusing the same methods as a full LSP client would call.
 */
export class ForstHttpLspClient {
  private nextId = 1;

  constructor(
    private readonly baseUrl: string,
    private readonly log?: LogOutputChannel
  ) {}

  /** Stable base URL for comparing reconnects and health checks. */
  get base(): string {
    return this.baseUrl;
  }

  private async post(method: string, params: unknown): Promise<unknown> {
    const id = this.nextId++;
    const body = {
      jsonrpc: "2.0",
      id,
      method,
      params,
    };
    let res: Response;
    let text: string;
    try {
      res = await fetch(`${this.baseUrl.replace(/\/$/, "")}/`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      text = await res.text();
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e);
      this.log?.error(`LSP ${method}: network error — ${msg}`);
      throw e;
    }
    if (!res.ok) {
      const err = new LspHttpResponseError(res.status, text.slice(0, 200));
      this.log?.error(`LSP ${method}: HTTP ${res.status} — ${err.message}`);
      throw err;
    }
    let json: JsonRpcResponse;
    try {
      json = JSON.parse(text) as JsonRpcResponse;
    } catch (cause) {
      const err = new LspMalformedResponseError(text.slice(0, 200), cause);
      this.log?.error(`LSP ${method}: invalid JSON — ${err.message}`);
      throw err;
    }
    if (json.error) {
      const err = new LspJsonRpcResponseError(
        json.error.code,
        json.error.message ?? ""
      );
      this.log?.error(
        `LSP ${method}: JSON-RPC ${json.error.code} — ${err.message}`
      );
      throw err;
    }
    return json.result;
  }

  async initialize(rootUri: string | null): Promise<void> {
    await this.post("initialize", {
      processId: null,
      rootUri,
      capabilities: {},
      clientInfo: { name: "vscode-forst", version: "0.1.0" },
    });
  }

  async didOpen(params: {
    uri: string;
    version: number;
    text: string;
  }): Promise<PublishDiagnosticsParams> {
    const result = await this.post("textDocument/didOpen", {
      textDocument: {
        uri: params.uri,
        languageId: "forst",
        version: params.version,
        text: params.text,
      },
    });
    return result as PublishDiagnosticsParams;
  }

  async didChange(params: {
    uri: string;
    version: number;
    text: string;
  }): Promise<PublishDiagnosticsParams> {
    const result = await this.post("textDocument/didChange", {
      textDocument: {
        uri: params.uri,
        version: params.version,
      },
      contentChanges: [{ text: params.text }],
    });
    return result as PublishDiagnosticsParams;
  }

  async didClose(uri: string): Promise<PublishDiagnosticsParams> {
    const result = await this.post("textDocument/didClose", {
      textDocument: { uri },
    });
    return result as PublishDiagnosticsParams;
  }

  async hover(params: {
    uri: string;
    line: number;
    character: number;
  }): Promise<LspHoverResult | null> {
    const result = await this.post("textDocument/hover", {
      textDocument: { uri: params.uri },
      position: { line: params.line, character: params.character },
    });
    if (result == null) {
      return null;
    }
    return result as LspHoverResult;
  }

  async completion(params: {
    uri: string;
    line: number;
    character: number;
    triggerKind?: number;
    triggerCharacter?: string | null;
  }): Promise<LspCompletionList | null> {
    const result = await this.post("textDocument/completion", {
      textDocument: { uri: params.uri },
      position: { line: params.line, character: params.character },
      context: {
        triggerKind: params.triggerKind ?? 1,
        triggerCharacter: params.triggerCharacter ?? undefined,
      },
    });
    if (result == null) {
      return null;
    }
    const list = result as { isIncomplete?: boolean; items?: LspCompletionItem[] };
    return {
      isIncomplete: list.isIncomplete,
      items: list.items ?? [],
    };
  }

  async definition(params: {
    uri: string;
    line: number;
    character: number;
  }): Promise<LspLocation | LspLocation[] | null> {
    const result = await this.post("textDocument/definition", {
      textDocument: { uri: params.uri },
      position: { line: params.line, character: params.character },
    });
    if (result == null) {
      return null;
    }
    return result as LspLocation | LspLocation[];
  }

  async references(params: {
    uri: string;
    line: number;
    character: number;
    includeDeclaration: boolean;
  }): Promise<LspLocation[]> {
    const result = await this.post("textDocument/references", {
      textDocument: { uri: params.uri },
      position: { line: params.line, character: params.character },
      context: { includeDeclaration: params.includeDeclaration },
    });
    if (result == null || !Array.isArray(result)) {
      return [];
    }
    return result as LspLocation[];
  }

  async documentSymbol(params: { uri: string }): Promise<unknown[]> {
    const result = await this.post("textDocument/documentSymbol", {
      textDocument: { uri: params.uri },
    });
    if (result == null || !Array.isArray(result)) {
      return [];
    }
    return result;
  }

  async workspaceSymbol(query: string): Promise<unknown[]> {
    const result = await this.post("workspace/symbol", { query });
    if (result == null || !Array.isArray(result)) {
      return [];
    }
    return result;
  }

  async formatting(params: {
    uri: string;
    tabSize: number;
    insertSpaces: boolean;
  }): Promise<LspTextEdit[] | null> {
    const result = await this.post("textDocument/formatting", {
      textDocument: { uri: params.uri },
      options: {
        tabSize: params.tabSize,
        insertSpaces: params.insertSpaces,
      },
    });
    if (result == null) {
      return null;
    }
    if (!Array.isArray(result)) {
      return null;
    }
    return result as LspTextEdit[];
  }

  async codeAction(params: {
    uri: string;
    range: LspRange;
    diagnostics?: unknown[];
  }): Promise<unknown[]> {
    const result = await this.post("textDocument/codeAction", {
      textDocument: { uri: params.uri },
      range: params.range,
      context: { diagnostics: params.diagnostics ?? [] },
    });
    if (result == null || !Array.isArray(result)) {
      return [];
    }
    return result;
  }

  async codeLens(params: { uri: string }): Promise<unknown[]> {
    const result = await this.post("textDocument/codeLens", {
      textDocument: { uri: params.uri },
    });
    if (result == null || !Array.isArray(result)) {
      return [];
    }
    return result;
  }

  async foldingRange(params: { uri: string }): Promise<unknown[]> {
    const result = await this.post("textDocument/foldingRange", {
      textDocument: { uri: params.uri },
    });
    if (result == null || !Array.isArray(result)) {
      return [];
    }
    return result;
  }
}
