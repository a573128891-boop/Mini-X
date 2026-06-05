# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install git for fetching deps
RUN apk add --no-cache git

# Copy go mod file and download dependencies
# go mod download works without go.sum (it creates it in module cache)
COPY go.mod ./
RUN go mod download

# Copy source code
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

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
