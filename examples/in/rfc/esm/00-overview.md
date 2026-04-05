# Forst as Native ES Modules

## Overview

This RFC explores how Forst could compile to native ES modules that run directly in Node.js/Bun, providing seamless integration with the JavaScript/TypeScript ecosystem while leveraging Go's performance and type safety.

## Core Concept

**Forst → Native ES Modules → Direct Node.js/Bun Integration**

Instead of sidecar patterns or external runtimes, Forst compiles directly to native ES modules that can be imported and used like any other JavaScript/TypeScript module.

## Key Benefits

### 1. Zero Complexity Integration

- **No sidecar processes** - runs in same process as TypeScript
- **No IPC overhead** - direct function calls
- **No external dependencies** - just npm install and import

### 2. Native Performance

- **Direct function calls** - no serialization/deserialization
- **Go performance** - compiled to optimized Go code
- **Memory efficiency** - shared memory space

### 3. Ecosystem Compatibility

- **Full npm ecosystem** - works with any JavaScript library
- **TypeScript integration** - seamless type checking
- **Standard tooling** - works with existing build tools

### 4. Developer Experience

- **Familiar syntax** - import/export like any module
- **Standard debugging** - same tools and processes
- **Easy deployment** - single application container

## Technical Approach

### Compilation Target

- **Input**: Forst source code (`.ft` files)
- **Output**: Native ES modules (`.mjs` or `.js` files)
- **Runtime**: Node.js/Bun with Go addon

### Module System

- **ES Module format** - standard import/export
- **TypeScript declarations** - generated `.d.ts` files
- **Go addon integration** - native Go code via Node.js addons

### Type Safety

- **Compile-time validation** - Forst's type system
- **Runtime type checking** - Go's type safety
- **TypeScript integration** - generated type definitions

## Implementation Strategy

### Phase 1: Basic ES Module Support

- Compile Forst to Go addon
- Generate ES module wrapper
- Basic TypeScript declarations

### Phase 2: Advanced Integration

- Full type system integration
- Error handling and validation
- Performance optimizations

### Phase 3: Ecosystem Integration

- npm package publishing
- Build tool integration
- Development tooling

## Success Criteria

1. **Zero configuration** - works out of the box
2. **Native performance** - Go performance in Node.js
3. **Full compatibility** - works with existing TypeScript code
4. **Easy adoption** - familiar developer experience

## Conclusion

Forst as native ES modules provides the perfect balance of:

- **Simplicity** - no complex integration patterns
- **Performance** - Go's native performance
- **Compatibility** - full JavaScript ecosystem support
- **Developer experience** - familiar and intuitive

This approach positions Forst as a natural evolution of the JavaScript ecosystem, providing Go's performance and type safety while maintaining the simplicity and familiarity of ES modules.
