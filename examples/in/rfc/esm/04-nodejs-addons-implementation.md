# Node.js Addons Implementation for Forst

## Overview

This document explores the simplest and most effective approaches to implement Node.js addons for Forst, focusing on Node.js first and foremost while considering Forst's specific requirements.

## Research Findings

Based on current best practices and available tools, the most straightforward approach is to use **Node-API (N-API)** with Go bindings, specifically the `napi-go` library.

## Recommended Approach: napi-go

### Why napi-go?

1. **Node-API Compatibility** - Works across all Node.js versions (8+)
2. **Go Native** - Direct Go implementation without C++ wrapper
3. **Active Maintenance** - Well-maintained library with good documentation
4. **Simple Build Process** - Minimal configuration required
5. **Performance** - Direct Go-to-Node.js communication

### Implementation Strategy

#### 1. Basic Setup

```go
// go.mod
module forst-addon

go 1.21

require (
    github.com/akshayganeshen/napi-go v0.1.0
)
```

#### 2. Forst Function Wrapper

```go
// handlers/forst.go
package handlers

import (
    "encoding/json"
    "github.com/akshayganeshen/napi-go"
)

// ForstFunction represents a compiled Forst function
type ForstFunction struct {
    Name        string
    Parameters  []Parameter
    ReturnType  string
    Handler     func(args []interface{}) (interface{}, error)
}

type Parameter struct {
    Name string
    Type string
}

// ProcessUser - Example Forst function
func ProcessUser(env napi.Env, info napi.CallbackInfo) napi.Value {
    // Get arguments
    args, err := napi.GetCallbackArgs(env, info)
    if err != nil {
        return napi.CreateError(env, "Failed to get arguments: "+err.Error())
    }

    if len(args) != 1 {
        return napi.CreateError(env, "Expected 1 argument, got "+string(len(args)))
    }

    // Extract ID from first argument
    id, err := napi.GetValueAsInt64(env, args[0])
    if err != nil {
        return napi.CreateError(env, "Invalid ID: "+err.Error())
    }

    // Call Forst function (this would be generated)
    result, err := processUserForst(int(id))
    if err != nil {
        return napi.CreateError(env, err.Error())
    }

    // Convert result to JavaScript object
    return napi.CreateObject(env, map[string]interface{}{
        "success": true,
        "data":    result,
    })
}

// processUserForst - Generated from Forst source
func processUserForst(id int) (map[string]interface{}, error) {
    // This would be generated from Forst code
    // For now, return mock data
    return map[string]interface{}{
        "id":    id,
        "name":  "John Doe",
        "email": "john@example.com",
    }, nil
}
```

#### 3. Module Registration

```go
// main.go
package main

import (
    "forst-addon/handlers"
    "github.com/akshayganeshen/napi-go/entry"
)

func init() {
    // Register Forst functions
    entry.Export("processUser", handlers.ProcessUser)
    entry.Export("validateUser", handlers.ValidateUser)
    entry.Export("createUser", handlers.CreateUser)
}

func main() {}
```

#### 4. Build Configuration

```bash
# Build script
#!/bin/bash

# Build the addon
go build -buildmode=c-shared -o addon.node .

# Verify the build
file addon.node
```

#### 5. Generated ES Module Wrapper

```javascript
// dist/index.js
import { createRequire } from "module";
import { fileURLToPath } from "url";
import path from "path";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const addon = require(path.join(__dirname, "addon.node"));

// Wrapper functions with error handling
export const processUser = (id) => {
  try {
    const result = addon.processUser(id);
    return result;
  } catch (error) {
    return {
      success: false,
      error: error.message,
    };
  }
};

export const validateUser = (user) => {
  try {
    const result = addon.validateUser(user);
    return result;
  } catch (error) {
    return {
      success: false,
      error: error.message,
    };
  }
};

export const createUser = (user) => {
  try {
    const result = addon.createUser(user);
    return result;
  } catch (error) {
    return {
      success: false,
      error: error.message,
    };
  }
};
```

## Forst Integration Pipeline

### 1. Forst Compilation to Go

