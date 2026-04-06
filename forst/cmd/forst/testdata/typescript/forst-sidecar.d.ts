/**
 * Minimal surface used by generated *.client.ts and client/index.ts.
 * Keeps `tsc --noEmit` working without installing the real @forst/sidecar package.
 */
export interface ForstSidecarClientConfig {
  baseUrl?: string;
  timeout?: number;
  retries?: number;
}

/** Successful `POST /invoke` (matches @forst/sidecar InvokeSuccess). */
export type InvokeSuccess<T> = {
  success: true;
  result: T;
  output?: string;
  error?: string;
};

export class ForstSidecarClient {
  constructor(config?: ForstSidecarClientConfig);
  invokeFunction<T>(
    packageName: string,
    functionName: string,
    args: unknown
  ): Promise<InvokeSuccess<T>>;
}
