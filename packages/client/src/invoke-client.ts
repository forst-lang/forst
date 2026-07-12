import { ForstSidecar, ForstSidecarClient, readInvokeReadyUrl } from "@forst/sidecar";
import type {
  ForstSidecar as ForstSidecarType,
  InvokeSuccess,
  ServerVersionInfo,
  StreamingResult,
} from "@forst/sidecar";

async function* invokeStreamRows<T>(
  gen: AsyncGenerator<StreamingResult & { data?: T }, void, undefined>
): AsyncGenerator<T, void, unknown> {
  for await (const row of gen) {
    if (row.data !== undefined) {
      yield row.data;
    }
  }
}

export interface ForstInvokeClientConfig {
  baseUrl?: string;
  timeout?: number;
  retries?: number;
  transport?: "auto" | "http" | "dev";
  sidecarRuntime?: "spawn" | "connect";
  devServerUrl?: string;
  port?: number;
  host?: string;
  rootDir?: string;
  logLevel?: "debug" | "info" | "warn" | "error";
  mode?: "development" | "production" | "testing";
  customSidecar?: ForstSidecarType;
}

export type ForstClientConfig = ForstInvokeClientConfig;

const defaultInvokeBaseUrl = "http://127.0.0.1:6321";

function resolveEnvBaseUrl(): string | undefined {
  return (
    process.env.FORST_INVOKE_URL ??
    process.env.FORST_BASE_URL ??
    process.env.FORST_DEV_URL
  );
}

function resolveExplicitBaseUrl(config?: ForstInvokeClientConfig): string | undefined {
  return config?.baseUrl ?? config?.devServerUrl ?? resolveEnvBaseUrl();
}

function createInvokeBaseUrlResolver(config?: ForstInvokeClientConfig) {
  const fallback =
    resolveExplicitBaseUrl(config) ?? defaultInvokeBaseUrl;
  return () =>
    readInvokeReadyUrl(config?.rootDir) ?? resolveExplicitBaseUrl(config) ?? fallback;
}

function resolveBaseUrl(config?: ForstInvokeClientConfig): string | undefined {
  return (
    resolveExplicitBaseUrl(config) ?? readInvokeReadyUrl(config?.rootDir)
  );
}

function shouldConnect(config?: ForstInvokeClientConfig): boolean {
  if (config?.transport === "http") {
    return Boolean(resolveBaseUrl(config));
  }
  if (config?.transport === "dev") {
    return false;
  }
  if (config?.sidecarRuntime === "spawn") {
    return false;
  }
  if (config?.sidecarRuntime === "connect") {
    return true;
  }
  if (process.env.FORST_SKIP_SPAWN === "1") {
    return true;
  }
  if (config?.baseUrl !== undefined || config?.devServerUrl !== undefined) {
    return true;
  }
  if (readInvokeReadyUrl(config?.rootDir)) {
    return true;
  }
  return Boolean(resolveEnvBaseUrl());
}

class HttpInvokeClient {
  private client: ForstSidecarClient;

  constructor(config?: ForstInvokeClientConfig) {
    const resolveBaseUrlFn = createInvokeBaseUrlResolver(config);
    this.client = new ForstSidecarClient({
      baseUrl: resolveBaseUrlFn(),
      resolveBaseUrl: resolveBaseUrlFn,
      timeout: config?.timeout ?? 30000,
      retries: config?.retries ?? 0,
      reloadAware: true,
    });
  }

  invokeFunction<T>(pkg: string, fn: string, args: unknown[]) {
    return this.client.invokeFunction<T>(pkg, fn, args);
  }

  invokeStream<T>(pkg: string, fn: string, args: unknown[]) {
    return invokeStreamRows(this.client.invokeStream<T>(pkg, fn, args));
  }

  healthCheck() {
    return this.client.healthCheck();
  }

  getVersion() {
    return this.client.getVersion();
  }
}

class SidecarInvokeClient {
  private sidecar: ForstSidecar;

  constructor(config?: ForstInvokeClientConfig) {
    this.sidecar = new ForstSidecar({
      mode: config?.mode ?? "development",
      port: config?.port ?? 6320,
      host: config?.host ?? "localhost",
      logLevel: config?.logLevel ?? "info",
      rootDir: config?.rootDir ?? process.cwd(),
      sidecarRuntime: config?.sidecarRuntime,
      devServerUrl: config?.devServerUrl,
    });
  }

  async invokeFunction<T>(pkg: string, fn: string, args: unknown[]) {
    await this.sidecar.start();
    return this.sidecar.invoke(pkg, fn, args) as Promise<InvokeSuccess<T>>;
  }

  async *invokeStream<T>(pkg: string, fn: string, args: unknown[]) {
    await this.sidecar.start();
    yield* invokeStreamRows(this.sidecar.invokeStream<T>(pkg, fn, args));
  }

  async healthCheck() {
    await this.sidecar.start();
    return this.sidecar.healthCheck();
  }

  async getVersion() {
    await this.sidecar.start();
    return this.sidecar.getVersion();
  }
}

export interface ForstInvokeClient {
  invokeFunction<T>(
    pkg: string,
    fn: string,
    args: unknown[]
  ): Promise<InvokeSuccess<T>>;
  invokeStream<T>(
    pkg: string,
    fn: string,
    args: unknown[]
  ): AsyncGenerator<T, void, unknown>;
  healthCheck(): Promise<boolean>;
  getVersion(): Promise<ServerVersionInfo>;
}

let defaultClient: ForstInvokeClient | undefined;

export function createInvokeClient(
  config?: ForstInvokeClientConfig
): ForstInvokeClient {
  if (shouldConnect(config)) {
    return new HttpInvokeClient(config);
  }
  return new SidecarInvokeClient(config);
}

export function getDefaultInvokeClient(): ForstInvokeClient {
  if (!defaultClient) {
    defaultClient = createInvokeClient();
  }
  return defaultClient;
}

export function resetDefaultInvokeClientForTest(): void {
  defaultClient = undefined;
}
