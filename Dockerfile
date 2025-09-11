# Build stage
FROM golang:1.22-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Generate ent code
RUN go run entgo.io/ent/cmd/ent generate ./ent/schema

# Build the application (no CGO needed with entsqlite)
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-w -s' -o vaultwarden-syncer ./cmd/server

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Create directories
RUN mkdir -p /app/data /app/logs /app/data/vaultwarden

WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/vaultwarden-syncer .

# Copy example config
COPY --from=builder /app/config.yaml.example .

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Change ownership of app directory
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8181

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8181/health || exit 1

# Run the application
CMD ["./vaultwarden-syncer"]