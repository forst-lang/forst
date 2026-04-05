/**
 * Keeps jsr.json "version" in sync with package.json (run after bumping package version).
 */
import { readFileSync, writeFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const dir = dirname(fileURLToPath(import.meta.url));
const root = join(dir, "..");
const pkg = JSON.parse(readFileSync(join(root, "package.json"), "utf8"));
const jsrPath = join(root, "jsr.json");
const jsr = JSON.parse(readFileSync(jsrPath, "utf8"));
jsr.version = pkg.version;
writeFileSync(jsrPath, JSON.stringify(jsr, null, 2) + "\n");
console.log(`jsr.json version set to ${jsr.version}`);
