import { beforeEach } from "bun:test";
import { resetHostInitCacheForTest } from "../src/runtime/lifecycle.js";

beforeEach(() => {
  resetHostInitCacheForTest();
});
