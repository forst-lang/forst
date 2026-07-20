import pino from "pino";

const level = process.env.FORST_LOG_LEVEL || "info";

const baseLogger = pino({
  level,
  transport: {
    target: "pino-pretty",
    options: {
      colorize: true,
      translateTime: "SYS:standard",
      ignore: "pid,hostname",
    },
  },
});

/** Root pino logger for the sidecar package; level from `FORST_LOG_LEVEL`. */
export const logger = baseLogger;

/** Builds a scoped logger so spawn, transport, and RPC layers tag log lines consistently. */
export function createLogger(scope: string) {
  return {
    info: (msg: string, ...args: any[]) =>
      baseLogger.info({ scope }, msg, ...args),
    error: (msg: string, ...args: any[]) =>
      baseLogger.error({ scope }, msg, ...args),
    warn: (msg: string, ...args: any[]) =>
      baseLogger.warn({ scope }, msg, ...args),
    debug: (msg: string, ...args: any[]) =>
      baseLogger.debug({ scope }, msg, ...args),
    trace: (msg: string, ...args: any[]) =>
      baseLogger.trace({ scope }, msg, ...args),
  };
}

/** Logs sidecar HTTP server lifecycle (listen, shutdown, health waits). */
export const serverLogger = createLogger("server");

/** Forwards stdout/stderr from the supervised `forst dev` child process. */
export const forstLogger = createLogger("forst");

/** Shared helpers used across sidecar modules (paths, config, misc). */
export const utilsLogger = createLogger("utils");

/** Sidecar-side messages when acting as the invoke client toward a dev server. */
export const clientLogger = createLogger("client");
