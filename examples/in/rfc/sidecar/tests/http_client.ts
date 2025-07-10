interface InvokeRequest {
  package: string;
  function: string;
  args?: any;
  streaming?: boolean;
}

interface HTTPResponse {
  success: boolean;
  output?: string;
  error?: string;
  result?: any;
}

export class ForstHTTPClient {
  private baseUrl: string;

  constructor(baseUrl: string) {
    this.baseUrl = baseUrl;
  }

  async healthCheck(): Promise<boolean> {
    try {
      const response = await fetch(`${this.baseUrl}/health`);
      const data: HTTPResponse = await response.json();
      return data.success;
    } catch (error) {
      console.error("Health check failed:", error);
      return false;
    }
  }

  async invoke(fn: string, args?: any): Promise<HTTPResponse> {
    // Parse function name to extract package and function
    const parts = fn.split(".");
    if (parts.length !== 2) {
      throw new Error(
        `Invalid function name format: ${fn}. Expected format: package.function`
      );
    }

    const [packageName, functionName] = parts;

    const request: InvokeRequest = {
      package: packageName,
      function: functionName,
      args: args,
    };

    const response = await fetch(`${this.baseUrl}/invoke`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(request),
    });

    if (!response.ok) {
      throw new Error(
        `HTTP request failed: ${response.status} ${response.statusText}`
      );
    }

    return await response.json();
  }
}

export type { InvokeRequest, HTTPResponse };
