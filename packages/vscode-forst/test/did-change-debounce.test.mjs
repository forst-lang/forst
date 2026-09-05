import assert from "node:assert";
import test from "node:test";
import { fileURLToPath } from "node:url";
import path from "node:path";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const { DidChangeDebouncer, DID_CHANGE_DEBOUNCE_MS } = await import(
  path.join(__dirname, "..", "out", "didChangeDebounce.js")
);

test("DID_CHANGE_DEBOUNCE_MS is 200", () => {
  assert.strictEqual(DID_CHANGE_DEBOUNCE_MS, 200);
});

test("DidChangeDebouncer sends latest payload after delay and drops superseded timers", async () => {
  const sent = [];
  /** @type {Map<ReturnType<typeof setTimeout>, () => void>} */
  const timers = new Map();
  let nextId = 1;

  const setTimer = (fn, _ms) => {
    const id = nextId++;
    timers.set(id, fn);
    return id;
  };
  const clearTimer = (id) => {
    timers.delete(id);
  };
  const fireAll = () => {
    const fns = [...timers.values()];
    timers.clear();
    for (const fn of fns) {
      fn();
    }
  };

  const debouncer = new DidChangeDebouncer(
    async (payload) => {
      sent.push({ ...payload });
    },
    200,
    setTimer,
    clearTimer
  );

  debouncer.schedule({ uri: "file:///a.ft", version: 1, text: "a" });
  debouncer.schedule({ uri: "file:///a.ft", version: 2, text: "ab" });
  assert.strictEqual(timers.size, 1, "only one pending timer per URI");

  fireAll();
  await Promise.resolve();
  await Promise.resolve();

  assert.deepStrictEqual(sent, [
    { uri: "file:///a.ft", version: 2, text: "ab" },
  ]);
});

test("DidChangeDebouncer.shouldApply drops older versions after markApplied", () => {
  const debouncer = new DidChangeDebouncer(async () => {}, 200, () => 1, () => {});
  assert.strictEqual(debouncer.shouldApply("file:///a.ft", 1), true);
  debouncer.markApplied("file:///a.ft", 3);
  assert.strictEqual(debouncer.shouldApply("file:///a.ft", 2), false);
  assert.strictEqual(debouncer.shouldApply("file:///a.ft", 3), true);
  assert.strictEqual(debouncer.shouldApply("file:///a.ft", 4), true);
});

test("DidChangeDebouncer.clear cancels pending and applied tracking", () => {
  /** @type {Map<ReturnType<typeof setTimeout>, () => void>} */
  const timers = new Map();
  let nextId = 1;
  const setTimer = (fn) => {
    const id = nextId++;
    timers.set(id, fn);
    return id;
  };
  const clearTimer = (id) => {
    timers.delete(id);
  };

  const debouncer = new DidChangeDebouncer(async () => {}, 200, setTimer, clearTimer);
  debouncer.schedule({ uri: "file:///a.ft", version: 1, text: "a" });
  debouncer.markApplied("file:///a.ft", 1);
  assert.strictEqual(timers.size, 1);
  debouncer.clear("file:///a.ft");
  assert.strictEqual(timers.size, 0);
  assert.strictEqual(debouncer.shouldApply("file:///a.ft", 1), true);
});
