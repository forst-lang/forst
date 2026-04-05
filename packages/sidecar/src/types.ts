export interface FunctionInfo {
  package: string;
  name: string;
  supportsStreaming: boolean;
  inputType: string;
  outputType: string;
  parameters: ParameterInfo[];
  returnType: string;
  filePath: string;
}

export interface ParameterInfo {
  name: string;
  type: string;
}

export interface InvokeRequest {
  package: string;
  function: string;
  args: any;
  streaming?: boolean;
}

export interface InvokeResponse<T extends any> {
  success: boolean;
  output?: string;
  error?: string;
  result?: T;
}

export interface StreamingResult {
  data: any;
  status: string;
  error?: string;
}

export interface ForstClientConfig {
  baseUrl: string;
  timeout?: number;
  retries?: number;
}

export interface ForstConfig {
  mode?: "development" | "production" | "testing";
  /** Directory containing `.ft` sources (watch + hot reload). Falls back to `rootDir` or project root. */
  forstDir?: string;
  outputDir?: string;
  /** Override project root for `forst dev -root` when you need it different from `forstDir`. */
  rootDir?: string;
  port?: number;
  host?: string;
  logLevel?: "debug" | "info" | "warn" | "error";
  /** Reserved for future non-HTTP transports (IPC, etc.). Not implemented; HTTP only today. */
  transports?: {
    development?: TransportConfig;
    production?: TransportConfig;
    testing?: TransportConfig;
  };
}

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

export interface CompilerInfo {
  version: string;
  path: string;
  platform: string;
  arch: string;
}

export interface DiscoveryResult {
  functions: FunctionInfo[];
  packages: string[];
  errors: string[];
}

export interface TypeGenerationResult {
  interfaces: string;
  functions: string;
  errors: string[];
}

export interface ServerInfo {
  pid: number;
  port: number;
  host: string;
  status: "starting" | "running" | "stopped" | "error";
}
