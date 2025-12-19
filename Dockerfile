# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o payment-sim ./cmd/payment-sim

# Runtime stage
FROM alpine:3.19

# Add non-root user for security
RUN adduser -D -g '' appuser

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/payment-sim /usr/local/bin/payment-sim

# Use non-root user
USER appuser

# Set entrypoint
ENTRYPOINT ["payment-sim"]
