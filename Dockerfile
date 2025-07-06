# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . ./

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o forst ./forst/cmd/forst

# Final stage
FROM alpine:3.22.0

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1001 -S forst && \
    adduser -u 1001 -S forst -G forst

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/forst .

# Change ownership to non-root user
RUN chown -R forst:forst /app

# Switch to non-root user
USER forst

# Health check disabled - CLI tool, not a service
# HEALTHCHECK NONE

# Set the binary as entrypoint
ENTRYPOINT ["./forst"]

# Default command
CMD ["--help"] 