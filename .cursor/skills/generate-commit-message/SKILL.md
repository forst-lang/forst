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

Use this workflow when the user wants a **commit message from what is already staged**. The agent must **read the cached diff** (`git diff --cached`) before writing anything, then output a **Conventional Commits**–style title and optional body. For **features and fixes**, add a **tiny example** (few lines) in the project’s language when it clarifies the change; skip bloated or irrelevant samples.

# Generate commit message from staged diff

## Workflow

1. **Inspect staged changes** (required). Run from the repo root:
   ```bash
   git diff --cached --stat
   git diff --cached
   ```
   If nothing is staged, say so and suggest `git add` or offer to use `git diff` for unstaged-only (only if the user explicitly wants unstaged).

2. **Draft the message** using [Conventional Commits](https://www.conventionalcommits.org/):
   - **Title** (≤72 chars): `type(scope): imperative summary`
   - Common types: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`, `perf`, `ci`
   - `scope` is optional; use a short area name when it helps (e.g. `typechecker`, `sidecar`, `compiler`)

3. **Body** (when useful): what changed and why in plain sentences—not a file list. Mention breaking changes or migration notes if applicable.

4. **Minimal example** (conditional):
   - For **`feat`** or **`fix`**: append a short subsection **Example** with a tiny snippet showing the new behavior, API, or fix.
   - Prefer the project’s language (e.g. **Forst** `.ft` for this repo) or the layer the change touches (Go, JSON, TS).
   - Keep it **minimal** (roughly 3–12 lines): one function, one type, or one config key—enough to copy-paste or read at a glance.
   - Omit the example for pure refactors, chore, CI-only, or doc typo fixes unless the user asks.

5. **Do not** edit the plan file, invent diffs, or claim files changed without reading `git diff --cached`.

## Output shape

- **Title line:** `feat(scope): short imperative description`
- **Body:** motivation and behavior when useful (not a raw file list).
- **Example block** (optional, for feat/fix): a fenced block in the right language, ~3–12 lines, e.g.:

        package demo

        func Example(): String {
          return "illustrates the change"
        }

## Anti-patterns

- Vague titles: `Update code`, `Fix stuff`, `WIP`
- Body that only repeats the title with no substance
- Overlong examples or whole files—stay tiny
