# Forst Roadmap

## 1. Design Philosophy

### Core Mission

- Replace TypeScript as the go-to language for type-safe backend development
- Take inspiration from TypeScript's best features while maintaining our own identity
- Focus on practical features that solve real development problems

### Language Design

- Embrace structural typing - where types are defined by their shape and structure
- Enable robust type checking and validation at both compile and runtime
- Provide clear, actionable development feedback through type information
- Focus on predictable, deterministic behavior
- Maintain strong type safety without excessive verbosity
- Keep the language spec concise and unambiguous

### Go Integration

- Generate clean, idiomatic Go code that's easy to understand and maintain
- Support direct use of Go packages and types with minimal overhead
- Work seamlessly with existing Go tools, testing and deployment

### Frontend Integration

- Generate accurate TypeScript definitions from backend types
- Ensure type safety across API boundaries
- Minimize friction in full-stack development workflows

## 2. Major Ideas

- Shape-based types for structural validation
- Type constraints for runtime safety
- Type inference for better developer experience
- Seamless Go interop
- TypeScript type generation
- First-class error handling

## 3. Features

### Anti-Features (Things We Won't Do)

- No unpredictable control flow

  - While we don't forbid features like panics or goto statements, we prioritize language constructs that encourage predictable execution paths. Our goal is to have the compiler and type system guide developers toward explicit error handling (using Go's standard error return values) and structured control flow that's easy to reason about, debug, and maintain.

- No implicit type coercion

  - We reject implicit type coercion. It's a dangerous pattern that can lead to data corruption and hard-to-find bugs. Backend systems need strong type safety to prevent unintended type conversions that could compromise data integrity.

- No inheritance-based OOP

  - We choose not to support inheritance-based OOP. Experience has shown that deep inheritance hierarchies often become difficult to maintain as codebases grow. Composition provides the flexibility we need while avoiding the common pitfalls of inheritance.

- No side effects in type system

  - Side effects in the type system add unnecessary complexity that we want to avoid. They can make type checking behavior unpredictable and debugging more difficult. Our type system focuses on clear, straightforward validation of data shapes without hidden interactions between types.

### Type System

- [x] Basic type system
- [x] Shape-based types
- [x] Type definitions
- [ ] Binary type expressions
- [ ] Type constraints
- [ ] Type aliases
- [ ] Generic types

### Code Generation

- [x] Basic Go code generation
- [ ] TypeScript type generation
- [ ] Validation code generation
- [ ] Better error messages

### Language Features

- [ ] Better Go interop
- [ ] More type constraints
- [ ] Advanced type features
- [ ] Better error handling

## 4. Systems

### Development

- [ ] Improve error messages with line numbers and suggestions
- [ ] Add more real-world examples showing common use cases
- [ ] Create basic VS Code extension for syntax highlighting

### Community

- [ ] Write "Getting Started" guide
- [ ] Add example project showing key features
- [ ] Create contributing guide with development setup

### Infrastructure

- [x] Set up CI with tests and linting
- [ ] Add code coverage reporting
- [ ] Create release automation script
