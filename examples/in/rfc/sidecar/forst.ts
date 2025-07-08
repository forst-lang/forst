// TypeScript client for Forst sidecar API
// This client provides a clean interface for calling Forst functions from TypeScript

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

export class ForstClient {
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
    const response = await this.makeRequest("/functions", {
      method: "GET",
    });

    if (!response.success) {
      throw new Error(`Failed to discover functions: ${response.error}`);
    }

    const functions = response.result as FunctionInfo[];

    // Cache the functions
    for (const fn of functions) {
      const key = `${fn.package}.${fn.name}`;
      this.functions.set(key, fn);
    }

    return functions;
  }

  /**
   * Invoke a Forst function
   */
  async invoke(
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

    const response = await this.makeRequest("/invoke", {
      method: "POST",
      body: JSON.stringify(request),
    });

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
            console.warn("Failed to parse streaming chunk:", error);
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
      const response = await this.makeRequest("/health", {
        method: "GET",
      });
      return response.success;
    } catch (error) {
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

    for (let attempt = 0; attempt < this.config.retries!; attempt++) {
      try {
        const controller = new AbortController();
        const timeoutId = setTimeout(
          () => controller.abort(),
          this.config.timeout
        );

        const response = await fetch(url, {
          ...options,
          signal: controller.signal,
        });

        clearTimeout(timeoutId);

        if (!response.ok) {
          const error = await response.text();
          throw new Error(`HTTP ${response.status}: ${error}`);
        }

        const result = await response.json();
        return result as InvokeResponse;
      } catch (error) {
        if (attempt === this.config.retries! - 1) {
          throw error;
        }

        // Wait before retry (exponential backoff)
        await this.delay(Math.pow(2, attempt) * 1000);
      }
    }

    throw new Error("Request failed after all retries");
  }

  private delay(ms: number): Promise<void> {
    return new Promise((resolve) => setTimeout(resolve, ms));
  }
}

// Example usage:
export async function exampleUsage() {
  const client = new ForstClient({
    baseUrl: "http://localhost:8080",
  });

  // Discover available functions
  const functions = await client.discoverFunctions();
  console.log("Available functions:", functions);

  // Call a regular function
  const result = await client.invoke("example", "ProcessData", {
    input: "test data",
  });
  console.log("Result:", result);

  // Call a streaming function
  await client.invokeStreaming(
    "example",
    "ProcessStream",
    {
      input: "streaming data",
    },
    (result) => {
      console.log("Streaming result:", result);
    }
  );
}

// Express.js integration example
export function createExpressMiddleware(client: ForstClient) {
  return async (req: any, res: any, next: any) => {
    if (req.path.startsWith("/forst/")) {
      const [packageName, functionName] = req.path.slice(7).split("/");

      try {
        const result = await client.invoke(packageName, functionName, req.body);
        res.json(result);
      } catch (error) {
        res.status(500).json({
          success: false,
          error: error instanceof Error ? error.message : String(error),
        });
      }
    } else {
      next();
    }
  };
}
