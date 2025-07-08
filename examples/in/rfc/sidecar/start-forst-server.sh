#!/bin/bash

# Start Forst development server
# This script starts the Forst HTTP server for Node.js communication

set -e

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
FORST_DIR="$(cd "$SCRIPT_DIR/../../../../../forst" && pwd)"

# Use PORT environment variable or default to 8080
PORT="${PORT:-8080}"

echo "Starting Forst development server..."
echo "Forst directory: $FORST_DIR"
echo "Port: $PORT"

# Check if forst binary exists
if [ ! -f "$FORST_DIR/bin/forst" ]; then
    echo "Building Forst binary..."
    cd "$FORST_DIR"
    go build -o bin/forst cmd/forst/main.go
fi

# Start the development server
echo "Starting Forst HTTP server on port $PORT..."
"$FORST_DIR/bin/forst" dev -port="$PORT" 