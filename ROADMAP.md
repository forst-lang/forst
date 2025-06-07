# Forst Roadmap

For our guiding principles including anti-features see [PHILOSOPHY.md](./PHILOSOPHY.md).

## 1. Core Features

### Type System

- [x] Basic type system
- [x] Shape-based types
- [x] Type definitions
- [x] `ensure` statements
  - [x] Basic type assertions
  - [ ] `is` operator for conditions
- [ ] Type guards
  - [x] Shape guards for struct refinement
- [ ] Immutability guarantees
  - [ ] For `ensure` based scopes
  - [ ] Unsafe mode for Go interop
- [ ] Binary type expressions
- [ ] Type aliases
- [ ] Generic types

### Go Interop

- [x] Basic Go code generation
- [x] Generate validation from type constraints
- [ ] Parse imported Go code
- [ ] Check argument types

### TypeScript Interop

- [x] Basic TS type generation
- [ ] Publish NPM package for compiler
- [ ] Allow running Forst code from NodeJS

## 2. Organization

### Development

- [ ] Improve error messages with line numbers and suggestions
- [ ] Add more real-world examples
- [ ] Create basic VS Code extension

### Community

- [ ] Write "Getting Started" guide
- [ ] Add example project showing key features
- [ ] Create contributing guide with development setup

### Infrastructure

- [x] Set up CI with tests and linting
- [ ] Add code coverage reporting
- [ ] Create release automation script
- [ ] Add validation code coverage reporting
