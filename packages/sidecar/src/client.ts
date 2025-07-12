import {
  ForstClientConfig,
  InvokeRequest,
  InvokeResponse,
  StreamingResult,
  FunctionInfo,
} from "./types";
import { logger } from "./logger";

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
        throw new Error(`Failed to discover functions: ${response.error}`);
      }

      const functions = response.result as FunctionInfo[];
      logger.debug(`🔍 Discovered ${functions.length} functions:`, functions);

      // Cache the functions
      for (const fn of functions) {
        const key = `${fn.package}.${fn.name}`;
        this.functions.set(key, fn);
      }

      logger.info(`✅ Discovered ${functions.length} functions`);
      return functions;
    } catch (error) {
      logger.error("❌ Failed to discover functions:", error);
      return [];
    }
  }

  /**
   * Invoke a Forst function using package.function format
   */
  async invoke(fn: string, args?: any): Promise<InvokeResponse> {
    // Parse function name to extract package and function
    const parts = fn.split(".");
    if (parts.length !== 2) {
      throw new Error(
        `Invalid function name format: ${fn}. Expected format: package.function`
      );
    }

    const [packageName, functionName] = parts;
    return this.invokeFunction(packageName, functionName, args);
  }

  /**
   * Invoke a Forst function with explicit package and function names
   */
  async invokeFunction(
    packageName: string,
    functionName: string,
    args: any = {},
    options: { streaming?: boolean } = {}
  ): Promise<InvokeResponse> {
    const request: InvokeRequest = {
      package: packageName,
      function: functionName,
      args,
      streaming: options.streaming || false,
    };

    logger.debug(`🚀 Invoking ${packageName}.${functionName} with args:`, args);
    logger.debug(`📋 Full request:`, request);

    const response = await this.makeRequest("/invoke", {
      method: "POST",
      body: JSON.stringify(request),
    });

    logger.debug(`📦 Response for ${packageName}.${functionName}:`, response);
    return response;
  }

  /**
   * Invoke a Forst function with streaming support
   */
  async invokeStreaming(
    packageName: string,
    functionName: string,
    args: any = {},
    onResult?: (result: StreamingResult) => void
  ): Promise<void> {
    const request: InvokeRequest = {
      package: packageName,
      function: functionName,
      args,
      streaming: true,
    };

    logger.debug(
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
      const error = await response.text();
      throw new Error(`Streaming request failed: ${error}`);
    }

    if (!response.body) {
      throw new Error("No response body available for streaming");
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
            logger.warn("Failed to parse streaming chunk:", error);
          }
        }
      }
    } finally {
      reader.releaseLock();
    }
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
      logger.debug(`🏥 Health check response:`, response);
      return response.success;
    } catch (error) {
      logger.error("🏥 Health check failed:", error);
      return false;
    }
  }

  /**
   * Make an HTTP request with retry logic
   */
  private async makeRequest(
    endpoint: string,
    options: RequestInit
  ): Promise<InvokeResponse> {
    const url = `${this.config.baseUrl}${endpoint}`;
    let lastError: Error | null = null;

    logger.debug(`🌐 Making request to: ${url}`);
    logger.debug(`📤 Request method: ${options.method}`);
    logger.debug(`📤 Request headers:`, options.headers);
    if (options.body) {
      logger.debug(`📤 Request body: ${options.body}`);
    }

    for (let attempt = 0; attempt <= this.config.retries!; attempt++) {
      try {
        logger.debug(
          `🔄 Request attempt ${attempt + 1}/${this.config.retries! + 1}`
        );

        const response = await fetch(url, {
          ...options,
          headers: {
            "Content-Type": "application/json",
            ...options.headers,
          },
          signal: AbortSignal.timeout(this.config.timeout!),
        });

        logger.debug(
          `📥 Response status: ${response.status} ${response.statusText}`
        );
        logger.debug(
          `📥 Response headers:`,
          Object.fromEntries(response.headers.entries())
        );

        if (!response.ok) {
          const errorText = await response.text();
          logger.error(`❌ HTTP ${response.status}: ${errorText}`);
          throw new Error(`HTTP ${response.status}: ${errorText}`);
        }

        const result = await response.json();
        logger.debug(`✅ Request successful:`, result);
        return result as InvokeResponse;
      } catch (error) {
        lastError = error as Error;
        logger.warn(`❌ Request attempt ${attempt + 1} failed:`, error);

        if (attempt < this.config.retries!) {
          const delay = Math.pow(2, attempt) * 1000; // Exponential backoff
          logger.debug(`⏳ Retrying in ${delay}ms...`);
          await this.delay(delay);
        }
      }
    }

    logger.error(`💥 All request attempts failed. Last error:`, lastError);
    throw lastError || new Error("Request failed after all retries");
  }

  /**
   * Delay utility for retry logic
   */
  private delay(ms: number): Promise<void> {
    return new Promise((resolve) => setTimeout(resolve, ms));
  }
}
