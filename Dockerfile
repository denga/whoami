# Build stage
FROM golang:1.24-alpine AS builder

# Install ca-certificates for https requests (if needed)
RUN apk --no-cache add ca-certificates git tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod ./

# Download dependencies (if any)
RUN go mod download

# Copy source code
COPY main.go ./

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o whoami \
    main.go

# Final stage - minimal image
FROM scratch

# Copy ca-certificates from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy timezone data (optional, for proper time handling)
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy the binary
COPY --from=builder /app/whoami /whoami

# Create non-root user
# Note: scratch doesn't have adduser, so we copy from builder
COPY --from=builder /etc/passwd /etc/passwd

# Labels for better metadata
LABEL maintainer="dennis.gaedeke@gmail.com"
LABEL org.opencontainers.image.source="https://github.com/denga/whoami"
LABEL org.opencontainers.image.description="Tiny Go webserver that prints OS information and HTTP request to output"
LABEL org.opencontainers.image.licenses="MIT"

# Expose the default port
EXPOSE 80

# Health check - we can't use curl in scratch, so we'll rely on external health checks
# HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
#     CMD ["/whoami", "-version"]

# Run as non-root user
USER 65534:65534

# Set the entrypoint
ENTRYPOINT ["/whoami", "-verbose"]

# Default command (can be overridden)
CMD [] 