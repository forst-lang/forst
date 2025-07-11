import pino from "pino";

const level = process.env.FORST_LOG_LEVEL || "info";

export const logger = pino({
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
