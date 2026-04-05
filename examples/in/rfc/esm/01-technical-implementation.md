# Technical Implementation: Forst as Native ES Modules

## Overview

This document details the technical implementation of compiling Forst to native ES modules that run directly in Node.js/Bun.

## Architecture

### Compilation Pipeline

```
Forst Source (.ft) → Go Code → Go Addon → ES Module Wrapper → TypeScript Declarations
```

### Components

1. **Forst Compiler** - compiles `.ft` to Go
2. **Go Addon Generator** - creates Node.js addon from Go code
3. **ES Module Wrapper** - JavaScript wrapper for Go addon
4. **TypeScript Generator** - generates `.d.ts` files

## Implementation Details

### 1. Go Addon Generation

#### Go Code Structure

```go
// Generated from Forst source
package main

import "C"
import "github.com/forst/forst/runtime"

//export processUser
func processUser(id int64) (string, error) {
    // Forst compiled code
    user, err := getUserById(id)
    if err != nil {
        return "", err
    }

    validated, err := validateUser(user)
    if err != nil {
        return "", err
    }

    return validated, nil
}

func main() {}
```

#### Node.js Addon Binding

```go
// addon.go
package main

import "C"
import "github.com/forst/forst/runtime"

//export init
func init() {
    runtime.RegisterFunction("processUser", processUser)
}

//export processUser
func processUser(id int64) (string, error) {
    // Implementation
}
```

### 2. ES Module Wrapper

#### JavaScript Wrapper

```javascript
// Generated ES module wrapper
import { createRequire } from "module";
import { fileURLToPath } from "url";
import path from "path";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// Load Go addon
const addon = require(path.join(__dirname, "addon.node"));

// Export functions
export const processUser = (id) => {
  try {
    const result = addon.processUser(id);
    return { success: true, data: result };
  } catch (error) {
    return { success: false, error: error.message };
  }
};

// Export types
export const types = {
  processUser: {
    input: { id: "number" },
    output: { success: "boolean", data: "string", error: "string" },
  },
};
```

#### TypeScript Declarations

```typescript
// Generated TypeScript declarations
export interface ProcessUserResult {
  success: boolean;
  data?: string;
  error?: string;
}

export function processUser(id: number): ProcessUserResult;

export const types: {
  processUser: {
    input: { id: "number" };
    output: { success: "boolean"; data: "string"; error: "string" };
  };
};
```

### 3. Build System Integration

#### Package.json

```json
{
  "name": "@forst/example",
  "version": "1.0.0",
  "type": "module",
  "main": "./dist/index.js",
  "types": "./dist/index.d.ts",
  "files": ["dist/", "addon.node"],
  "scripts": {
    "build": "forst compile --target=esm",
    "dev": "forst compile --target=esm --watch"
  },
  "dependencies": {
    "@forst/runtime": "^1.0.0"
  }
}
```

#### Build Configuration

```yaml
# forst.config.yaml
target: esm
output: dist/
addon: addon.node
types: dist/index.d.ts
wrapper: dist/index.js
```

### 4. Runtime Integration

#### Forst Runtime

```go
// runtime/runtime.go
package runtime

import (
    "C"
    "encoding/json"
    "unsafe"
)

type FunctionRegistry struct {
    functions map[string]interface{}
}

var registry = &FunctionRegistry{
    functions: make(map[string]interface{}),
}

func RegisterFunction(name string, fn interface{}) {
    registry.functions[name] = fn
}

func CallFunction(name string, args ...interface{}) (interface{}, error) {
    fn, exists := registry.functions[name]
    if !exists {
        return nil, fmt.Errorf("function %s not found", name)
    }

    // Call function with args
    return callFunction(fn, args...)
}

func callFunction(fn interface{}, args ...interface{}) (interface{}, error) {
    // Type-safe function calling
    switch f := fn.(type) {
    case func(int64) (string, error):
        if len(args) != 1 {
            return nil, fmt.Errorf("expected 1 argument, got %d", len(args))
        }
        id, ok := args[0].(int64)
        if !ok {
            return nil, fmt.Errorf("expected int64, got %T", args[0])
        }
        return f(id)
    default:
        return nil, fmt.Errorf("unsupported function type")
    }
}
```

