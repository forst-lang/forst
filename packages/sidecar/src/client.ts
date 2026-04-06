import {
  ForstClientConfig,
  InvokeRequest,
  InvokeResponse,
  InvokeSuccess,
  StreamingResult,
  FunctionInfo,
  ServerVersionInfo,
} from "./types";
import { logger } from "./logger";
import {
  invokeRequestLogFields,
  invokeResponseLogFields,
  sanitizePayload,
  sanitizeRequestBodyString,
} from "./sanitizeLogPayload";
import {
  SIDECAR_PACKAGE_VERSION,
  SIDECAR_VERSION_HTTP_HEADER,
} from "./constants";
import {
  DevServerFunctionsRejected,
  DevServerHttpFailure,
  DevServerInvokeRejected,
  DevServerRequestRetriesExhausted,
  DevServerTypesOutputMissing,
  DevServerTypesRejected,
  DevServerVersionRejected,
  InvalidFunctionNameFormat,
  DevServerStreamingInvokeNoResponseBody,
} from "./errors";

/**
 * HTTP client for the Forst dev server (`/invoke`, `/functions`, `/types`, `/health`).
 */
export class ForstSidecarClient {
  private config: ForstClientConfig;
  private functions: Map<string, FunctionInfo> = new Map();

  constructor(config: ForstClientConfig) {
    this.config = {
      timeout: 30000,
      retries: 3,
      ...config,
    };
  }

  /**
   * Discover available functions from the Forst server
   */
  async discoverFunctions(): Promise<FunctionInfo[]> {
    try {
      logger.debug(
        `🔍 Discovering functions from ${this.config.baseUrl}/functions`
      );
      const response = await this.makeRequest("/functions", {
        method: "GET",
      });

      if (!response.success) {
        logger.error(`❌ Failed to discover functions: ${response.error}`);
        throw new DevServerFunctionsRejected(response.error);
      }

      const functions = response.result as FunctionInfo[];
      logger.debug(
        {
          count: functions.length,
          functions: sanitizePayload(functions),
        },
        `🔍 Discovered ${functions.length} functions`
      );

      // Cache the functions
      for (const fn of functions) {
        const key = `${fn.package}.${fn.name}`;
        this.functions.set(key, fn);
      }

      logger.info(`✅ Discovered ${functions.length} functions`);
      return functions;
    } catch (error) {
      logger.error({ err: error }, "❌ Failed to discover functions");
      return [];
    }
  }

  /**
   * Invoke a Forst function using package.function format
   */
  async invoke<T>(fn: string, args?: any[]): Promise<InvokeSuccess<T>> {
    // Parse function name to extract package and function
    const parts = fn.split(".");
    if (parts.length !== 2) {
      throw new InvalidFunctionNameFormat(fn);
    }

    const [packageName, functionName] = parts;
    return this.invokeFunction<T>(packageName, functionName, args);
  }

  /**
   * Invoke a Forst function with explicit package and function names
   */
  async invokeFunction<T>(
    packageName: string,
    functionName: string,
    args: any[] = [],
    options: { streaming?: boolean } = {}
  ): Promise<InvokeSuccess<T>> {
    const request: InvokeRequest = {
      package: packageName,
      function: functionName,
      args,
      streaming: options.streaming || false,
    };

    logger.debug(
      invokeRequestLogFields(request),
      `🚀 Invoking ${packageName}.${functionName}`
    );

    const response = await this.makeRequest<T>("/invoke", {
      method: "POST",
      body: JSON.stringify(request),
    });

    logger.debug(
      {
        package: packageName,
        function: functionName,
        ...invokeResponseLogFields(response),
      },
      `📦 Response for ${packageName}.${functionName}`
    );
    if (!response.success) {
      throw new DevServerInvokeRejected(packageName, functionName, response);
    }
    if (response.result === undefined) {
      throw new DevServerInvokeRejected(packageName, functionName, {
        success: false,
        error:
          response.error ??
          "invoke response missing result (success was true but result is undefined)",
        output: response.output,
      });
    }
    return {
      success: true as const,
      result: response.result,
      output: response.output,
      error: response.error,
    };
  }

  /**
   * Invoke a Forst function with streaming support
   */
  async invokeStreaming(
    packageName: string,
    functionName: string,
    args: unknown[] = [],
    onResult?: (result: StreamingResult) => void
  ): Promise<void> {
    const request: InvokeRequest = {
      package: packageName,
      function: functionName,
      args,
      streaming: true,
    };

    logger.debug(
      invokeRequestLogFields(request),
      `Starting streaming invocation of ${packageName}.${functionName}`
    );

    const response = await fetch(`${this.config.baseUrl}/invoke`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(request),
    });

    if (!response.ok) {
      const errorText = await response.text();
      throw new DevServerHttpFailure(
        response.status,
        errorText,
        parseDevServerHttpErrorField(errorText)
      );
    }

    if (!response.body) {
      throw new DevServerStreamingInvokeNoResponseBody();
    }

    const reader = response.body.getReader();
    const decoder = new TextDecoder();

