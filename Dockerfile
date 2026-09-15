# Build stage
FROM golang:1.26-alpine AS builder

# Install git and ca-certificates (needed for fetching private modules and HTTPS)
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
# Adjust the path if your main.go is not in ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/api

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/main .

# Copy migrations if they exist (auto-migration runs at startup)
# COPY --from=builder /app/migrations ./migrations

# Expose port
EXPOSE 8080

# Run
CMD ["./main"]
