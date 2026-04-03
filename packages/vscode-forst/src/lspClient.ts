/** LSP diagnostic subset used from forst HTTP responses */
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

export interface PublishDiagnosticsParams {
  uri: string;
  diagnostics: LspDiagnostic[];
}

/** LSP MarkedString (`language` + `value`) or MarkupContent (`kind` + `value`). */
export interface LspMarkedString {
  language?: string;
  value: string;
}

export type LspHoverContents =
  | string
  | LspMarkedString
  | { kind: string; value: string }
  | (string | LspMarkedString)[];

/** Subset of LSP Hover result returned by forst HTTP LSP. */
export interface LspHoverResult {
  contents: LspHoverContents;
  range?: {
    start: { line: number; character: number };
    end: { line: number; character: number };
  };
}

export interface LspPosition {
  line: number;
  character: number;
}

export interface LspRange {
  start: LspPosition;
  end: LspPosition;
}

export interface LspLocation {
  uri: string;
  range: LspRange;
}

export interface LspTextEdit {
  range: LspRange;
  newText: string;
}

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

export interface LspCompletionList {
  isIncomplete?: boolean;
  items: LspCompletionItem[];
}

interface JsonRpcError {
  code: number;
  message: string;
}

interface JsonRpcResponse {
  jsonrpc: string;
  id?: number | string | null;
  result?: unknown;
  error?: JsonRpcError;
}

/** Base class for Forst HTTP LSP client failures. */
export class LspClientError extends Error {
  constructor(message: string, options?: ErrorOptions) {
    super(message, options);
    this.name = "LspClientError";
    Object.setPrototypeOf(this, new.target.prototype);
  }
}

/** HTTP response status was not OK. */
export class LspHttpResponseError extends LspClientError {
  constructor(
    readonly status: number,
    readonly bodySnippet: string
  ) {
    super(`LSP HTTP ${status}: ${bodySnippet}`);
    this.name = "LspHttpResponseError";
  }
}

/** Response body was not valid JSON. */
export class LspMalformedResponseError extends LspClientError {
  constructor(
    readonly rawBodySnippet: string,
    cause: unknown
  ) {
    super(`Invalid JSON from LSP: ${rawBodySnippet}`, { cause });
    this.name = "LspMalformedResponseError";
  }
}

/** JSON-RPC error object was present in the response. */
export class LspJsonRpcResponseError extends LspClientError {
  constructor(
    readonly rpcCode: number,
    rpcMessage: string
  ) {
    super(
      rpcMessage.trim().length > 0 ? rpcMessage : `LSP error ${rpcCode}`
    );
    this.name = "LspJsonRpcResponseError";
  }
}

/** LSP /health did not become ready within the timeout. */
export class LspHealthTimeoutError extends LspClientError {
  constructor(
    readonly healthUrl: string,
    readonly lastError: string | undefined
  ) {
    super(
      `Forst LSP did not respond on ${healthUrl} (${lastError ?? "unknown"})`
    );
    this.name = "LspHealthTimeoutError";
  }
}

export class ForstHttpLspClient {
  private nextId = 1;

  constructor(private readonly baseUrl: string) {}

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
    const res = await fetch(`${this.baseUrl.replace(/\/$/, "")}/`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
    const text = await res.text();
    if (!res.ok) {
      throw new LspHttpResponseError(res.status, text.slice(0, 200));
    }
    let json: JsonRpcResponse;
    try {
      json = JSON.parse(text) as JsonRpcResponse;
    } catch (cause) {
      throw new LspMalformedResponseError(text.slice(0, 200), cause);
    }
    if (json.error) {
      throw new LspJsonRpcResponseError(
        json.error.code,
        json.error.message ?? ""
      );
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