    try {
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        const chunk = decoder.decode(value, { stream: true });
        const lines = chunk.split("\n").filter((line) => line.trim());

        for (const line of lines) {
          try {
            const result: StreamingResult = JSON.parse(line);
            if (onResult) {
              onResult(result);
            }
          } catch (error) {
            logger.warn({ err: error }, "Failed to parse streaming chunk");
          }
        }
      }
    } finally {
      reader.releaseLock();
    }
  }

  /**
   * Clears the in-memory cache from {@link discoverFunctions}. Call after the dev server restarts
   * or reloads so the next discovery reflects new/removed functions.
   */
  invalidateFunctionCache(): void {
    this.functions.clear();
  }

  /**
   * Get information about a specific function
   */
  getFunctionInfo(
    packageName: string,
    functionName: string
  ): FunctionInfo | undefined {
    const key = `${packageName}.${functionName}`;
    return this.functions.get(key);
  }

  /**
   * Check if a function supports streaming
   */
  supportsStreaming(packageName: string, functionName: string): boolean {
    const fn = this.getFunctionInfo(packageName, functionName);
    return fn?.supportsStreaming || false;
  }

  /**
   * Fetch merged TypeScript definitions from `GET /types` (JSON response envelope per the HTTP contract).
   * @param force - when true, sends `?force=true` to bypass server cache.
   */
  async fetchTypes(options?: { force?: boolean }): Promise<string> {
    const q = options?.force ? "?force=true" : "";
    const response = await this.makeRequest(`/types${q}`, {
      method: "GET",
    });
    if (!response.success) {
      throw new DevServerTypesRejected(response.error);
    }
    if (response.output === undefined || response.output === "") {
      throw new DevServerTypesOutputMissing();
    }
    return response.output;
  }

  /**
   * Fetch compiler and HTTP contract metadata from `GET /version`.
   */
  async getVersion(): Promise<ServerVersionInfo> {
    const response = await this.makeRequest<ServerVersionInfo>("/version", {
      method: "GET",
    });
    if (!response.success) {
      throw new DevServerVersionRejected(response.error);
    }
    const raw = response.result as unknown;
    if (!raw || typeof raw !== "object") {
      throw new DevServerVersionRejected("missing result payload");
    }
    const r = raw as Record<string, unknown>;
    const version = String(r.version ?? "");
    const commit = String(r.commit ?? "");
    const date = String(r.date ?? "");
    const contractVersion = String(r.contractVersion ?? "");
    if (!version || !contractVersion) {
      throw new DevServerVersionRejected(
        "version or contractVersion missing in /version result"
      );
    }
    return {
      version,
      commit,
      date,
      contractVersion,
    };
  }

  /**
   * Health check
   */
  async healthCheck(): Promise<boolean> {
    try {
      logger.debug(
        `🏥 Performing health check to ${this.config.baseUrl}/health`
      );
      const response = await this.makeRequest("/health", {
        method: "GET",
      });
      logger.debug(
        invokeResponseLogFields(response),
        "🏥 Health check response"
      );
      return response.success;
    } catch (error) {
      logger.error({ err: error }, "🏥 Health check failed");
      return false;
    }
  }

  /**
   * Make an HTTP request with retry logic
   */
  private async makeRequest<T>(
    endpoint: string,
    options: RequestInit
  ): Promise<InvokeResponse<T>> {
    const url = `${this.config.baseUrl}${endpoint}`;
    let lastError: Error | null = null;

    logger.debug(`🌐 Making request to: ${url}`);
    logger.debug(`📤 Request method: ${options.method}`);
    logger.debug(
      { headers: sanitizePayload(options.headers) },
      "📤 Request headers"
    );
    if (options.body) {
      const bodyPreview =
        typeof options.body === "string"
          ? sanitizeRequestBodyString(options.body)
          : options.body instanceof ArrayBuffer
            ? { kind: "ArrayBuffer", byteLength: options.body.byteLength }
            : { kind: typeof options.body, note: "non-string body omitted" };
      logger.debug({ body: bodyPreview }, "📤 Request body (sanitized)");
    }

    for (let attempt = 0; attempt <= this.config.retries!; attempt++) {
      try {
        logger.debug(
          `🔄 Request attempt ${attempt + 1}/${this.config.retries! + 1}`
        );

        const response = await fetch(url, {
          ...options,
          headers: {
            [SIDECAR_VERSION_HTTP_HEADER]: SIDECAR_PACKAGE_VERSION,
            "Content-Type": "application/json",
            ...options.headers,
          },
          signal: AbortSignal.timeout(this.config.timeout!),
        });

        logger.debug(
          `📥 Response status: ${response.status} ${response.statusText}`
        );
        logger.debug(
          {
            headers: sanitizePayload(
              Object.fromEntries(response.headers.entries())
            ),
          },
          "📥 Response headers"
        );

        if (!response.ok) {
          const errorText = await response.text();
          logger.error(`❌ HTTP ${response.status}: ${errorText}`);
          throw new DevServerHttpFailure(
            response.status,
            errorText,
            parseDevServerHttpErrorField(errorText)
          );
        }

        const result = (await response.json()) as InvokeResponse<T>;
        logger.debug(
          invokeResponseLogFields(result),
          "✅ Request successful"
        );
        return result;
      } catch (error) {
        lastError = error as Error;
        logger.warn(
          { err: error },
          `❌ Request attempt ${attempt + 1} failed`
        );

        if (attempt < this.config.retries!) {
          const delay = Math.pow(2, attempt) * 1000; // Exponential backoff
          logger.debug(`⏳ Retrying in ${delay}ms...`);
          await this.delay(delay);
          continue;
        }
        if (this.config.retries! > 0) {
          throw new DevServerRequestRetriesExhausted(lastError);
        }
        throw lastError;
      }
    }
    throw new DevServerRequestRetriesExhausted(lastError);
  }

  /**
   * Delay utility for retry logic
   */
  private delay(ms: number): Promise<void> {
    return new Promise((resolve) => setTimeout(resolve, ms));
  }
}

/** When `forst dev` returns JSON `{ success: false, error: "..." }` with a non-2xx status. */
function parseDevServerHttpErrorField(errorText: string): string | undefined {
  try {
    const parsed = JSON.parse(errorText) as { error?: string };
    if (typeof parsed.error === "string") {
      return parsed.error;
    }
  } catch {
    /* plain-text body */
  }
  return undefined;
}
