# Forst &emsp; [![CI]][actions] [![Release]][release] [![Go Report Card]][goreport] [![Coverage Status]][coveralls] [![License]][license]

[CI]: https://img.shields.io/github/actions/workflow/status/forst-lang/forst/lint-test-coverage.yml
[actions]: https://github.com/forst-lang/forst/actions
[release]: https://img.shields.io/github/v/release/forst-lang/forst?filter=v*
[Go Report Card]: https://goreportcard.com/badge/github.com/forst-lang/forst
[goreport]: https://goreportcard.com/report/github.com/forst-lang/forst
[Coverage Status]: https://coveralls.io/repos/github/forst-lang/forst/badge.svg?branch=main
[coveralls]: https://coveralls.io/github/forst-lang/forst?branch=main
[License]: https://img.shields.io/github/license/forst-lang/forst

**Forst is a programming language that brings TypeScript's type safety and developer experience to Go.**

Its primary goal is to help you move away from TypeScript on the backend:

- Generate instantly usable TypeScript types from backend endpoints – enabling full-stack development without build steps.
- Strong static typing with aggressive inference and smart narrowing – so you move fast while staying safe.
- Data schemas acting as guards, automatically validating deeply nested input data through type definitions – to keep untrusted user input out of your application logic.

See also [ROADMAP.md](./ROADMAP.md) for planned work and **feature parity**.

## Why?

We love TypeScript's efficiency in structuring data.

We love Go's efficiency at compile and runtime.

We want the best of both worlds.

We want to be to Go what TypeScript is to JavaScript.

## Examples

Side-by-side comparisons with Go and TypeScript, then snippets that follow one **user signup** path: you accept structured input, attach domain-specific failures, assert invariants with `ensure`, and narrow `Result` values at call sites.

### Validation (Go → Forst)

<table>
<thead>
<tr>
<th align="left">Before (Go)</th>
<th align="left">After (Forst)</th>
</tr>
</thead>
<tbody>
<tr>
<td valign="top" width="50%">

Structs, struct tags, and separate validation logic.

<pre><code class="language-go">package example

import "errors"

type Register struct {
	Name  string `json:"name" validate:"required,min=3"`
	Phone string `json:"phone"`
}

func (r Register) Validate() error {
	if len(r.Name) &lt; 3 {
		return errors.New("name too short")
	}
	return nil
}</code></pre>

</td>
<td valign="top" width="50%">

Constraints on the type; invalid data is rejected with the same model the typechecker uses.

<pre><code class="language-forst">type PhoneNumber =
  String.Min(3).Max(10) &amp; (
    String.HasPrefix("+")
    | String.HasPrefix("0")
  )

func registerUser(op: Mutation.Input({
  name: String.Min(3).Max(10),
  phoneNumber: PhoneNumber,
})) {
  fmt.Printf("Registering user %s\n", op.input.name)
}</code></pre>

</td>
</tr>
</tbody>
</table>

### Types across the wire (TypeScript → Forst)

<table>
<thead>
<tr>
<th align="left">Before (TypeScript)</th>
<th align="left">After (Forst)</th>
</tr>
</thead>
<tbody>
<tr>
<td valign="top" width="50%">

<code>interface</code> and a runtime schema (e.g. Zod)—two layers to keep aligned.

<pre><code class="language-typescript">import { z } from "zod";

interface RegisterInput {
  name: string;
  phone: string;
}

const RegisterSchema = z.object({
  name: z.string().min(3),
  phone: z.string().regex(/^(\+|0)/),
});</code></pre>

</td>
<td valign="top" width="50%">

One definition; <code>forst generate</code> emits TypeScript declarations (and a small client) from your <code>.ft</code> sources so the front end imports the same shapes as the server.

<pre><code class="language-forst">func registerUser(op: Mutation.Input({
  name: String.Min(3).Max(10),
  phoneNumber: PhoneNumber,
})) {
  // …
}</code></pre>

</td>
</tr>
</tbody>
</table>

### Hello World

Forst accepts ordinary Go: the same program compiles as Go or as Forst.

<table>
<thead>
<tr>
<th align="left">Before (Go)</th>
<th align="left">After (Forst)</th>
</tr>
</thead>
<tbody>
<tr>
<td valign="top" width="50%">

Standard <code>package main</code> using the Go toolchain.

<pre><code class="language-go">package main

