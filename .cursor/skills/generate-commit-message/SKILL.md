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

Use this skill to generate a **commit message from staged changes**. The agent must **always read the cached diff** (`git diff --cached`) before writing a message. Output should follow **Conventional Commits** format. For any feature or bugfix in the commit, provide a **minimal illustrative example** (using Forst or other relevant language) when it helps clarify the change.

## Workflow

1. **Inspect staged changes (required):**
   - Run from repo root:
     ```bash
     git diff --cached --stat
     git diff --cached
     ```
   - If nothing is staged, say so and suggest `git add`, or, if requested, offer to use `git diff` for unstaged changes instead.

2. **Draft a Conventional Commit message:**
   - For each feature or bugfix implemented (can be more than one per commit), start a new semantic title:
     - `type(scope): imperative summary`
   - Types: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`, `perf`, `ci`
   - The `scope` is optional but should be concise (e.g. `typechecker`, `compiler`).

3. **Body** (when useful): Describe motivation, behavior, and rationale. Do not simply repeat the title. Add notes for breaking changes or migration instructions as needed.

4. **Minimal code example (when applicable):**
   - For each `feat` or `fix` entry, append an Example section with a short snippet that illustrates the relevant behavior.
   - Use the project's main language (e.g. Forst `.ft`) or another relevant format (Go, TypeScript, JSON, etc.).
   - Keep the example very short (approx. 3–12 lines); illustrate a single function, type, or config.
   - Examples must use triple backticks in the output, and those backticks must be **escaped as \`\`\`** (no additional slashes, just one backslash per backtick), so that the message remains inside the outer code block.
     - For example:
         \`\`\`forst
         func MyExample(): String {
           return "preview"
         }
         \`\`\`
   - Omit the example for non-feature/fix changes unless the user explicitly requests to include one.

5. **Never** fabricate or assume diff content, or claim changes to files without reading `git diff --cached`.

## Output Format

**Present the entire commit message in a single code block.**  
- For multiple `feat` or `fix` areas within a single commit, each should receive its own semantic title and body (and example if appropriate) within the same code block, separated by a blank line.
- Example blocks must use escaped backticks (`\`\`\``) *without* extra slashes—just a single backslash per backtick—to avoid breaking the outer code block.

## Common Mistakes to Avoid

- Vague titles like `Update code`, `Fix stuff`, `WIP`
- Repeating the title in the body with no extra info
- Including full files or overly long examples—keep it concise and directly illustrative
