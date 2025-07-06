# Sidecar Integration MVP Tests

Tiny test cases demonstrating the core sidecar integration functionality.

## Test Cases

### 1. Basic Communication (`basic_communication.ft`)

- **Goal**: Verify HTTP request/response handling
- **Input**: JSON message
- **Expected**: Echo response with timestamp

### 2. Error Handling (`error_handling.ft`)

- **Goal**: Test validation failures and error responses
- **Input**: Invalid value (15)
- **Expected**: Validation error response

### 3. Type Safety (`type_safety.ft`)

- **Goal**: Demonstrate shape definitions and type generation
- **Input**: User query
- **Expected**: User object with validation

## Running Tests

### Direct Forst Tests

```bash
cd examples/in/rfc/sidecar/tests
npx ts-node runner.ts
```

### HTTP Sidecar Tests

```bash
# Terminal 1: Start server
cd examples/in/rfc/sidecar/tests
npx ts-node server.ts

# Terminal 2: Run client tests
npx ts-node client.ts
```

## Expected Results

- **Test 1**: Should return `{"echo": "hello world", "timestamp": 1234567890}`
- **Test 2**: Should return validation error for value 15
- **Test 3**: Should return user object with id, name, and age

All tests are minimal and require no external dependencies beyond Node.js standard library.
