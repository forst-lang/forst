/**
 * Minimal surface used by generated client/index.ts and *.client.ts.
 * Keeps `tsc --noEmit` working without installing the real @forst/client package.
 */
export type InvokeSuccess<T> = {
  success: true;
  result: T;
  output?: string;
  error?: string;
};

export type ServerVersionInfo = {
  contractVersion: string;
  version?: string;
  runtime?: string;
};

export interface ForstInvokeClient {
  invokeFunction<T>(
    packageName: string,
    functionName: string,
    args: unknown[]
  ): Promise<InvokeSuccess<T>>;
  invokeStream<T>(
    packageName: string,
    functionName: string,
    args: unknown[]
  ): AsyncGenerator<T, void, unknown>;
  healthCheck(): Promise<boolean>;
  getVersion(): Promise<ServerVersionInfo>;
}

export interface ForstInvokeClientConfig {
  baseUrl?: string;
  timeout?: number;
  retries?: number;
  transport?: "auto" | "http" | "dev";
  sidecarRuntime?: "spawn" | "connect";
  devServerUrl?: string;
}

export interface ForstClientConfig extends ForstInvokeClientConfig {}

export function createInvokeClient(
  config?: ForstInvokeClientConfig
): ForstInvokeClient;

export function getDefaultInvokeClient(): ForstInvokeClient;
