package transformerts

import (
	"fmt"
	"strings"
)

// CliPeerDependencyRange is the optional peerDependency floor for startForstTestServer.
// Matches @forst/cli 0.2.0 which introduces the ./invoke subpath (Phase 7).
const CliPeerDependencyRange = ">=0.2.0"

// CliInvokeModuleSpecifier is the dynamic-import target for real-server lifecycle.
const CliInvokeModuleSpecifier = "@forst/cli/invoke"

// CliInstallCommand is embedded in ForstTestServerFailed when the peer is missing.
const CliInstallCommand = "npm i -D @forst/cli"

// emitForstTestServerOptionsDTS writes ForstTestServerOptions (structural copy of
// @forst/cli/invoke StartForstInvokeServerOptions so tsc works without the peer).
func emitForstTestServerOptionsDTS(b *strings.Builder) {
	b.WriteString(jsdocForstTestServerOptions)
	b.WriteString("\nexport interface ForstTestServerOptions {\n")
	b.WriteString("  root?: string;\n")
	b.WriteString("  mode?: \"auto\" | \"dev\" | \"embedded\";\n")
	b.WriteString("  entry?: string;\n")
	b.WriteString("  port?: number;\n")
	b.WriteString("  baseUrl?: string;\n")
	b.WriteString("  env?: Record<string, string>;\n")
	b.WriteString("  timeoutMs?: number;\n")
	b.WriteString("  logLevel?: \"error\" | \"warn\" | \"info\" | \"debug\" | \"trace\";\n")
	b.WriteString("  onLog?: (line: string, stream: \"stdout\" | \"stderr\") => void;\n")
	b.WriteString("}\n\n")
}

// emitForstTestServerHandleDTS writes the Promise-mode handle interface.
func emitForstTestServerHandleDTS(b *strings.Builder) {
	b.WriteString(jsdocForstTestServerHandle)
	b.WriteString("\nexport interface ForstTestServer {\n")
	b.WriteString("  readonly baseUrl: string;\n")
	b.WriteString("  readonly port: number;\n")
	b.WriteString("  readonly pid?: number;\n")
	b.WriteString("  readonly connection: \"spawn\" | \"connect\";\n")
	b.WriteString("  stop(): Promise<void>;\n")
	b.WriteString("  [Symbol.asyncDispose](): Promise<void>;\n")
	b.WriteString("}\n\n")
}

// emitTestServerStartHelperESM writes the shared lazy @forst/cli/invoke import helper.
func emitTestServerStartHelperESM(b *strings.Builder) {
	fmt.Fprintf(b, "const CLI_INVOKE = %q;\n", CliInvokeModuleSpecifier)
	fmt.Fprintf(b, "const CLI_INSTALL = %q;\n\n", CliInstallCommand)
	b.WriteString(`async function importCliInvoke() {
  try {
    return await import(CLI_INVOKE);
  } catch (cause) {
    const causeMessage =
      cause && typeof cause === "object" && "message" in cause
        ? String(cause.message)
        : String(cause);
    throw new ForstTestServerFailed({
      reason: "cli_missing",
      installCommand: CLI_INSTALL,
      causeMessage,
      message:
        "Install @forst/cli to start a real Forst invoke server in tests: " +
        CLI_INSTALL,
    });
  }
}

async function startInvokeServerHandle(options) {
  const mod = await importCliInvoke();
  if (typeof mod.startForstInvokeServer !== "function") {
    throw new ForstTestServerFailed({
      reason: "cli_missing",
      installCommand: CLI_INSTALL,
      causeMessage: "startForstInvokeServer export missing",
      message:
        "Install @forst/cli (>=0.2.0) with the ./invoke subpath: " + CLI_INSTALL,
    });
  }
  try {
    return await mod.startForstInvokeServer(options);
  } catch (cause) {
    if (cause instanceof ForstTestServerFailed) {
      throw cause;
    }
    const causeMessage =
      cause && typeof cause === "object" && "message" in cause
        ? String(cause.message)
        : String(cause);
    const reason =
      /timeout|timed out/i.test(causeMessage) ? "ready_timeout" :
      /ECONNREFUSED|unreachable/i.test(causeMessage) ? "unreachable" :
      "spawn_failed";
    throw new ForstTestServerFailed({
      reason,
      causeMessage,
      message: causeMessage || "failed to start Forst invoke server",
    });
  }
}
`)
}
