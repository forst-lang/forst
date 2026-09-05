/** Default pause before sending textDocument/didChange after the last keystroke. */
export const DID_CHANGE_DEBOUNCE_MS = 200;

/** Snapshot of a document at the moment a debounced didChange should fire. */
export type DidChangePayload = {
  uri: string;
  version: number;
  text: string;
};

type PendingEntry = {
  timer: ReturnType<typeof setTimeout>;
  payload: DidChangePayload;
};

/**
 * Per-URI debounce for didChange: waits `delayMs` after the last schedule, then
 * invokes `send` with the latest payload. Tracks the highest version whose
 * diagnostics were applied so a slow response cannot overwrite a newer one.
 */
export class DidChangeDebouncer {
  private readonly pending = new Map<string, PendingEntry>();
  private readonly lastAppliedVersion = new Map<string, number>();

  constructor(
    private readonly send: (payload: DidChangePayload) => Promise<void>,
    private readonly delayMs: number = DID_CHANGE_DEBOUNCE_MS,
    private readonly setTimer: typeof setTimeout = setTimeout,
    private readonly clearTimer: typeof clearTimeout = clearTimeout
  ) {}

  /** Schedules a didChange for `payload.uri`, replacing any pending timer for that URI. */
  schedule(payload: DidChangePayload): void {
    const prev = this.pending.get(payload.uri);
    if (prev) {
      this.clearTimer(prev.timer);
    }
    const timer = this.setTimer(() => {
      this.pending.delete(payload.uri);
      void this.flush(payload);
    }, this.delayMs);
    this.pending.set(payload.uri, { timer, payload });
  }

  /**
   * Returns true when diagnostics for `version` should be applied (not older
   * than the last successfully applied version for this URI).
   */
  shouldApply(uri: string, version: number): boolean {
    const last = this.lastAppliedVersion.get(uri);
    if (last === undefined) {
      return true;
    }
    return version >= last;
  }

  /** Records that diagnostics for `version` were applied to `uri`. */
  markApplied(uri: string, version: number): void {
    const last = this.lastAppliedVersion.get(uri);
    if (last === undefined || version >= last) {
      this.lastAppliedVersion.set(uri, version);
    }
  }

  /** Cancels a pending timer and clears applied-version tracking for `uri`. */
  clear(uri: string): void {
    const prev = this.pending.get(uri);
    if (prev) {
      this.clearTimer(prev.timer);
      this.pending.delete(uri);
    }
    this.lastAppliedVersion.delete(uri);
  }

  /** Cancels all pending timers (e.g. on language-server restart). */
  clearAll(): void {
    for (const entry of this.pending.values()) {
      this.clearTimer(entry.timer);
    }
    this.pending.clear();
    this.lastAppliedVersion.clear();
  }

  private async flush(payload: DidChangePayload): Promise<void> {
    // Skip obsolete payloads if a newer version was already applied (overlapping sends).
    if (!this.shouldApply(payload.uri, payload.version)) {
      return;
    }
    await this.send(payload);
  }
}
