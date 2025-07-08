interface HTTPRequest {
  action: string;
  testFile?: string;
  args?: any;
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

  async runTest(testFile: string): Promise<HTTPResponse> {
    const request: HTTPRequest = {
      action: "run",
      testFile: testFile,
    };

    const response = await fetch(`${this.baseUrl}/run`, {
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

  async compileFile(testFile: string): Promise<HTTPResponse> {
    const request: HTTPRequest = {
      action: "compile",
      testFile: testFile,
    };

    const response = await fetch(`${this.baseUrl}/compile`, {
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

export type { HTTPRequest, HTTPResponse };
