package transformerts

// JSDoc blocks for compiler-owned symbols that consumers import from a generated client.
// User-defined package exports keep their existing comments.
//
// Use {@link Symbol} for generated and global types. Use {@link URL | label} for Effect
// APIs and other external docs so IDE hover shows clickable references.

const jsdocLinkEffectRetry = `{@link https://effect.website/docs/error-management/retrying/ | Effect.retry}`

const jsdocLinkEffectLayer = `{@link https://effect.website/docs/requirements-management/layers/ | Effect Layer}`

const jsdocLinkLayerProvide = `{@link https://effect.website/docs/requirements-management/layers/#provide | Layer.provide}`

const jsdocLinkManagedRuntime = `{@link https://effect.website/docs/runtime/managed-runtime/ | ManagedRuntime}`

const jsdocLinkSymbolAsyncDispose = `{@link https://developer.mozilla.org/docs/Web/JavaScript/Reference/Global_Objects/Symbol/asyncDispose | Symbol.asyncDispose}`

const jsdocStreamingResult = `/**
 * One parsed line from a streaming invoke HTTP response.
 *
 * The transport reads the body as newline-delimited JSON.
 * When status is "done", the stream has finished.
 */`

const jsdocInvokeSuccess = `/**
 * Successful non-streaming invoke envelope after HTTP 2xx.
 *
 * Read result for the decoded return value from the Forst function.
 */`

const jsdocInvokeCallOptions = `/**
 * Per-call options for {@link ForstInvokeClient.invokeFunction} and {@link ForstInvokeClient.invokeStream}.
 *
 * Merges over defaults from {@link createInvokeClient} or {@link configureDefaultInvokeClient}.
 */`

const jsdocInvokeContext = `/**
 * Metadata passed to {@link ForstInvokeMiddleware} hooks for one invoke attempt.
 *
 * attempt starts at 1 and increases when the transport retries.
 */`

const jsdocForstInvokeMiddleware = `/**
 * Observability hooks around each invoke attempt.
 *
 * Register hooks on {@link createForstClient} or {@link configureDefaultInvokeClient} through the middleware array.
 */`

const jsdocForstInvokeClientConfig = `/**
 * Configuration for the HTTP invoke transport.
 * Set baseUrl or rely on FORST_BASE_URL, FORST_INVOKE_URL, and FORST_DEV_URL discovery.
 *
 * When NODE_ENV is production, the transport connects only. Never spawns forst dev.
 */`

const jsdocForstInvokeClient = `/**
 * Low-level invoke surface used by generated package modules.
 *
 * Call {@link ForstInvokeClient.invokeFunction} for request-response RPC and {@link ForstInvokeClient.invokeStream} for NDJSON streaming.
 */`

const jsdocCreateInvokeClient = `/**
 * Creates a dedicated HTTP invoke client from {@link ForstInvokeClientConfig}.
 *
 * Generated flat exports use {@link getDefaultInvokeClient} instead of holding their own client.
 */`

const jsdocGetDefaultInvokeClient = `/**
 * Returns the process-wide invoke client used by flat package subpath exports.
 *
 * {@link configureDefaultInvokeClient} replaces the cached instance on the next call.
 */`

const jsdocConfigureDefaultInvokeClient = `/**
 * Merges config into the default invoke client and clears the cached instance.
 *
 * Flat exports and {@link createForstClient} pick up the new base URL, headers, and middleware on the next invoke.
 */`

const jsdocResetDefaultInvokeClientForTest = `/**
 * Clears the cached default invoke client so the next {@link getDefaultInvokeClient} call rebuilds from current config.
 *
 * Tests call this between cases and after {@link startForstTestServer} reconfigures the transport.
 */`

const jsdocCreateForstClient = `/**
 * Builds a namespaced client object with one property per generated Forst package.
 *
 * Each property exposes the same functions as the matching flat export module.
 */`

const jsdocForstClientConfig = `/**
 * Root-level client configuration accepted by {@link createForstClient}.
 *
 * Accepts the same fields as {@link ForstInvokeClientConfig} except transport spawn flags stay on the transport config when you pass {@link ForstInvokeClientConfig} directly.
 */`

const jsdocForstClientType = `/**
 * Return type of {@link createForstClient}.
 *
 * Keys match generated package names and values are the bound namespace objects.
 */`

