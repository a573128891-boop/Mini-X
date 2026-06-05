# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install git for fetching deps
RUN apk add --no-cache git

# Copy go mod and sum files first (for layer caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o server ./cmd/server

# Runtime stage
FROM alpine:3.19

WORKDIR /app

# Install CA certificates for HTTPS
RUN apk add --no-cache ca-certificates

# Copy binary from builder
COPY --from=builder /app/server .

# Expose port
EXPOSE 3001

# Run
CMD ["./server"]
