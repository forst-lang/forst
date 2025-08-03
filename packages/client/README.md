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
import { ForstClient } from "@forst/client";

// Create a client instance
const forst = new ForstClient({
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

### `new ForstClient(config?)`

Creates a new Forst client instance with the specified configuration.

**Parameters:**

- `config` (optional): Client configuration object
  - `baseUrl`: The URL of your Forst server (default: `http://localhost:8080`)
  - `timeout`: Request timeout in milliseconds (default: `30000`)
  - `retries`: Number of retry attempts, e.g. during function invocation (default: `3`)
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
import { ForstClient } from "@forst/client";

const forst = new ForstClient();

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
const forst = new ForstClient();

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
const forst = new ForstClient();

// Call functions from different packages
const userResult = await forst.user.GetUserById("123");
const echoResult = await forst.echo.Echo({ message: "Hello!" });
const mathResult = await forst.math.Add(5, 3);
```

## TypeScript Support

The client provides full TypeScript support with generated types:

```typescript
import { ForstClient } from "@forst/client";

const forst = new ForstClient();

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
