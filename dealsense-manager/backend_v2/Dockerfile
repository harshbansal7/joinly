# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the application with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
    -ldflags="-w -s -X main.version=$(git describe --tags --always 2>/dev/null || echo 'dev')" \
    -a -installsuffix cgo \
    -o dealsense-manager \
    ./cmd/server

# Final stage - use distroless for smaller attack surface
FROM gcr.io/distroless/static-debian12

# Copy timezone data and certificates
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Create a non-root user and group
USER 65534:65534

# Set working directory
WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/dealsense-manager .

# Expose port
EXPOSE 8001

# Run the binary
CMD ["/app/dealsense-manager"]