import "fmt"

func main() {
	fmt.Println("Hello World!")
}</code></pre>

</td>
<td valign="top" width="50%">

Same source as Forst—no rewrite for a minimal program.

<pre><code class="language-forst">package main

import "fmt"

func main() {
  fmt.Println("Hello World!")
}</code></pre>

</td>
</tr>
</tbody>
</table>

### Basic Function

Go requires explicit return types; Forst can infer them. Open seats before accepting a new registration:

<table>
<thead>
<tr>
<th align="left">Before (Go)</th>
<th align="left">After (Forst)</th>
</tr>
</thead>
<tbody>
<tr>
<td valign="top" width="50%">

<pre><code class="language-go">func spotsLeft(capacity, registered int) int {
	return capacity - registered
}</code></pre>

</td>
<td valign="top" width="50%">

<pre><code class="language-forst">func spotsLeft(capacity Int, registered Int) {
  return capacity - registered
}</code></pre>

</td>
</tr>
</tbody>
</table>

### Input Validation

Go: separate structs, tags, and ad hoc checks. Forst: refinements on types plus a single input shape—compile-time and runtime stay aligned.

<table>
<thead>
<tr>
<th align="left">Before (Go)</th>
<th align="left">After (Forst)</th>
</tr>
</thead>
<tbody>
<tr>
<td valign="top" width="50%">

<pre><code class="language-go">package example

import (
	"errors"
	"fmt"
	"strings"
)

type RegisterUserInput struct {
	ID          string
	Name        string
	PhoneNumber string
	BankAccount struct{ IBAN string }
}

func validPhone(s string) bool {
	n := len(s)
	return n &gt;= 3 &amp;&amp; n &lt;= 10 &amp;&amp;
		(strings.HasPrefix(s, "+") ||
			strings.HasPrefix(s, "0"))
}

func RegisterUser(op RegisterUserInput) error {
	n := len(op.Name)
	if n &lt; 3 || n &gt; 10 {
		return errors.New("invalid name")
	}
	iban := len(op.BankAccount.IBAN)
	if iban &lt; 10 || iban &gt; 34 {
		return errors.New("invalid iban")
	}
	if !validPhone(op.PhoneNumber) {
		return errors.New("invalid phone")
	}
	fmt.Printf("Registering user %s (%s)\n",
		op.Name, op.ID)
	return nil
}</code></pre>

</td>
<td valign="top" width="50%">

<pre><code class="language-forst">type PhoneNumber =
  String.Min(3).Max(10) &amp; (
    String.HasPrefix("+")
    | String.HasPrefix("0")
  )

func registerUser(op: Mutation.Input({
  id: UUID.V4(),
  name: String.Min(3).Max(10),
  phoneNumber: PhoneNumber,
  bankAccount: {
    iban: String.Min(10).Max(34),
  },
})) {
  fmt.Printf("Registering user %s (%s)\n", op.input.name, op.input.id)
}</code></pre>

</td>
</tr>
</tbody>
</table>

### Nominal errors

Go models named errors with a struct and <code>Error()</code>. Forst uses <code>error Name { … }</code> as a first-class declaration.

<table>
<thead>
<tr>
<th align="left">Before (Go)</th>
<th align="left">After (Forst)</th>
</tr>
</thead>
<tbody>
<tr>
<td valign="top" width="50%">

<pre><code class="language-go">type NameTooShort struct {
	Message string
}

func (e NameTooShort) Error() string {
	return e.Message
}</code></pre>

</td>
<td valign="top" width="50%">

<pre><code class="language-forst">error NameTooShort {
  message: String
}</code></pre>

</td>
</tr>
</tbody>
</table>

### Ensure

Go: explicit branches; typical style is <code>errors.New</code> or <code>fmt.Errorf</code>—callers match on strings or wrap, not on a dedicated failure type tied to the check. Forst: <code>ensure</code> pairs the refinement with a nominal error in one place.

<table>
<thead>
<tr>
<th align="left">Before (Go)</th>
<th align="left">After (Forst)</th>
</tr>
</thead>
<tbody>
<tr>
<td valign="top" width="50%">

<pre><code class="language-go">package example

import "errors"

func validateDisplayName(name string) error {
	if len(name) &lt; 3 {
		return errors.New(
			"name must be at least 3 characters",
		)
	}
	return nil
}</code></pre>

