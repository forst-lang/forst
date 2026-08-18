# Multi-stage build for Forst compiler images.
# Build from repo root: docker build -f docker/Dockerfile .
# Default target includes the Go toolchain; use --target minimal for compiler-only.

ARG GO_VERSION=1.26-bookworm

FROM golang:${GO_VERSION} AS builder

SHELL ["/bin/bash", "-euo", "pipefail", "-c"]

WORKDIR /src/forst

COPY forst/go.mod forst/go.sum ./
RUN go mod download

COPY forst/ ./

ARG VERSION=dev
ARG COMMIT=unknown
ARG DATE=unknown

    RUN set -euo pipefail; \
    PGO_FLAG=""; \
    if [ -f default.pgo ]; then PGO_FLAG="-pgo=default.pgo"; fi; \
    LDFLAGS="-s -w -X main.Version=${VERSION} -X main.Commit=${COMMIT} -X main.Date=${DATE}"; \
    CGO_ENABLED=0 go build ${PGO_FLAG} -ldflags="${LDFLAGS}" -o /out/forst ./cmd/forst; \
    for pkg in forst-gen-jsonschema forst-gen-orpc forst-gen-file-routes forst-gen-react-router forst-gen-echo; do \
      CGO_ENABLED=0 go build ${PGO_FLAG} -ldflags="${LDFLAGS}" -o "/out/${pkg}" "./plugins/${pkg}"; \
    done

# Side image: compiler only (generate, fmt, build, dev, lsp — no Go toolchain).
FROM alpine:3.21 AS minimal

RUN apk --no-cache add ca-certificates

COPY --from=builder /out/forst /usr/local/bin/forst
COPY --from=builder /out/forst-gen-jsonschema /usr/local/bin/forst-gen-jsonschema
COPY --from=builder /out/forst-gen-orpc /usr/local/bin/forst-gen-orpc
COPY --from=builder /out/forst-gen-file-routes /usr/local/bin/forst-gen-file-routes
COPY --from=builder /out/forst-gen-react-router /usr/local/bin/forst-gen-react-router
COPY --from=builder /out/forst-gen-echo /usr/local/bin/forst-gen-echo

LABEL org.opencontainers.image.source=https://github.com/forst-lang/forst

USER 65532:65532
WORKDIR /workspace
ENTRYPOINT ["forst"]

# Default image: compiler + Go toolchain (run, test, generate, and full workflows).
FROM golang:${GO_VERSION} AS default

COPY --from=builder /out/forst /usr/local/bin/forst
COPY --from=builder /out/forst-gen-jsonschema /usr/local/bin/forst-gen-jsonschema
COPY --from=builder /out/forst-gen-orpc /usr/local/bin/forst-gen-orpc
COPY --from=builder /out/forst-gen-file-routes /usr/local/bin/forst-gen-file-routes
COPY --from=builder /out/forst-gen-react-router /usr/local/bin/forst-gen-react-router
COPY --from=builder /out/forst-gen-echo /usr/local/bin/forst-gen-echo

LABEL org.opencontainers.image.source=https://github.com/forst-lang/forst

WORKDIR /workspace
ENTRYPOINT ["forst"]
