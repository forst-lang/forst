/**
 * Promise-mode client: InvokeFailure tags must be catchTag-compatible in Effect code.
 * Generated package must not import effect.
 */
import { Effect } from "effect";
import type { InvokeFailure } from "@forst/gen/invoke";

declare const program: Effect.Effect<void, InvokeFailure>;
const _ = program.pipe(
  Effect.catchTag("@forst/InvokeTimedOut", () => Effect.void)
);
void _;
