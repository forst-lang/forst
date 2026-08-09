import { readFileSync, writeFileSync } from "node:fs";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const root = join(dirname(fileURLToPath(import.meta.url)), "..");
const pkg = JSON.parse(readFileSync(join(root, "package.json"), "utf8"));
const jsrPath = join(root, "jsr.json");
const jsr = JSON.parse(readFileSync(jsrPath, "utf8"));
jsr.version = pkg.version;
writeFileSync(jsrPath, JSON.stringify(jsr, null, 2) + "\n");
