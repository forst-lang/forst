# Sidecar Integration Tests

This directory contains integration tests for the Forst sidecar functionality, demonstrating HTTP communication between Node.js and Forst applications.

## Overview

The tests demonstrate:

- HTTP communication between Node.js and Forst
- JSON-based low-overhead communication
- Health checks and error handling
- Running Forst test files via HTTP API

## Running the Tests

### Prerequisites

1. Make sure you have Go installed and the Forst compiler built
2. Ensure you're in the sidecar directory: `cd examples/in/rfc/sidecar`

### Option 1: Using npm scripts

```bash
# Start the Forst development server
npm run dev:server

# In another terminal, run the tests
npm run test:sidecar
```

### Option 2: Manual setup

```bash
# Start the Forst HTTP server
./start-forst-server.sh

# In another terminal, run the tests
npx tsx tests/runner.ts
```

### Option 3: Using concurrently (starts both server and tests)

```bash
npm run dev
```

## Test Files

- `basic_communication.ft`: Tests basic echo functionality
- `type_safety.ft`: Tests type safety and validation

## HTTP API Endpoints

The Forst HTTP server provides the following endpoints:

- `GET /health`: Health check endpoint
- `POST /run`: Run a Forst test file
- `POST /compile`: Compile a Forst file

### Example Usage

```bash
# Health check
curl http://localhost:8080/health

# Run a test file
curl -X POST http://localhost:8080/run \
  -H "Content-Type: application/json" \
  -d '{"action": "run", "testFile": "/path/to/test.ft"}'
```

## Architecture

The integration follows the sidecar pattern described in the RFC:

1. **Node.js Test Runner**: Sends HTTP requests to Forst server
2. **Forst HTTP Server**: Compiles and runs Forst files
3. **JSON Communication**: Low-overhead JSON-based communication
4. **Error Handling**: Proper error propagation and health checks

## Troubleshooting

### Server not starting

- Ensure Go is installed and `go` command is available
- Check that the Forst binary can be built: `cd ../../../../../forst && go build -o bin/forst cmd/forst/main.go`

### Connection refused

- Make sure the server is running on port 8080
- Check that no other process is using port 8080
- Verify the server started successfully by checking the logs

### Test failures

- Check the Forst server logs for compilation errors
- Verify the test files exist and are valid Forst code
- Ensure the server has proper permissions to read the test files
