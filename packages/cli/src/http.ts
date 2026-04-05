/** Retries transient failures (network errors, 429, 502, 503). */
export async function fetchWithRetry(
  fetchFn: typeof fetch,
  url: string | URL,
  init?: RequestInit,
  maxAttempts = 5
): Promise<Response> {
  for (let attempt = 1; attempt <= maxAttempts; attempt++) {
    try {
      const response = await fetchFn(url, init);
      if (
        response.status === 429 ||
        response.status === 502 ||
        response.status === 503
      ) {
        if (attempt < maxAttempts) {
          await sleep(backoffMs(attempt));
          continue;
        }
      }
      return response;
    } catch (e) {
      if (attempt < maxAttempts) {
        await sleep(backoffMs(attempt));
        continue;
      }
      throw e;
    }
  }

  throw new Error("fetchWithRetry: maxAttempts must be >= 1");
}

function backoffMs(attempt: number): number {
  return Math.min(1000 * 2 ** (attempt - 1), 10_000);
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}
