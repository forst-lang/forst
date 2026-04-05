# `forst dev` HTTP contract (Node / sidecar)

Normative reference for the **HTTP** surface exposed by `forst dev` and consumed by [`@forst/sidecar`](../../../../packages/sidecar/README.md). Implementation: [`forst/cmd/forst/dev_server.go`](../../../../forst/cmd/forst/dev_server.go).

**Contract version:** tied to compiler releases; breaking changes to JSON field names or routes should be noted in release notes until a separate version field is added.

## Base URL

Server listens on `http://<host>:<port>/` where `port` is the `-port` flag to `forst dev` (default from config).

## JSON envelope

Responses use `Content-Type: application/json` unless noted.

```go
// Go names — JSON uses lowercase tags
type DevServerResponse struct {
    Success bool            `json:"success"`
    Output  string          `json:"output,omitempty"`
    Error   string          `json:"error,omitempty"`
    Result  json.RawMessage `json:"result,omitempty"`
}
```

Errors from `sendError` set HTTP status (4xx/5xx) and `success: false`, `error: "<message>"`.

## Endpoints

### `GET /health`

- **Response:** `{ "success": true, "output": "Forst HTTP server is healthy" }`
- **Errors:** `405` if not GET.

### `GET /functions`

- Refreshes discovery, then returns all discovered functions.
- **Response:** `success: true`, `result`: JSON array of `discovery.FunctionInfo` (package, name, parameters, return type, file path, streaming flag, etc.).
- **Errors:** `500` if discovery fails; `405` if not GET.

### `POST /invoke`

**Request body:**

```json
{
  "package": "<forst package name>",
  "function": "<function name>",
  "args": <json> ,
  "streaming": false
}
```

`args` is opaque JSON passed to the executor (often a JSON array of call arguments). The sidecar sends an **array** for positional args (see [`packages/sidecar`](../../../../packages/sidecar)).

**Non-streaming response:** `success`, `output`, `error`, `result` mirror executor output (`Result` is JSON-encoded payload).

**Streaming:** `streaming: true` requires `SupportsStreaming` on the function; response uses chunked JSON lines (`application/octet-stream`, chunked transfer).

**Errors:** `404` unknown package/function; `400` bad JSON or streaming not supported; `500` execution failure.

### `GET /types`

- Returns merged TypeScript for discovered functions (regenerates when forced, cache empty, or cache older than ~5 minutes).
- Query: `?force=true` to bypass cache.
- **Response:** JSON with `Content-Type: application/json`, `success: true`, and **`output`**: string containing the merged TypeScript source. Clients parse JSON and read **`output`** (do not assume raw `.ts` bytes without parsing the envelope).

**Errors:** `500` if generation fails; `405` if not GET.

## CORS

When server config enables CORS, responses include `Access-Control-Allow-Origin: *` and standard method/header allowances for browser clients.

## Reference client

[`@forst/sidecar`](../../../../packages/sidecar) `ForstSidecarClient` is the **reference** HTTP client; keep it aligned with this document.

## Related

- Sidecar RFC: [../sidecar/00-sidecar.md](../sidecar/00-sidecar.md)
- Implementation plan (codegen): [00-implementation-plan.md](./00-implementation-plan.md)
