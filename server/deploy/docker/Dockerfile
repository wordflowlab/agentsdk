# Multi-stage build for AgentSDK Server

# Stage 1: Build
FROM golang:1.21-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git make

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the server
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-w -s' \
    -o agentsdk-server ./cmd/agentsdk-server

# Stage 2: Runtime
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/agentsdk-server .

# Create data directory
RUN mkdir -p .data

# Expose ports
EXPOSE 8080 9090

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run as non-root user
RUN adduser -D -u 1000 agentsdk
RUN chown -R agentsdk:agentsdk /app
USER agentsdk

# Start server
ENTRYPOINT ["./agentsdk-server"]
