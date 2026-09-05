/**
 * Browser bootstrap for Mintlify docs. Bundled into docs/forst-highlight.js by sync script.
 */

/* global GRAMMAR, highlightToHtml, isForstLabel, looksLikeForstSource, walkSelfAndAncestors, hasExplicitFtLanguage, hasForstTabContext, isPlainTextFallback */

/**
 * @param {HTMLElement} code
 * @returns {boolean}
 */
function hasForstCodeBlockLabel(code) {
  const root =
    code.closest('[data-component-part="code-block-root"]') ||
    code.closest('[data-component-part="code-group-root"]') ||
    code.closest("pre")?.parentElement;

  if (!root) return false;

  const labelSelectors = [
    '[data-component-part="code-block-language"]',
    '[data-component-part="code-block-title"]',
    '[data-component-part="code-group-tab"]',
    "button[role='tab'][aria-selected='true']",
  ];

  for (const selector of labelSelectors) {
    const el = root.querySelector(selector) || root.parentElement?.querySelector(selector);
    if (el && isForstLabel(el.textContent)) return true;
  }

  const header = root.previousElementSibling;
  if (header instanceof HTMLElement && isForstLabel(header.textContent)) return true;

  return false;
}

/**
 * @param {HTMLElement} code
 * @returns {boolean}
 */
function isForstBlock(code) {
  if (hasExplicitFtLanguage(code)) return true;
  if (hasForstTabContext(code, document)) return true;
  if (hasForstCodeBlockLabel(code)) return true;
  if (isPlainTextFallback(code) && looksLikeForstSource(code.textContent ?? "")) return true;
  return false;
}

/**
 * @param {HTMLElement} code
 * @returns {boolean}
 */
function alreadyHighlighted(code) {
  if (code.dataset.ftHighlighted === "1") return true;
  return Boolean(
    code.querySelector(
      "span.ft-tok-keyword, span.ft-tok-string, span.ft-tok-comment, span.ft-tok-function, span.ft-tok-type"
    )
  );
}

/**
 * @param {HTMLElement} container
 * @param {string} source
 */
function applyHighlightHtml(container, source) {
  container.innerHTML = highlightToHtml(source, GRAMMAR);
}

/**
 * @param {HTMLElement} code
 */
function highlightBlock(code) {
  if (!isForstBlock(code) || alreadyHighlighted(code)) return;

  const lineEls = code.querySelectorAll(":scope > .line");
  if (lineEls.length > 0) {
    lineEls.forEach((line) => {
      if (!(line instanceof HTMLElement)) return;
      applyHighlightHtml(line, line.textContent ?? "");
    });
  } else {
    applyHighlightHtml(code, code.textContent ?? "");
  }

  code.dataset.ftHighlighted = "1";
}

/**
 * @param {ParentNode} root
 */
function highlightAll(root) {
  root.querySelectorAll("pre code").forEach((code) => {
    if (code instanceof HTMLElement) highlightBlock(code);
  });
}

function init() {
  highlightAll(document);
  const observer = new MutationObserver((mutations) => {
    for (const mutation of mutations) {
      mutation.addedNodes.forEach((node) => {
        if (node instanceof HTMLElement) highlightAll(node);
      });
    }
  });
  observer.observe(document.body, { childList: true, subtree: true });
}

if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", init);
} else {
  init();
}
