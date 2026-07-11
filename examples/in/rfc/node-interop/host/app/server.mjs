/**
 * Mock app server entry for host-mode Node interop.
 *
 * startForstNodeHost() runs via auto-injected register.mjs (deferAppReady).
 * Signal app readiness after seeding shared state.
 */
import { signalForstAppReady } from "@forst/node-runtime/host";

globalThis.__forstTest = { n: 1 };
await signalForstAppReady();
