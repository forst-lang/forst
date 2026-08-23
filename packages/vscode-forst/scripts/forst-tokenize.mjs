/**
 * Lightweight TextMate-subset highlighter for Forst docs (no Oniguruma).
 * Driven by docs/languages/forst.json grammar shape.
 */

/** @typedef {{ start: number, end: number, scope: string }} Token */

/**
 * @param {string} source
 * @param {object} grammar
 * @returns {Token[]}
 */
export function tokenize(source, grammar) {
  /** @type {Token[]} */
  const tokens = [];
  const patterns = resolvePatterns(grammar.patterns, grammar.repository);
  scan(source, 0, source.length, patterns, grammar.repository, tokens);
  return tokens;
}

/**
 * @param {string} scope
 * @returns {string}
 */
export function scopeToClass(scope) {
  if (!scope || scope === "source.forst") return "ft-tok-plain";
  if (scope.includes("comment")) return "ft-tok-comment";
  if (scope.includes("string")) return "ft-tok-string";
  if (scope.includes("keyword.declaration") || scope.includes("keyword.control") || scope.includes("keyword.other")) {
    return "ft-tok-keyword";
  }
  if (scope.includes("storage.type")) return "ft-tok-keyword";
  if (scope.includes("entity.name.function")) return "ft-tok-function";
  if (scope.includes("entity.name.type")) return "ft-tok-type";
  if (scope.includes("support.type")) return "ft-tok-type";
  if (scope.includes("constant.language") || scope.includes("constant.numeric") || scope.includes("constant.character")) {
    return "ft-tok-constant";
  }
  if (scope.includes("variable")) return "ft-tok-variable";
  if (scope.includes("keyword.operator")) return "ft-tok-punct";
  return "ft-tok-plain";
}

/**
 * @param {string} source
 * @param {Token[]} tokens
 * @returns {string}
 */
export function renderHtml(source, tokens) {
  if (!source) return "";
  const scopes = new Array(source.length).fill("source.forst");
  const sorted = [...tokens].sort((a, b) => b.end - b.start - (a.end - a.start));
  for (const token of sorted) {
    for (let i = token.start; i < token.end && i < source.length; i++) {
      scopes[i] = token.scope;
    }
  }

  let html = "";
  let i = 0;
  while (i < source.length) {
    const cls = scopeToClass(scopes[i]);
    let j = i + 1;
    while (j < source.length && scopeToClass(scopes[j]) === cls) j++;
    const chunk = source.slice(i, j);
    if (cls === "ft-tok-plain") {
      html += escapeHtml(chunk);
    } else {
      html += `<span class="${cls}">${escapeHtml(chunk)}</span>`;
    }
    i = j;
  }
  return html;
}

/**
 * @param {string} source
 * @param {object} grammar
 * @returns {string}
 */
export function highlightToHtml(source, grammar) {
  return renderHtml(source, tokenize(source, grammar));
}

/**
 * @param {string} text
 * @returns {string}
 */
export function escapeHtml(text) {
  return text
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}

/**
 * @param {object[]} patterns
 * @param {Record<string, object>} repository
 * @returns {object[]}
 */
function resolvePatterns(patterns, repository) {
  /** @type {object[]} */
  const resolved = [];
  for (const pattern of patterns ?? []) {
    if (pattern.include) {
      const ref = pattern.include;
      if (ref.startsWith("#")) {
        const key = ref.slice(1);
        const repo = repository[key];
        if (!repo) continue;
        // A repo rule with match/begin is one pattern (inner `patterns` are nested).
        // A repo that only has `patterns` is a group — flatten it.
        if (repo.match || repo.begin) {
          resolved.push(repo);
        } else if (repo.patterns) {
          resolved.push(...resolvePatterns(repo.patterns, repository));
        }
      }
    } else {
      resolved.push(pattern);
    }
  }
  return resolved;
}

/**
 * @param {string} source
 * @param {number} start
 * @param {number} end
 * @param {object[]} patterns
 * @param {Record<string, object>} repository
 * @param {Token[]} tokens
 */
