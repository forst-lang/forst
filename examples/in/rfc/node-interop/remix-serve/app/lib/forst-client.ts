import { createRequire } from "node:module";

const require = createRequire(import.meta.url);
const client = require("@forst/client") as typeof import("@forst/client");

export const createInvokeClient = client.createInvokeClient;
export const getDefaultInvokeClient = client.getDefaultInvokeClient;
export const resetDefaultInvokeClientForTest =
  client.resetDefaultInvokeClientForTest;
export type { ForstInvokeClient, ForstInvokeClientConfig } from "@forst/client";
