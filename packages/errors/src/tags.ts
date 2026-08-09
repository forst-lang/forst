/** Prefix for every built-in _tag emitted by @forst/errors. */
export const ERRORS_TAG_PREFIX = "@forst/errors";

export function errorsTag(shortName: string): `${typeof ERRORS_TAG_PREFIX}/${string}` {
  return `${ERRORS_TAG_PREFIX}/${shortName}`;
}
