/**
 * Metadata for a single discovered Forst function from `GET /functions`.
 */
export interface FunctionInfo {
  /** Go/Forst package path segment (e.g. `main`). */
  package: string;
  /** Function identifier within the package. */
  name: string;
  /** Whether the executor advertises streaming for this function. */
  supportsStreaming: boolean;
  /** Serialized input type string from discovery. */
  inputType: string;
  /** Serialized output type string from discovery. */
  outputType: string;
  /** Positional parameters and their type strings. */
  parameters: ParameterInfo[];
  /** Return type string from discovery. */
  returnType: string;
  /** Source `.ft` path on disk (when provided by the server). */
  filePath: string;
}

/** One parameter in a {@link FunctionInfo} signature. */
export interface ParameterInfo {
  name: string;
  type: string;
}

/**
 * Request body for `POST /invoke` (non-streaming or streaming).
 */
export interface InvokeRequest {
  package: string;
  function: string;
  args: any;
  streaming?: boolean;
}

/**
 * JSON envelope returned by `POST /invoke`, `GET /health`-style RPC helpers, and related endpoints.
 */
export interface InvokeResponse<T extends any> {
  success: boolean;
  output?: string;
  error?: string;
  result?: T;
}

/** One NDJSON line from a streaming `POST /invoke` response. */
export interface StreamingResult {
  data: any;
  status: string;
  error?: string;
}

/** HTTP client options for `ForstSidecarClient`. */
export interface ForstClientConfig {
  /** Base URL of the Forst dev server (e.g. `http://localhost:8080`). */
  baseUrl: string;
  /** Per-request timeout in ms. */
  timeout?: number;
  /** Retries after transient failures (exponential backoff). */
  retries?: number;
}

/** Configuration for `ForstSidecar` and `ForstServer`. */
export interface ForstConfig {
  mode?: "development" | "production" | "testing";
  /** Directory containing `.ft` sources (watch + hot reload). Falls back to `rootDir` or project root. */
  forstDir?: string;
  /** Optional output directory hint for tooling (reserved). */
  outputDir?: string;
  /** Override project root for `forst dev -root` when you need it different from `forstDir`. */
  rootDir?: string;
  /** TCP port for the embedded HTTP dev server. */
  port?: number;
  /** Bind host for the dev server. */
  host?: string;
  logLevel?: "debug" | "info" | "warn" | "error";
  /** Reserved for future non-HTTP transports (IPC, etc.). Not implemented; HTTP only today. */
  transports?: {
    development?: TransportConfig;
    production?: TransportConfig;
    testing?: TransportConfig;
  };
}

/**
 * Transport options under {@link ForstConfig.transports} (future use; HTTP is the only implemented mode).
 */
export interface TransportConfig {
  mode: "http" | "ipc" | "direct";
  http?: {
    port?: number;
    cors?: boolean;
    healthCheck?: string;
  };
  ipc?: {
    socketPath?: string;
    permissions?: number;
  };
}

/** Resolved compiler binary metadata from `ForstUtils.getCompilerInfo`. */
export interface CompilerInfo {
  version: string;
  path: string;
  platform: string;
  arch: string;
}

/**
 * Result shape for discovery workflows (reserved; not all APIs return this yet).
 */
export interface DiscoveryResult {
  functions: FunctionInfo[];
  packages: string[];
  errors: string[];
}

/**
 * Split TypeScript codegen fragments (reserved for future `forst generate` integration).
 */
export interface TypeGenerationResult {
  interfaces: string;
  functions: string;
  errors: string[];
}

/**
 * Snapshot of the embedded `forst dev` process: listen address and lifecycle status.
 */
export interface ServerInfo {
  pid: number;
  port: number;
  host: string;
  status: "starting" | "running" | "stopped" | "error";
}
