# @forst/client

A type-safe, Prisma-like client for Forst that provides seamless integration with your Forst functions.

## Features

- **Prisma-like Syntax**: Use `forst.package.function()` syntax for intuitive function calls
- **Type Safety**: Full TypeScript support with generated types
- **Auto-completion**: IntelliSense support for all your Forst functions
- **Error Handling**: Comprehensive error handling with detailed error messages
- **Configuration**: Flexible configuration options for different environments

## Installation

```bash
npm install @forst/client
```

## Quick Start

### 1. Generate Your Client

First, generate the TypeScript client from your Forst files:

```bash
forst generate ./your-forst-files
```

This will create:

- TypeScript declaration files (`.d.ts`)
- Client implementation files (`.client.ts`)
- A complete client package structure

### 2. Use the Client

```typescript
import { createForstClient } from "@forst/client";

// Create a client instance
const forst = createForstClient({
  baseUrl: "http://localhost:8080",
  timeout: 30000,
});

// Use Prisma-like syntax
const result = await forst.user.CreateUser({
  name: "Alice Johnson",
  age: 28,
  email: "alice@example.com",
});

console.log("User created:", result);
```

## API Reference

### `createForstClient(config?)`

Creates a new Forst client instance with the specified configuration.

**Parameters:**

- `config` (optional): Client configuration object
  - `baseUrl`: The URL of your Forst server (default: `http://localhost:8080`)
  - `timeout`: Request timeout in milliseconds (default: `30000`)
  - `retries`: Number of retry attempts (default: `3`)
  - `mode`: Environment mode (default: `'development'`)

**Returns:** A client instance with dynamic package access

### Dynamic Package Access

The client supports dynamic package access, allowing you to call functions using the package name as a property:

```typescript
// Access package functions dynamically
const user = await forst.user.GetUserById("123");
const echo = await forst.echo.Echo({ message: "Hello!" });
const math = await forst.math.Add(5, 3);
```

## Examples

### Basic Usage

```typescript
import { createForstClient } from "@forst/client";

const forst = createForstClient();

async function main() {
  try {
    // Create a user
    const createResult = await forst.user.CreateUser({
      name: "Alice Johnson",
      age: 28,
      email: "alice@example.com",
    });
    console.log("User created:", createResult);

    // Get user by ID
    const user = await forst.user.GetUserById("123");
    console.log("User found:", user);

    // Update user age
    const updatedUser = await forst.user.UpdateUserAge("123", 29);
    console.log("User updated:", updatedUser);
  } catch (error) {
    console.error("Error:", error);
  }
}

main();
```

### Error Handling

```typescript
const forst = createForstClient();

try {
  const result = await forst.user.CreateUser({
    name: "", // Invalid: empty name
    age: 200, // Invalid: age too high
    email: "invalid-email", // Invalid: no @ symbol
  });
} catch (error) {
  console.log("Validation error:", error.message);
  // Output: "Validation error: user.CreateUser failed"
}
```

### Multiple Packages

```typescript
const forst = createForstClient();

// Call functions from different packages
const userResult = await forst.user.GetUserById("123");
const echoResult = await forst.echo.Echo({ message: "Hello!" });
const mathResult = await forst.math.Add(5, 3);
```

### Configuration

```typescript
// Development configuration
const devClient = createForstClient({
  baseUrl: "http://localhost:8080",
  timeout: 30000,
  mode: "development",
});

// Production configuration
const prodClient = createForstClient({
  baseUrl: "https://api.yourdomain.com",
  timeout: 60000,
  retries: 5,
  mode: "production",
});
```

## TypeScript Support

The client provides full TypeScript support with generated types:

```typescript
import { createForstClient } from "@forst/client";

const forst = createForstClient();

// TypeScript knows the exact types
const user = await forst.user.GetUserById("123");
console.log(user.name); // TypeScript knows this is a string
console.log(user.age); // TypeScript knows this is a number

// This would cause a TypeScript error:
// console.log(user.invalidField); // Property 'invalidField' does not exist
```

## Generated Files Structure

When you run `forst generate`, it creates:

```
your-project/
├── generated/
│   ├── user.d.ts          # TypeScript declarations
│   ├── user.client.ts     # Client implementation
│   ├── echo.d.ts
│   └── echo.client.ts
└── client/
    ├── index.ts           # Main client entry point
    └── package.json       # Client package configuration
```

## Environment Variables

The client respects these environment variables:

- `FORST_BASE_URL`: Default server URL (defaults to `http://localhost:8080`)

## Error Handling

The client provides comprehensive error handling:

- **Network Errors**: Automatic retry with exponential backoff
- **Validation Errors**: Detailed error messages from Forst validation
- **Function Errors**: Clear error messages with function context
- **Type Errors**: TypeScript compilation errors for type mismatches

## Migration from Sidecar

If you're currently using `@forst/sidecar` directly, migration is straightforward:

**Before (Sidecar):**

```typescript
import { ForstClient } from "@forst/sidecar";

const client = new ForstClient({ baseUrl: "http://localhost:8080" });
const result = await client.invokeFunction("user", "CreateUser", input);
```

**After (Client):**

```typescript
import { createForstClient } from "@forst/client";

const forst = createForstClient({ baseUrl: "http://localhost:8080" });
const result = await forst.user.CreateUser(input);
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

MIT