#### Node.js Integration

```javascript
// runtime/node.js
import { createRequire } from "module";

const require = createRequire(import.meta.url);
const addon = require("./addon.node");

export class ForstRuntime {
  constructor() {
    this.addon = addon;
  }

  async callFunction(name, ...args) {
    try {
      const result = this.addon.callFunction(name, ...args);
      return { success: true, data: result };
    } catch (error) {
      return { success: false, error: error.message };
    }
  }

  registerFunction(name, fn) {
    this.addon.registerFunction(name, fn);
  }
}
```

### 5. Type System Integration

#### Forst Type Mapping

```go
// types/mapping.go
package types

import "github.com/forst/forst/ast"

type TypeMapping struct {
    ForstType ast.TypeNode
    GoType    string
    JSType    string
    TSType    string
}

var typeMappings = map[string]TypeMapping{
    "Int": {
        GoType: "int64",
        JSType: "number",
        TSType: "number",
    },
    "String": {
        GoType: "string",
        JSType: "string",
        TSType: "string",
    },
    "Bool": {
        GoType: "bool",
        JSType: "boolean",
        TSType: "boolean",
    },
    "List": {
        GoType: "[]interface{}",
        JSType: "Array",
        TSType: "Array<T>",
    },
    "Map": {
        GoType: "map[string]interface{}",
        JSType: "Object",
        TSType: "Record<string, T>",
    },
}
```

#### Type Generation

```go
// generator/types.go
package generator

import (
    "github.com/forst/forst/ast"
    "github.com/forst/forst/types"
)

func GenerateTypeScriptTypes(node ast.Node) string {
    var tsTypes strings.Builder

    switch n := node.(type) {
    case *ast.FunctionNode:
        tsTypes.WriteString(generateFunctionType(n))
    case *ast.StructNode:
        tsTypes.WriteString(generateStructType(n))
    case *ast.TypeDefNode:
        tsTypes.WriteString(generateTypeDef(n))
    }

    return tsTypes.String()
}

func generateFunctionType(fn *ast.FunctionNode) string {
    var ts strings.Builder

    ts.WriteString("export function ")
    ts.WriteString(fn.Name)
    ts.WriteString("(")

    // Generate parameters
    for i, param := range fn.Parameters {
        if i > 0 {
            ts.WriteString(", ")
        }
        ts.WriteString(param.Name)
        ts.WriteString(": ")
        ts.WriteString(mapForstTypeToTS(param.Type))
    }

    ts.WriteString("): ")
    ts.WriteString(mapForstTypeToTS(fn.ReturnType))
    ts.WriteString(";")

    return ts.String()
}
```

### 6. Error Handling

#### Go Error Handling

```go
// errors/handling.go
package errors

import (
    "encoding/json"
    "fmt"
)

type ForstError struct {
    Type    string `json:"type"`
    Message string `json:"message"`
    Code    int    `json:"code"`
    Details map[string]interface{} `json:"details"`
}

func (e *ForstError) Error() string {
    return e.Message
}

func (e *ForstError) ToJSON() string {
    data, _ := json.Marshal(e)
    return string(data)
}

func NewForstError(errorType, message string, code int) *ForstError {
    return &ForstError{
        Type:    errorType,
        Message: message,
        Code:    code,
        Details: make(map[string]interface{}),
    }
}
```

#### JavaScript Error Handling

```javascript
// errors/handling.js
export class ForstError extends Error {
  constructor(type, message, code, details = {}) {
    super(message);
    this.name = "ForstError";
    this.type = type;
    this.code = code;
    this.details = details;
  }

  static fromJSON(json) {
    const data = JSON.parse(json);
    return new ForstError(data.type, data.message, data.code, data.details);
  }

  toJSON() {
    return {
      type: this.type,
      message: this.message,
      code: this.code,
      details: this.details,
    };
  }
}
```

