import semver from "semver";

/**
 * True when local and remote compiler versions should be treated as matching for sidecar checks.
 * Uses semver equality when both sides coerce to valid semver; otherwise falls back to exact string match.
 */
export function versionsEquivalentForSidecar(
  local: string,
  remote: string
): boolean {
  const l = local.trim();
  const r = remote.trim();
  if (l === r) {
    return true;
  }
  const lv = semver.coerce(l);
  const rv = semver.coerce(r);
  if (
    lv &&
    rv &&
    semver.valid(lv) &&
    semver.valid(rv)
  ) {
    return semver.eq(lv, rv);
  }
  return false;
}

/**
 * True when the server's `contractVersion` is compatible with this package (same major string for now).
 */
export function contractVersionCompatible(
  serverContractVersion: string,
  expected: string
): boolean {
  const a = serverContractVersion.trim();
  const b = expected.trim();
  if (a === b) {
    return true;
  }
  const av = semver.coerce(a);
  const bv = semver.coerce(b);
  if (av && bv && semver.valid(av) && semver.valid(bv)) {
    return semver.eq(av, bv);
  }
  return a === b;
}
