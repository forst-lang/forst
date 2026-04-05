```
---
name: generate-commit-message
description: >-
  Produces a conventional semantic commit message from staged changes by reading
  git diff --cached. When the change is a feature or bugfix, adds a minimal
  illustrative snippet (e.g. Forst, Go, or config) that demonstrates the change.
  Use when the user asks for a commit message, semantic commit, or summary of
  staged files, or invokes /generate-commit-message.
---

## Introduction

Use this skill to generate a **commit message from staged changes**. The agent must **always read the cached diff** (`git diff --cached`) before writing a message. Output should follow **Conventional Commits** format, and for feature or bugfix commits, provide a **minimal illustrative example** (in Forst or whatever language/layer applies) if it helps clarify the change.

## Workflow

1. **Inspect staged changes (required):**
   - Run from repo root:
     ```bash
     git diff --cached --stat
     git diff --cached
     ```
   - If nothing is staged, say so and suggest `git add` or, if the user requests, offer to use `git diff` for unstaged changes instead.

2. **Draft a Conventional Commit message:**
   - **Title** (≤72 chars): `type(scope): imperative summary`
   - Types: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`, `perf`, `ci`
   - `scope` is optional and should be a concise area name if helpful (e.g. `typechecker`, `compiler`).

3. **Body** (when useful): Describe the what and why in plain sentences. Don't just repeat the title. Note breaking changes or migration instructions as needed.

4. **Minimal code example (conditional):**
   - For `feat` or `fix` only, append an Example section with a minimal snippet that shows off the new function, type, config, or bugfix behavior.
   - Use the project's main language (e.g. Forst `.ft`), or Go/TypeScript/JSON if more appropriate.
   - Keep the example very short (about 3–12 lines: a single function, type, or config key).
   - Skip the example for non-feature/fix changes unless the user requests.

5. **Never** fabricate or assume diff content, or claim changes to files without reading `git diff --cached`.

## Output Format

**The commit message (including example) must be presented as a single code block; if you include a code example, escape its backticks to avoid breaking the output formatting.**

- **Title:** `feat(scope): short imperative summary`
- **Body:** Motivation, behavior, or rationale (not just what files changed).
- **Example block** (optional, for feat/fix): To include a code example, use triple backticks (```) in the output, but escape each backtick with a backslash (`\```) so that the overall message remains within a single code block. For example:

    \`\`\`forst
    package demo

    func Example(): String {
      return "illustrates the change"
    }
    \`\`\`

## Common Mistakes to Avoid

- Vague titles like `Update code`, `Fix stuff`, `WIP`
- Repeating the title in the body with no extra info
- Including full files or lengthy/bloated examples—keep it tiny and illustrative
