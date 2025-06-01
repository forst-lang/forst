# To Explore

## Fundamental decisions

- Compile down to Go or Zig (interop, compiler performance)

## Distribution Strategy

### Core Principles

- Address genuine, significant pain points in the development workflow
- Minimize implementation friction and cognitive overhead
- Support gradual, risk-free adoption path

### Implementation Approaches

#### TypeScript Ecosystem Integration

- **Package Generation**
  - Comprehensive TypeScript definitions (`d.ts`)
  - Native module support for modern TypeScript environments
- **Architecture Patterns**
  - Binary-first approach: automatically run standalone executables
  - External validation layer through Forst CLI integration
  - Native tRPC router generation and route integration
- **Supported Environments**
  - Node.js
  - Next.js
  - Nest.js
  - Other TypeScript-based frameworks

#### Go Ecosystem Integration

- **Compatibility**
  - Full backward compatibility with existing Go codebases
  - Native Go module support
  - Seamless integration with Go tooling

## Type system

- Types have functions that define constraints and narrow them down (types with multiple inheritance / polymorphism if you will)
  - e.g. `String.Min(3)` is a type that is constrained to be a string with a minimum length of 3
- All types are capital case or anonymous struct types
- All nominal types are aliases of constraints on other types
  - e.g. `Array(String)` is an anonymous, non-nominal type
  - but `type StringArray = Array(String)` is a nominal type
- Traits from Rust instead of classes? Allow any objects that satisfy the trait to be used in place of the trait.
  - Disallow methods in parameter object types

### Features

- Pattern matching from Rust / match statements
- Avoid curly braces in method parameters by using named parameters. Only require named parameters if the method has more than 1 parameter. This avoids the need to add curly braces when you want to add another parameter. Instead, you just name the existing parameter, add one or more other parameters and require the calling site to now name all parameters.
- Error handling patterns that follow Go's idiomatic approach with explicit error returns, while providing enhanced tooling for common patterns like error wrapping (`errors.Wrap`, `fmt.Errorf` with `%w`), structured logging and stack trace integration (e.g. `slogerr` with error context), and error type hierarchies - the language should make it easy to define custom error types with additional context fields, support error wrapping and unwrapping for error chain inspection, and integrate with logging frameworks to capture error details and stack traces

### Anti-features

- Don't fall through in switch statements.

### Use Cases (to keep an eye on)

- Make function signatures GraphQL compatible as well