</td>
<td valign="top" width="50%">

<pre><code class="language-forst">func validateDisplayName(name String) {
  ensure name is Min(3) or NameTooShort({
    message: "name must be at least 3 characters",
  })
}</code></pre>

</td>
</tr>
</tbody>
</table>

### Result and narrowing

Go: <code>(T, error)</code> and manual checks. Forst: <code>Result</code> plus <code>ensure</code> so success paths narrow the value.

<table>
<thead>
<tr>
<th align="left">Before (Go)</th>
<th align="left">After (Forst)</th>
</tr>
</thead>
<tbody>
<tr>
<td valign="top" width="50%">

<pre><code class="language-go">package main

import (
	"errors"
	"fmt"
)

func allocateUserID() (int, error) {
	id := 1001
	if id &lt;= 0 {
		return 0, errors.New("invalid id")
	}
	return id, nil
}

func main() {
	x, err := allocateUserID()
	if err != nil {
		panic(err)
	}
	fmt.Println(x)
}</code></pre>

</td>
<td valign="top" width="50%">

<pre><code class="language-forst">func allocateUserID() {
  id := 1001
  ensure id is GreaterThan(0)
  return id
}

func main() {
  x := allocateUserID()
  ensure x is Ok()
  println(x) // x is guaranteed to be an Int here
}</code></pre>

</td>
</tr>
</tbody>
</table>

## Features

- Static typing
- Strong type inference
- Backwards compatibility with Go
- Seamless TypeScript type generation inspired by tRPC – publishing types of API endpoints should be easy
- Structural typing for function parameters and return values
- Type-based assertions that allow complex type narrowing in function parameters
- First-class scoped errors including stack traces, avoiding exceptions
- No class or module reopening

## Design Philosophy

See also [PHILOSOPHY.md](./PHILOSOPHY.md) for what guides and motivates us.

## Development

Install Task using the [official instructions](https://taskfile.dev/installation/).

Run tests:

```bash
task test                  # Run all tests
task test:unit             # Run compiler unit tests
task test:unit:parser      # Run parser tests
task test:unit:typechecker # Run typechecker tests
```

Run examples:

```bash
task test:integration                    # Run compilation examples / integration tests
task example -- ../examples/in/basic.ft  # Run specific example
task example:function                    # Run function example
```

## VS Code

The workspace includes an optional extension in [`packages/vscode-forst`](./packages/vscode-forst): it registers `.ft` and talks to the compiler’s HTTP LSP (`forst lsp`) for diagnostics. Its **release cadence is separate** from compiler `v*` tags (see `vscode-forst-v*` in [`.github/workflows/publish-vscode-extension.yml`](./.github/workflows/publish-vscode-extension.yml)). After `bun install` at the repo root, run `task build:vscode` to compile it (or rely on the F5 **preLaunchTask** in [`.vscode/launch.json`](./.vscode/launch.json)). CI runs the same compile as the first step of `task ci:test`. See [`packages/vscode-forst/README.md`](./packages/vscode-forst/README.md) for F5 and troubleshooting.

## npm

**[`@forst/cli`](./packages/cli/README.md)** installs the Forst compiler in JS/TS projects: `npm i -D @forst/cli`, then `npx forst` / `node_modules/.bin/forst` (it pulls the matching native binary from GitHub Releases). For the dev-server + HTTP client, use **[`@forst/sidecar`](./packages/sidecar/README.md)** instead.

## TypeScript client output

You can generate **TypeScript types and a small client** from your Forst code so front ends or Node callers get the same shapes your server uses, without copying types by hand.

Run `forst generate` with a `.ft` file or a folder of `.ft` files; it writes a `generated/` tree (declarations plus helpers) and a `client/` stub you can wire to your app. The dev server can also expose types over HTTP while you iterate.

## Inspirations

Our primary inspiration is TypeScript's structural type system and its enormous success in making JavaScript development more ergonomic, robust and gradually typeable. We aim to bring similar benefits to Go development, insofar as they are not already present.

We also draw inspiration from:

- **Zod** — constraints and shape guards as composable runtime checks on nested data.
- **tRPC** — one source of truth for API shapes, with **TypeScript types and a small client** generated from Forst (`forst generate`, `examples/client-integration/`).
- **Go** and **Rust** — **errors as values** and explicit control flow (`ensure` … `or …`), not exceptions.