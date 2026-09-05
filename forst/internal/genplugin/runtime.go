package genplugin

// TSReadBodyHelper is a sealed helper for HTTP handlers (JSON or form body → object).
const TSReadBodyHelper = `export async function readBody(request: Request): Promise<Record<string, unknown>> {
  const ct = request.headers.get("content-type") ?? "";
  if (ct.includes("application/json")) {
    const v = await request.json().catch(() => ({}));
    return v && typeof v === "object" && !Array.isArray(v) ? (v as Record<string, unknown>) : {};
  }
  if (ct.includes("form")) {
    const fd = await request.formData();
    return Object.fromEntries(fd.entries());
  }
  return {};
}
`

// TSMatchRouteHelper matches `/api/orders/:id` patterns against a pathname.
const TSMatchRouteHelper = `export type MatchedRoute<T> = { value: T; params: Record<string, string> };

export function matchRoute<T>(
  pathname: string,
  routes: Record<string, T>,
): MatchedRoute<T> | undefined {
  const parts = pathname.replace(/\/+$/, "").split("/").filter(Boolean);
  for (const [pattern, value] of Object.entries(routes)) {
    const segs = pattern.replace(/\/+$/, "").split("/").filter(Boolean);
    if (segs.length !== parts.length) continue;
    const params: Record<string, string> = {};
    let ok = true;
    for (let i = 0; i < segs.length; i++) {
      const seg = segs[i];
      if (seg.startsWith(":")) {
        params[seg.slice(1)] = decodeURIComponent(parts[i] ?? "");
        continue;
      }
      if (seg !== parts[i]) {
        ok = false;
        break;
      }
    }
    if (ok) return { value, params };
  }
  return undefined;
}
`