const jsdocWithTransport = `/**
 * Merges an invoke client and {@link AbortSignal} into {@link EffectInvokeCallOptions} for generated Effect functions.
 *
 * Effect mode passes retries through ` + jsdocLinkEffectRetry + `, so this helper strips retries from the options object.
 */`

const jsdocForstTransport = `/**
 * Effect service tag that holds the shared HTTP invoke client for generated package modules.
 *
 * Provide {@link ForstTransportLayer} before calling generated Effect functions in a custom runtime.
 */`

const jsdocForstTransportLayer = `/**
 * Builds an ` + jsdocLinkEffectLayer + ` that supplies {@link ForstTransport} with {@link createInvokeClient}(config).
 *
 * One layer instance shares the same client across every generated package service in the program.
 */`

const jsdocForstClientLive = `/**
 * Live Effect layer that merges the Default layer of every generated package service.
 *
 * Use this when the program already provides {@link ForstTransport} through another ` + jsdocLinkLayerProvide + ` step.
 */`

const jsdocForstClientLayer = `/**
 * Builds the full client stack for Effect programs in this project.
 *
 * Merges every package DefaultWithoutDependencies layer and provides a shared {@link ForstTransportLayer} built from config.
 */`

const jsdocMakeForstClientRuntime = `/**
 * Creates a ` + jsdocLinkManagedRuntime + ` from {@link ForstClientLayer}(config).
 *
 * Run generated Effect functions through runtime.runPromise or runtime.runSync after supplying layers.
 */`

const jsdocForstTestLayer = `/**
 * Builds an Effect Layer that replaces generated package handlers with mocks from {@link ForstTestOverrides.packages}.
 *
 * Compose this layer above {@link ForstClientLayer} in tests that need partial handler overrides.
 */`

const jsdocForstTestServerTag = `/**
 * Effect context tag for a running Forst invoke test server started by {@link ForstTestServerLayer}.
 *
 * The service value exposes baseUrl, port, connection mode, and a stop function.
 */`

const jsdocForstTestServerLayer = `/**
 * Starts a real Forst invoke server for the scope of an Effect program and wires {@link ForstTransport} to it.
 *
 * Requires ` + "`@forst/cli`" + ` as a dev dependency. Failures surface as {@link ForstTestServerFailed} on the error channel.
 */`

const jsdocMakeForstTestServer = `/**
 * Creates a ` + jsdocLinkManagedRuntime + ` backed by {@link ForstTestServerLayer}(options).
 *
 * Use it in Effect tests that need a live server without assembling layers manually.
 */`

const jsdocWithForstTestScope = `/**
 * Installs package, function, and transport overrides for the duration of run.
 *
 * Nested calls merge overrides and the innermost handler wins for each package and function name.
 */`

const jsdocCreateTestForstClient = `/**
 * Builds an in-memory client object backed by handler maps instead of HTTP.
 *
 * Pass the same shape as {@link ForstTestOverrides.packages} to stub individual functions without AsyncLocalStorage scope.
 */`

const jsdocStartForstTestServer = `/**
 * Starts a real Forst invoke server and configures the default transport to reach it.
 *
 * Returns a handle with baseUrl, port, stop, and ` + jsdocLinkSymbolAsyncDispose + `.
 * Requires ` + "`@forst/cli`" + ` as a dev dependency.
 */`

const jsdocForstTestOverrides = `/**
 * Override map for {@link withForstTestScope} and {@link createTestForstClient}.
 *
 * Replace whole packages, single functions, or the raw invoke transport to assert on wire behavior.
 */`

const jsdocForstTestServerOptions = `/**
 * Options forwarded to ` + "`@forst/cli/invoke`" + ` startForstInvokeServer when a test starts a real server.
 *
 * Set root to the project directory that contains ftconfig.json when it differs from process.cwd().
 */`

const jsdocForstTestServerHandle = `/**
 * Handle for a running invoke test server in Promise-mode tests.
 *
 * Call stop or await ` + jsdocLinkSymbolAsyncDispose + ` to tear down the child process and reset the default transport.
 */`

const jsdocEffectInvokeCallOptions = `/**
 * Per-call options for generated Effect package functions.
 *
 * Omits retries from {@link InvokeCallOptions} because Effect mode expects callers to wrap the program with ` + jsdocLinkEffectRetry + `.
 */`
