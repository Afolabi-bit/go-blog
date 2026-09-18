# ==========================================
# Stage 1: Build the Go binary
# ==========================================
FROM golang:alpine AS builder

WORKDIR /build

# Install git and ca-certificates (needed for fetching dependencies & TLS)
RUN apk add --no-cache git ca-certificates

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy application source code
COPY . .

# Build statically linked binary with debug symbols stripped (-s -w)
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /build/api ./cmd/api

# ==========================================
# Stage 2: Minimal Production Runtime
# ==========================================
FROM alpine:latest

WORKDIR /app

# Install ca-certificates and tzdata for TLS with MongoDB Atlas and accurate timestamps
RUN apk --no-cache add ca-certificates tzdata

# Create unprivileged non-root user for security
RUN adduser -D -g '' appuser

# Copy the compiled binary from builder stage
COPY --from=builder /build/api /app/api

# Run as non-root user
USER appuser

# Expose default application port
EXPOSE 5000

# Run the API server
ENTRYPOINT ["/app/api"]