function scan(source, start, end, patterns, repository, tokens) {
  let pos = start;
  while (pos < end) {
    const slice = source.slice(start, end);
    const rel = pos - start;
    let matched = false;

    for (const pattern of patterns) {
      if (pattern.match) {
        const re = new RegExp(pattern.match, "yd");
        re.lastIndex = rel;
        const match = re.exec(slice);
        if (!match || match.index !== rel) continue;
        const absStart = start + rel;
        const absEnd = absStart + match[0].length;
        applyMatchTokens(pattern, match, absStart, tokens);
        pos = absEnd;
        matched = true;
        break;
      }

      if (pattern.begin && pattern.end) {
        const beginRe = new RegExp(pattern.begin, "yd");
        beginRe.lastIndex = rel;
        const beginMatch = beginRe.exec(slice);
        if (!beginMatch || beginMatch.index !== rel) continue;

        const openStart = start + rel;
        const contentStart = openStart + beginMatch[0].length;
        const inner = resolvePatterns(pattern.patterns ?? [], repository);
        const closeEnd = scanUntilEnd(
          source,
          contentStart,
          end,
          pattern.end,
          inner,
          repository,
          tokens
        );
        applyBeginEndTokens(pattern, beginMatch, openStart, closeEnd, tokens);

        pos = closeEnd;
        matched = true;
        break;
      }
    }

    if (!matched) pos++;
  }
}

/**
 * @param {string} source
 * @param {number} contentStart
 * @param {number} end
 * @param {string} endPattern
 * @param {object[]} innerPatterns
 * @param {Record<string, object>} repository
 * @param {Token[]} tokens
 * @returns {number}
 */
function scanUntilEnd(source, contentStart, end, endPattern, innerPatterns, repository, tokens) {
  const endRe = new RegExp(endPattern, "yd");
  let pos = contentStart;
  while (pos < end) {
    const slice = source.slice(pos, end);
    endRe.lastIndex = 0;
    const endMatch = endRe.exec(slice);
    if (endMatch?.index === 0) {
      return pos + endMatch[0].length;
    }

    let matched = false;
    for (const inner of innerPatterns) {
      if (inner.match) {
        const re = new RegExp(inner.match, "yd");
        re.lastIndex = 0;
        const match = re.exec(slice);
        if (!match || match.index !== 0) continue;
        applyMatchTokens(inner, match, pos, tokens);
        pos += match[0].length;
        matched = true;
        break;
      }
      if (inner.begin && inner.end) {
        const beginRe = new RegExp(inner.begin, "yd");
        beginRe.lastIndex = 0;
        const beginMatch = beginRe.exec(slice);
        if (!beginMatch || beginMatch.index !== 0) continue;
        const openStart = pos;
        const nestedStart = openStart + beginMatch[0].length;
        const nestedInner = resolvePatterns(inner.patterns ?? [], repository);
        pos = scanUntilEnd(source, nestedStart, end, inner.end, nestedInner, repository, tokens);
        applyBeginEndTokens(inner, beginMatch, openStart, pos, tokens);
        matched = true;
        break;
      }
    }

    if (!matched) pos++;
  }
  return end;
}

/**
 * @param {object} pattern
 * @param {RegExpExecArray} beginMatch
 * @param {number} openStart
 * @param {number} closeEnd
 * @param {Token[]} tokens
 */
function applyBeginEndTokens(pattern, beginMatch, openStart, closeEnd, tokens) {
  if (pattern.name) {
    tokens.push({ start: openStart, end: closeEnd, scope: pattern.name });
  }
  applyCaptureTokens(pattern.beginCaptures ?? pattern.captures, beginMatch, openStart, tokens, undefined);
}

/**
 * @param {object} pattern
 * @param {RegExpExecArray} match
 * @param {number} offset
 * @param {Token[]} tokens
 */
function applyMatchTokens(pattern, match, offset, tokens) {
  if (pattern.name) {
    tokens.push({ start: offset, end: offset + match[0].length, scope: pattern.name });
  }
  applyCaptureTokens(pattern.captures, match, offset, tokens, undefined);
}

/**
 * @param {Record<string, { name?: string }> | undefined} captures
 * @param {RegExpExecArray} match
 * @param {number} offset
 * @param {Token[]} tokens
 * @param {string | undefined} fallbackName
 */
function applyCaptureTokens(captures, match, offset, tokens, fallbackName) {
  if (fallbackName) {
    tokens.push({ start: offset, end: offset + match[0].length, scope: fallbackName });
  }
  if (!captures) return;

  const indices = match.indices;
  for (const [idx, capture] of Object.entries(captures)) {
    const i = Number(idx);
    if (!capture?.name) continue;
    if (indices?.[i]) {
      const [rawStart, rawEnd] = indices[i];
      if (rawStart !== undefined && rawEnd !== undefined && rawEnd > rawStart) {
        // `match.indices` are into the exec'd string, not relative to the match.
        tokens.push({
          start: offset + (rawStart - match.index),
          end: offset + (rawEnd - match.index),
          scope: capture.name,
        });
        continue;
      }
    }
    if (i <= 0 || match[i] == null) continue;
    const part = match[i];
    const localStart = match[0].indexOf(part);
    if (localStart < 0) continue;
    tokens.push({
      start: offset + localStart,
      end: offset + localStart + part.length,
      scope: capture.name,
    });
  }
}
