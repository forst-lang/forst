/** Default pause before sending textDocument/didChange after the last keystroke. */
export const DID_CHANGE_DEBOUNCE_MS = 200;

/** Snapshot of a document at the moment a debounced didChange should fire. */
export type DidChangePayload = {
  uri: string;
  version: number;
  text: string;
  generation: number;
};

type DidChangeSchedule = Omit<DidChangePayload, "generation">;

type PendingEntry = {
  timer: ReturnType<typeof setTimeout>;
  payload: DidChangePayload;
};

/**
 * Per-URI debounce for didChange: waits `delayMs` after the last schedule, then
 * invokes `send` with the latest payload. Tracks the highest version whose
 * diagnostics were applied so a slow response cannot overwrite a newer one.
 * A per-URI generation invalidates in-flight sends after close or restart.
 */
export class DidChangeDebouncer {
  private readonly pending = new Map<string, PendingEntry>();
  private readonly lastAppliedVersion = new Map<string, number>();
  private readonly uriEpoch = new Map<string, number>();
  private globalEpoch = 0;

  constructor(
    private readonly send: (payload: DidChangePayload) => Promise<void>,
    private readonly delayMs: number = DID_CHANGE_DEBOUNCE_MS,
    private readonly setTimer: typeof setTimeout = setTimeout,
    private readonly clearTimer: typeof clearTimeout = clearTimeout
  ) {}

  /** Current generation for `uri`; in-flight payloads with an older generation must not apply. */
  generationOf(uri: string): number {
    return (this.uriEpoch.get(uri) ?? 0) + this.globalEpoch;
  }

  /** Returns true when `generation` still matches the live generation for `uri`. */
  isCurrentGeneration(uri: string, generation: number): boolean {
    return generation === this.generationOf(uri);
  }

  /** Schedules a didChange for `payload.uri`, replacing any pending timer for that URI. */
  schedule(payload: DidChangeSchedule): void {
    const stamped: DidChangePayload = {
      ...payload,
      generation: this.generationOf(payload.uri),
    };
    const prev = this.pending.get(payload.uri);
    if (prev) {
      this.clearTimer(prev.timer);
    }
    const timer = this.setTimer(() => {
      this.pending.delete(payload.uri);
      void this.flush(stamped);
    }, this.delayMs);
    this.pending.set(payload.uri, { timer, payload: stamped });
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

  /** Cancels a pending timer, clears applied-version tracking, and invalidates in-flight sends for `uri`. */
  clear(uri: string): void {
    const prev = this.pending.get(uri);
    if (prev) {
      this.clearTimer(prev.timer);
      this.pending.delete(uri);
    }
    this.lastAppliedVersion.delete(uri);
    this.uriEpoch.set(uri, (this.uriEpoch.get(uri) ?? 0) + 1);
  }

  /** Cancels all pending timers and invalidates every in-flight generation (e.g. on language-server restart). */
  clearAll(): void {
    for (const entry of this.pending.values()) {
      this.clearTimer(entry.timer);
    }
    this.pending.clear();
    this.lastAppliedVersion.clear();
    this.globalEpoch += 1;
  }

  private async flush(payload: DidChangePayload): Promise<void> {
    // Skip obsolete payloads if a newer version was already applied (overlapping sends).
    if (!this.shouldApply(payload.uri, payload.version)) {
      return;
    }
    if (!this.isCurrentGeneration(payload.uri, payload.generation)) {
      return;
    }
    await this.send(payload);
  }
}
