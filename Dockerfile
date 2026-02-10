# ============================================
# Stage 1: Builder
# ============================================
FROM golang:1.25.6-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /build

# Copy dependency files first (for better caching)
COPY go.mod go.sum ./
ENV GOPROXY=direct
RUN go mod download

# Copy source code
COPY . .

# Build the application
# - Disable CGO for static binary
# - Strip debug symbols for smaller binary
# - Enable optimizations
RUN GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o api \
    ./cmd/api

# ============================================
# Stage 2: Runtime (Minimal Production Image)
# ============================================
FROM alpine:3.23

# Install CA certificates for HTTPS requests
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user for security
RUN addgroup -g 1000 appgroup && \
    adduser -D -u 1000 -G appgroup appuser

# Set working directory
WORKDIR /app

# Copy compiled binary from builder
COPY --from=builder /build/api .

# Copy Swagger docs (required even in prod for the docs package import)
# The routes themselves won't be registered in production
COPY --from=builder /build/docs ./docs

# Copy migrations (if needed at runtime)
COPY --from=builder /build/migrations ./migrations

# Change ownership to non-root user
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose port 8080
EXPOSE 8080

# Set production environment by default
# This ensures Swagger routes are NOT registered
ENV APP_ENV=prod

# Health check (optional but recommended)
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
CMD ["./api"]
