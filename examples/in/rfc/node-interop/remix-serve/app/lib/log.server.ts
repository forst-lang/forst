/** Server-side logs for remix-serve (host child stdout/stderr → forst run). */
export function logServer(scope: string, message: string, detail?: Record<string, unknown>) {
  const suffix = detail ? " " + JSON.stringify(detail) : "";
  console.error(`[remix-serve/${scope}] ${message}${suffix}`);
}

export function logRequest(method: string, url: string) {
  logServer("request", `${method} ${url}`);
}
