# @forst/sidecar

Sidecar integration for TypeScript applications to use Forst for high-performance backend operations.

## Overview

The Forst sidecar enables gradual adoption of Forst within TypeScript applications by providing a seamless bridge between TypeScript and high-performance Forst-compiled Go binaries. This allows teams to solve performance bottlenecks incrementally without requiring a complete rewrite.

## Features

- **Zero-config setup**: Automatic detection and compilation of Forst files
- **Hot reloading**: Automatic recompilation when `.ft` files change
- **Type safety**: Automatic TypeScript interface generation
- **HTTP transport**: Familiar REST/JSON patterns for easy debugging
- **Express.js integration**: Drop-in middleware for Express applications
- **Health monitoring**: Built-in health checks and status monitoring

## Quick Start

### Installation

```bash
# Clone the repository and navigate to the sidecar package
cd packages/sidecar

# Install dependencies
npm install

# Build the package
npm run build
```

### Basic Usage

```typescript
import { autoStart } from "@forst/sidecar";

async function main() {
  // Start the sidecar with zero configuration
  const sidecar = await autoStart({
    forstDir: "./forst", // Directory containing your .ft files
    port: 8080,
  });

  // Discover available functions
  const functions = await sidecar.discoverFunctions();
  console.log("Available functions:", functions);

  // Call a Forst function
  const result = await sidecar.invoke("myPackage", "myFunction", {
    arg: "value",
  });
  console.log("Result:", result);
}

main().catch(console.error);
```

### Express.js Integration

```typescript
import express from "express";
import { ForstSidecar, createExpressMiddleware } from "@forst/sidecar";

const app = express();

// Create and start the sidecar
const sidecar = new ForstSidecar({
  forstDir: "./forst",
  port: 8080,
});
await sidecar.start();

// Add middleware to make sidecar available in routes
app.use(createExpressMiddleware(sidecar));

// Use Forst functions in your routes
app.post("/process-data", async (req, res) => {
  try {
    const result = await req.forst.invoke("data", "process", req.body);
    res.json(result);
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
});

app.listen(3000, () => {
  console.log("Server running on port 3000");
});
```

## Configuration

### Basic Configuration

```typescript
import { ForstConfig } from "@forst/sidecar";

const config: ForstConfig = {
  mode: "development", // 'development' | 'production' | 'testing'
  forstDir: "./forst", // Directory containing .ft files
  outputDir: "./dist/forst", // Output directory for compiled files
  port: 8080, // HTTP server port
  host: "localhost", // HTTP server host
  logLevel: "info", // 'debug' | 'info' | 'warn' | 'error'
  transports: {
    development: {
      mode: "http",
      http: {
        port: 8080,
        cors: true,
        healthCheck: "/health",
      },
    },
    production: {
      mode: "http",
      http: {
        port: 8080,
        cors: true,
        healthCheck: "/health",
      },
    },
  },
};
```

### Environment Variables

- `NODE_ENV`: Set to 'production' for production mode
- `FORST_DIR`: Override the Forst directory path
- `FORST_PORT`: Override the HTTP server port

## API Reference

### ForstSidecar

Main class for managing the sidecar integration.

#### Methods

- `start()`: Start the development server
- `stop()`: Stop the development server
- `discoverFunctions()`: Discover available Forst functions
- `invoke(package, function, args)`: Call a Forst function
- `invokeStreaming(package, function, args, onResult)`: Call a function with streaming
- `healthCheck()`: Check server health
- `isRunning()`: Check if server is running

### ForstClient

HTTP client for communicating with the Forst server.

#### Methods

- `discoverFunctions()`: Get list of available functions
- `invoke(package, function, args)`: Call a function
- `invokeStreaming(package, function, args, onResult)`: Stream function results
- `healthCheck()`: Check server health

### ForstServer

Manages the Forst development server process.

#### Methods

- `start()`: Start the server
- `stop()`: Stop the server
- `restart()`: Restart the server
- `getServerInfo()`: Get server status information

## Development

### Building

```bash
npm run build
```

### Development Mode

```bash
npm run dev
```

### Testing

```bash
npm test
```

### Running Examples

```bash
# Run the basic example
npm run example

# Or run the compiled example
node dist/examples/basic.js
```

## Project Structure

```
packages/sidecar/
├── src/
│   ├── index.ts          # Main entry point
│   ├── client.ts         # HTTP client
│   ├── server.ts         # Development server
│   ├── types.ts          # TypeScript types
│   └── utils.ts          # Utility functions
├── examples/
│   └── basic.ts          # Basic usage example
├── dist/                 # Compiled output
├── package.json
├── tsconfig.json
└── README.md
```

## Forst File Structure

The sidecar expects Forst files to be organized as follows:

```
forst/
├── routes/
│   ├── process_data.ft
│   ├── search.ft
│   └── calculations.ft
└── utils/
    ├── validation.ft
    └── helpers.ft
```

Each `.ft` file should contain public functions that can be called from TypeScript:

```go
// forst/routes/process_data.ft
package routes

type ProcessDataInput = {
  records: Array({ id: String, data: String })
}

func processData(input ProcessDataInput) {
  // High-performance processing logic
  return { processed: len(input.records), status: "success" }
}
```

## Performance Benefits

- **10x faster latency** compared to TypeScript for CPU-intensive operations
- **Memory efficiency** through Go's garbage collector
- **Concurrent processing** with goroutines
- **Compiled performance** vs interpreted TypeScript

## Error Handling

The sidecar provides comprehensive error handling:

- **Compilation errors**: Graceful handling of Forst compilation failures
- **Runtime errors**: Proper error propagation from Forst to TypeScript
- **Network errors**: Retry logic with exponential backoff
- **Health monitoring**: Automatic detection of server issues

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

MIT License - see LICENSE file for details.
