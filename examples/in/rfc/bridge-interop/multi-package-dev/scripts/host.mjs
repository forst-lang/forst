// Minimal persistent host shim for multi-package-dev hostMode reload e2e.
// Auto-register (register.mjs) writes node.sock.ready; this process only stays alive.
await new Promise(() => {});