### 7. Performance Optimizations

#### Memory Management

```go
// performance/memory.go
package performance

import (
    "runtime"
    "unsafe"
)

type MemoryPool struct {
    pools map[string]*sync.Pool
}

func NewMemoryPool() *MemoryPool {
    return &MemoryPool{
        pools: make(map[string]*sync.Pool),
    }
}

func (mp *MemoryPool) Get(size int) []byte {
    pool, exists := mp.pools[size]
    if !exists {
        pool = &sync.Pool{
            New: func() interface{} {
                return make([]byte, size)
            },
        }
        mp.pools[size] = pool
    }

    return pool.Get().([]byte)
}

func (mp *MemoryPool) Put(data []byte) {
    pool, exists := mp.pools[len(data)]
    if exists {
        pool.Put(data)
    }
}
```

#### Function Caching

```go
// performance/cache.go
package performance

import (
    "sync"
    "time"
)

type FunctionCache struct {
    cache map[string]CacheEntry
    mutex sync.RWMutex
    ttl   time.Duration
}

type CacheEntry struct {
    Value     interface{}
    ExpiresAt time.Time
}

func NewFunctionCache(ttl time.Duration) *FunctionCache {
    return &FunctionCache{
        cache: make(map[string]CacheEntry),
        ttl:   ttl,
    }
}

func (fc *FunctionCache) Get(key string) (interface{}, bool) {
    fc.mutex.RLock()
    defer fc.mutex.RUnlock()

    entry, exists := fc.cache[key]
    if !exists || time.Now().After(entry.ExpiresAt) {
        return nil, false
    }

    return entry.Value, true
}

func (fc *FunctionCache) Set(key string, value interface{}) {
    fc.mutex.Lock()
    defer fc.mutex.Unlock()

    fc.cache[key] = CacheEntry{
        Value:     value,
        ExpiresAt: time.Now().Add(fc.ttl),
    }
}
```

## Build Process

### 1. Compilation Steps

```bash
# 1. Compile Forst to Go
forst compile --target=go --output=src/

# 2. Generate Go addon
go build -buildmode=c-shared -o addon.node src/main.go

# 3. Generate ES module wrapper
forst generate --target=esm --output=dist/

# 4. Generate TypeScript declarations
forst generate --target=typescript --output=dist/
```

### 2. Development Workflow

```bash
# Watch mode for development
forst compile --target=esm --watch

# Build for production
forst compile --target=esm --optimize

# Test the module
node --input-type=module dist/index.js
```

### 3. Package Publishing

```bash
# Build package
npm run build

# Publish to npm
npm publish

# Install in project
npm install @forst/example
```

## Usage Example

### Forst Source

```forst
// example.ft
func processUser(id: Int) String {
    user := getUserById(id)
    ensure user != nil, "User not found"

    validated := validateUser(user)
    ensure validated, "User validation failed"

    return validated
}
```

### Generated ES Module

```javascript
// dist/index.js
import { createRequire } from "module";
import { fileURLToPath } from "url";
import path from "path";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const addon = require(path.join(__dirname, "addon.node"));

export const processUser = (id) => {
  try {
    const result = addon.processUser(id);
    return { success: true, data: result };
  } catch (error) {
    return { success: false, error: error.message };
  }
};
```

### TypeScript Usage

```typescript
// app.ts
import { processUser } from "@forst/example";

const result = processUser(123);
if (result.success) {
  console.log("User:", result.data);
} else {
  console.error("Error:", result.error);
}
```

## Conclusion

This technical implementation provides:

1. **Native ES module support** - direct Node.js/Bun integration
2. **Go performance** - compiled to optimized Go code
3. **Type safety** - full type system integration
4. **Zero complexity** - works like any other npm package
5. **Full ecosystem compatibility** - works with existing JavaScript/TypeScript code

The result is a seamless integration that provides Go's performance and type safety while maintaining the simplicity and familiarity of ES modules.