```go
// Generated from Forst source
// example.ft -> example.go

package main

import (
    "database/sql"
    "encoding/json"
    "errors"
    "fmt"
)

// User represents a user in the system
type User struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

// DatabaseError represents a database error
type DatabaseError struct {
    Message string `json:"message"`
}

func (e DatabaseError) Error() string {
    return e.Message
}

// ValidationError represents a validation error
type ValidationError struct {
    Field   string `json:"field"`
    Message string `json:"message"`
}

func (e ValidationError) Error() string {
    return e.Message
}

// processUser - Generated from Forst
func processUser(id int) (map[string]interface{}, error) {
    // Database query
    user, err := getUserById(id)
    if err != nil {
        return nil, DatabaseError{Message: err.Error()}
    }

    // Validation
    if user.Name == "" {
        return nil, ValidationError{Field: "name", Message: "Name is required"}
    }

    if user.Email == "" {
        return nil, ValidationError{Field: "email", Message: "Email is required"}
    }

    return map[string]interface{}{
        "id":    user.ID,
        "name":  user.Name,
        "email": user.Email,
    }, nil
}

func getUserById(id int) (*User, error) {
    // Mock database query
    if id <= 0 {
        return nil, errors.New("invalid user ID")
    }

    return &User{
        ID:    id,
        Name:  "John Doe",
        Email: "john@example.com",
    }, nil
}
```

### 2. napi-go Handler Generation

```go
// Generated napi-go handlers
package handlers

import (
    "github.com/akshayganeshen/napi-go"
)

func ProcessUser(env napi.Env, info napi.CallbackInfo) napi.Value {
    args, err := napi.GetCallbackArgs(env, info)
    if err != nil {
        return napi.CreateError(env, "Failed to get arguments: "+err.Error())
    }

    if len(args) != 1 {
        return napi.CreateError(env, "Expected 1 argument")
    }

    id, err := napi.GetValueAsInt64(env, args[0])
    if err != nil {
        return napi.CreateError(env, "Invalid ID: "+err.Error())
    }

    result, err := processUser(int(id))
    if err != nil {
        return napi.CreateObject(env, map[string]interface{}{
            "success": false,
            "error":   err.Error(),
        })
    }

    return napi.CreateObject(env, map[string]interface{}{
        "success": true,
        "data":    result,
    })
}
```

### 3. TypeScript Declaration Generation

```typescript
// Generated TypeScript declarations
export interface User {
  id: number;
  name: string;
  email: string;
}

export interface ProcessUserResult {
  success: boolean;
  data?: User;
  error?: string;
}

export interface ValidationError {
  field: string;
  message: string;
}

export function processUser(id: number): ProcessUserResult;
export function validateUser(user: User): ProcessUserResult;
export function createUser(user: Omit<User, "id">): ProcessUserResult;
```

## Build Process Automation

### 1. Forst Compiler Integration

```go
// compiler/addon.go
package compiler

import (
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
)

type AddonConfig struct {
    SourceDir    string
    OutputDir    string
    AddonName    string
    Functions    []FunctionConfig
}

type FunctionConfig struct {
    Name       string
    Parameters []Parameter
    ReturnType string
}

func BuildAddon(config AddonConfig) error {
    // 1. Compile Forst to Go
    if err := compileForstToGo(config); err != nil {
        return fmt.Errorf("failed to compile Forst: %w", err)
    }

    // 2. Generate napi-go handlers
    if err := generateNapiHandlers(config); err != nil {
        return fmt.Errorf("failed to generate handlers: %w", err)
    }

    // 3. Build the addon
    if err := buildAddon(config); err != nil {
        return fmt.Errorf("failed to build addon: %w", err)
    }

    // 4. Generate ES module wrapper
    if err := generateESModule(config); err != nil {
        return fmt.Errorf("failed to generate ES module: %w", err)
    }

    // 5. Generate TypeScript declarations
    if err := generateTypeScriptDeclarations(config); err != nil {
        return fmt.Errorf("failed to generate TypeScript: %w", err)
    }

    return nil
}

func buildAddon(config AddonConfig) error {
    cmd := exec.Command("go", "build", "-buildmode=c-shared", "-o", filepath.Join(config.OutputDir, config.AddonName+".node"), ".")
    cmd.Dir = config.SourceDir
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr

    return cmd.Run()
}
```

### 2. CLI Integration

```bash
# Forst CLI command
forst compile --target=nodejs-addon --output=dist/ --addon-name=example

# This would:
# 1. Compile Forst source to Go
# 2. Generate napi-go handlers
# 3. Build the .node addon
# 4. Generate ES module wrapper
# 5. Generate TypeScript declarations
```

## Performance Considerations

### 1. Memory Management

```go
// handlers/memory.go
package handlers

import (
    "runtime"
    "sync"
    "unsafe"
)

type MemoryPool struct {
    pools map[int]*sync.Pool
    mutex sync.RWMutex
}

var globalPool = &MemoryPool{
    pools: make(map[int]*sync.Pool),
}

func (mp *MemoryPool) Get(size int) []byte {
    mp.mutex.RLock()
    pool, exists := mp.pools[size]
    mp.mutex.RUnlock()

    if !exists {
        mp.mutex.Lock()
        pool = &sync.Pool{
            New: func() interface{} {
                return make([]byte, size)
            },
        }
        mp.pools[size] = pool
        mp.mutex.Unlock()
    }

    return pool.Get().([]byte)
}

func (mp *MemoryPool) Put(data []byte) {
    pool, exists := mp.pools[len(data)]
    if exists {
        pool.Put(data)
    }
}

// Use in handlers to reduce GC pressure
func ProcessUserWithPool(env napi.Env, info napi.CallbackInfo) napi.Value {
    // Use memory pool for temporary allocations
    buffer := globalPool.Get(1024)
    defer globalPool.Put(buffer)

    // ... handler logic
}
```

