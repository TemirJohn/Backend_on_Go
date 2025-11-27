# ==================== BUILD STAGE ====================
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev

# Set working directory
WORKDIR /app

# Copy go mod files
COPY server/go.mod server/go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY server/ .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/main.go

# ==================== RUNTIME STAGE ====================
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Create app directory
WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/main .

# Copy certificates
COPY server/certs ./certs

# Create uploads directory
RUN mkdir -p uploads logs

# Expose port
EXPOSE 8080

# Run the application
CMD ["./main"]