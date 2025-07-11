import pino from "pino";

const baseLogger = pino({
  level: process.env.LOG_LEVEL || "info",
  transport: {
    target: "pino-pretty",
    options: {
      colorize: true,
      translateTime: "SYS:standard",
      ignore: "pid,hostname",
    },
  },
});

export const logger = baseLogger;

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

export const testLogger = createLogger("test");
export const runnerLogger = createLogger("runner");
export const sidecarLogger = createLogger("sidecar");