### 2. Function Caching

```go
// handlers/cache.go
package handlers

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

var functionCache = &FunctionCache{
    cache: make(map[string]CacheEntry),
    ttl:   5 * time.Minute,
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

## Error Handling

### 1. Forst Error Types

```go
// errors/forst.go
package errors

import (
    "encoding/json"
    "fmt"
)

type ForstError struct {
    Type    string                 `json:"type"`
    Message string                 `json:"message"`
    Code    int                    `json:"code"`
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

// Predefined error types
var (
    DatabaseError    = "DatabaseError"
    ValidationError  = "ValidationError"
    NetworkError     = "NetworkError"
    InternalError    = "InternalError"
)
```

### 2. napi-go Error Handling

```go
// handlers/errors.go
package handlers

import (
    "encoding/json"
    "github.com/akshayganeshen/napi-go"
    "forst-addon/errors"
)

func handleForstError(env napi.Env, err error) napi.Value {
    if forstErr, ok := err.(*errors.ForstError); ok {
        return napi.CreateObject(env, map[string]interface{}{
            "success": false,
            "error":   forstErr.ToJSON(),
            "type":    forstErr.Type,
            "code":    forstErr.Code,
        })
    }

    return napi.CreateObject(env, map[string]interface{}{
        "success": false,
        "error":   err.Error(),
        "type":    "UnknownError",
        "code":    500,
    })
}
```

## Testing Strategy

### 1. Unit Tests

```go
// handlers/forst_test.go
package handlers

import (
    "testing"
    "github.com/akshayganeshen/napi-go"
)

func TestProcessUser(t *testing.T) {
    // Mock napi.Env and napi.CallbackInfo
    env := createMockEnv()
    info := createMockCallbackInfo([]interface{}{123})

    result := ProcessUser(env, info)

    // Verify result
    if !napi.IsObject(env, result) {
        t.Error("Expected object result")
    }
}
```

### 2. Integration Tests

```javascript
// test/integration.test.js
import { processUser, validateUser } from "../dist/index.js";

describe("Forst Addon Integration", () => {
  test("processUser should work correctly", async () => {
    const result = await processUser(123);

    expect(result.success).toBe(true);
    expect(result.data).toHaveProperty("id");
    expect(result.data).toHaveProperty("name");
    expect(result.data).toHaveProperty("email");
  });

  test("processUser should handle errors", async () => {
    const result = await processUser(-1);

    expect(result.success).toBe(false);
    expect(result.error).toBeDefined();
  });
});
```

## Deployment Considerations

### 1. Cross-Platform Builds

```bash
# Build for different platforms
GOOS=linux GOARCH=amd64 go build -buildmode=c-shared -o addon-linux.node .
GOOS=darwin GOARCH=amd64 go build -buildmode=c-shared -o addon-darwin.node .
GOOS=windows GOARCH=amd64 go build -buildmode=c-shared -o addon-windows.node .
```

### 2. Package.json Configuration

```json
{
  "name": "@forst/example",
  "version": "1.0.0",
  "type": "module",
  "main": "./dist/index.js",
  "types": "./dist/index.d.ts",
  "files": ["dist/", "addon-*.node"],
  "scripts": {
    "build": "forst compile --target=nodejs-addon",
    "test": "node --test test/*.test.js"
  },
  "engines": {
    "node": ">=18.0.0"
  }
}
```

## Conclusion

The **napi-go** approach provides the simplest and most effective way to implement Node.js addons for Forst:

### Benefits

1. **Simplicity** - Direct Go implementation without C++ wrapper
2. **Performance** - Native Go performance with minimal overhead
3. **Compatibility** - Works across all Node.js versions
4. **Maintainability** - Single language (Go) for both Forst and addon
5. **Type Safety** - Full type safety from Forst to TypeScript

### Implementation Steps

1. **Use napi-go** for Node-API bindings
2. **Generate handlers** from Forst functions
3. **Build with go build -buildmode=c-shared**
4. **Create ES module wrapper** for easy consumption
5. **Generate TypeScript declarations** for type safety

This approach positions Forst as a powerful tool for creating high-performance Node.js modules while maintaining the simplicity and familiarity of ES modules.
