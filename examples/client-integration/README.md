# Forst Client Integration Example

This example demonstrates how to transform the current sidecar approach into a seamless Node.js integration similar to Prisma.

## Current Approach vs Target Approach

### Current Approach (Manual Sidecar Setup)

```typescript
// Manual setup required
import { createSidecar } from "@forst/sidecar";

const sidecar = createSidecar({
  mode: "development",
  port: 8080,
  host: "localhost",
});

await sidecar.start();

// Manual function discovery and invocation
const functions = await sidecar.discoverFunctions();
const result = await sidecar.invoke("echo", "Echo", { message: "Hello" });

await sidecar.stop();
```

### Target Approach (Seamless Integration)

```typescript
// Simple import - no setup required
import { Echo, GetUserAge } from "@forst/client";

// Direct function calls with full type safety
const result = await Echo({ message: "Hello, Forst!" });
const age = await GetUserAge();
```

## Implementation Plan

### Phase 1: TypeScript Transformer

**1.1 Create TypeScript Transformer (`@forst/transformer`)**

The transformer converts Forst AST to TypeScript:

```go
// forst/internal/transformer/ts/transformer.go
type TypeScriptTransformer struct {
    TypeChecker *typechecker.TypeChecker
    Output      *TypeScriptOutput
}

func (t *TypeScriptTransformer) TransformForstFileToTypeScript(nodes []ast.Node) (*TypeScriptOutput, error) {
    // Convert Forst types to TypeScript interfaces
    // Convert Forst functions to TypeScript async functions
    // Generate client-side implementation
}
```

**1.2 Type Generation**

Forst code:

```forst
package echo

type EchoRequest = {
  message: String
}

type EchoResponse = {
  echo: String,
  timestamp: Int
}

func Echo(input EchoRequest) {
  return {
    echo: input.message,
    timestamp: 1234567890
  }
}
```

Generated TypeScript:

```typescript
export interface EchoRequest {
  message: string;
}

export interface EchoResponse {
  echo: string;
  timestamp: number;
}

export async function Echo(input: EchoRequest): Promise<EchoResponse> {
  const response = await client.invokeFunction("echo", "Echo", input);
  if (!response.success) {
    throw new Error(response.error || "Function call failed");
  }
  return response.result as EchoResponse;
}
```

### Phase 2: Client Package (`@forst/client`)

**2.1 Package Structure**

```
packages/client/
├── package.json
├── tsconfig.json
├── src/
│   └── index.ts          # Main entry point
└── dist/                 # Generated output
    ├── index.js
    ├── index.d.ts
    └── generated/        # Auto-generated from .ft files
        ├── echo.ts
        └── typesafety.ts
```

**2.2 CLI Integration**

```bash
# Install the client package
npm install @forst/client

# Generate TypeScript types and functions
forst generate ./src

# Use in your code
import { Echo, GetUserAge } from "@forst/client";
```

### Phase 3: Development Workflow

**3.1 File Structure**

```
my-project/
├── package.json
├── src/
│   ├── functions/
│   │   ├── echo.ft
│   │   └── validation.ft
│   └── index.ts
└── node_modules/
    └── @forst/
        └── client/
            └── dist/
                └── generated/
                    ├── echo.ts
                    └── validation.ts
```

**3.2 Development Process**

1. **Write Forst functions** in `.ft` files
2. **Run `forst generate`** to create TypeScript client
3. **Import and use** functions directly in TypeScript
4. **Hot reload** during development

**3.3 Example Usage**

```typescript
// src/index.ts
import { Echo, ValidateUser } from "@forst/client";

async function main() {
  // Call Forst functions with full type safety
  const echoResult = await Echo({ message: "Hello, Forst!" });
  console.log(echoResult.echo); // TypeScript knows this is a string

  const user = await ValidateUser({
    name: "John",
    age: 25,
  });
  console.log(user.id); // TypeScript knows this exists
}

main().catch(console.error);
```

### Phase 4: Advanced Features

**4.1 Type Guards**

Forst code:

```forst
package validation

type User = {
  name: String,
  age: Int
}

is (user User) Valid {
  ensure user.age is GreaterThan(0)
  ensure user.name is Min(1)
}

func ValidateUser(user User) {
  ensure user is Valid
  return { id: "123", ...user }
}
```

Generated TypeScript:

```typescript
export interface User {
  name: string;
  age: number;
}

export function isValidUser(user: any): user is User {
  return (
    typeof user.age === "number" &&
    user.age > 0 &&
    typeof user.name === "string" &&
    user.name.length >= 1
  );
}

export async function ValidateUser(user: User): Promise<User & { id: string }> {
  if (!isValidUser(user)) {
    throw new Error("Invalid user data");
  }

  const response = await client.invokeFunction(
    "validation",
    "ValidateUser",
    user
  );
  if (!response.success) {
    throw new Error(response.error || "Validation failed");
  }
  return response.result;
}
```

**4.2 Streaming Support**

```typescript
import { ProcessLargeDataset } from "@forst/client";

// Streaming for large datasets
await ProcessLargeDataset({ records: largeDataset }, (chunk) => {
  console.log("Processed chunk:", chunk);
});
```

## Benefits

### 1. **Developer Experience**

- **Zero configuration** for common use cases
- **Full TypeScript support** with auto-generated types
- **IntelliSense** and autocomplete
- **Hot reloading** during development

### 2. **Performance**

- **10x faster** than TypeScript for CPU-intensive operations
- **Memory efficient** with Go's garbage collector
- **Concurrent processing** with goroutines

### 3. **Gradual Adoption**

- **Replace functions one-by-one** without disrupting existing code
- **Hybrid approach** - keep TypeScript for simple operations, use Forst for performance-critical parts
- **Backward compatibility** with existing TypeScript code

### 4. **Production Ready**

- **Standard observability** with OpenTelemetry
- **Error handling** and retry logic
- **Health checks** and monitoring
- **Multiple transport options** (HTTP, IPC, Direct)

## Migration Path

### Step 1: Install and Setup

```bash
npm install @forst/client
forst generate ./src
```

### Step 2: Identify Performance Bottlenecks

```typescript
// Before: Slow TypeScript implementation
function processData(records: DataRecord[]): ProcessedData[] {
  return records.map((record) => {
    // CPU-intensive processing
    return heavyComputation(record);
  });
}
```

### Step 3: Create Forst Implementation

```forst
// functions/data_processing.ft
package dataprocessing

type DataRecord = {
  id: String,
  data: String
}

type ProcessedData = {
  id: String,
  result: String,
  timestamp: Int
}

func ProcessData(records Array(DataRecord)) {
  // High-performance Go implementation
  return records.map(record => {
    return {
      id: record.id,
      result: heavyComputation(record.data),
      timestamp: 1234567890
    }
  })
}
```

### Step 4: Update TypeScript Code

```typescript
// After: Fast Forst implementation
import { ProcessData } from "@forst/client";

async function handleRequest(records: DataRecord[]) {
  // This now runs in high-performance Go binary
  const processed = await ProcessData(records);
  return processed;
}
```

## Configuration

### Environment Variables

```bash
# Development
FORST_BASE_URL=http://localhost:8080
FORST_LOG_LEVEL=debug

# Production
FORST_BASE_URL=https://api.example.com
FORST_LOG_LEVEL=info
```

### Package.json Scripts

```json
{
  "scripts": {
    "dev": "forst dev & npm run start",
    "generate": "forst generate ./src",
    "build": "forst generate ./src && tsc",
    "start": "node dist/index.js"
  }
}
```

This approach provides a seamless developer experience while maintaining the performance benefits of the sidecar architecture.
