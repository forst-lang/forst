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

export interface InvokeResponse {
  success: boolean;
  output?: string;
  error?: string;
  result?: any;
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
  forstDir?: string;
  outputDir?: string;
  rootDir?: string;
  port?: number;
  host?: string;
  logLevel?: "debug" | "info" | "warn" | "error";
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
