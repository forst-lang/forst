/**
 * Minimal surface used by generated *.client.ts and client/index.ts.
 * Keeps `tsc --noEmit` working without installing the real @forst/sidecar package.
 */
export interface ForstSidecarClientConfig {
  baseUrl?: string;
  timeout?: number;
  retries?: number;
}

export interface InvokeFunctionResult<T> {
  success: boolean;
  error?: string;
  result: T;
}

export class ForstSidecarClient {
  constructor(config?: ForstSidecarClientConfig);
  invokeFunction<T>(
    packageName: string,
    functionName: string,
    args: unknown
  ): Promise<InvokeFunctionResult<T>>;
}
