import type { Readable } from "node:stream";

export const envInvokeAuthFd = "FORST_INVOKE_AUTH_FD";

export interface AuthHandoff {
  generation: number;
  token: Uint8Array;
}

interface AuthHandoffPayload {
  generation?: number;
  token?: string;
}

export async function readAuthHandoffFromStream(
  stream: Readable
): Promise<AuthHandoff> {
  const line = await readFirstLine(stream);
  let payload: AuthHandoffPayload;
  try {
    payload = JSON.parse(line) as AuthHandoffPayload;
  } catch {
    throw new Error("invoke auth handoff: invalid JSON");
  }
  if (
    payload.generation === undefined ||
    typeof payload.generation !== "number" ||
    !Number.isSafeInteger(payload.generation) ||
    payload.generation < 0 ||
    typeof payload.token !== "string" ||
    payload.token.trim() === ""
  ) {
    throw new Error("invoke auth handoff: missing generation or token");
  }
  return {
    generation: payload.generation,
    token: Uint8Array.from(Buffer.from(payload.token, "base64url")),
  };
}

function readFirstLine(stream: Readable): Promise<string> {
  return new Promise((resolve, reject) => {
    let buffer = "";
    const onData = (chunk: Buffer | string) => {
      buffer += chunk.toString();
      const newline = buffer.indexOf("\n");
      if (newline >= 0) {
        cleanup();
        resolve(buffer.slice(0, newline));
      }
    };
    const onEnd = () => {
      cleanup();
      if (buffer.length > 0) {
        resolve(buffer);
        return;
      }
      reject(new Error("invoke auth handoff: stream closed before payload"));
    };
    const onError = (error: Error) => {
      cleanup();
      reject(error);
    };
    const cleanup = () => {
      stream.off("data", onData);
      stream.off("end", onEnd);
      stream.off("error", onError);
    };
    stream.on("data", onData);
    stream.on("end", onEnd);
    stream.on("error", onError);
  });
}
