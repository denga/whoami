# whoami

[![CI](https://github.com/denga/whoami/actions/workflows/ci.yml/badge.svg)](https://github.com/denga/whoami/actions/workflows/ci.yml)
[![Release](https://github.com/denga/whoami/actions/workflows/release.yml/badge.svg)](https://github.com/denga/whoami/actions/workflows/release.yml)
[![Docker Pulls](https://img.shields.io/docker/pulls/denga/whoami)](https://hub.docker.com/r/denga/whoami)
[![Go Report Card](https://goreportcard.com/badge/github.com/denga/whoami)](https://goreportcard.com/report/github.com/denga/whoami)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A tiny Go webserver that prints OS information and HTTP request details to output. Perfect for debugging, testing, and understanding network configurations in containerized environments.

## Features

- **Lightweight**: Minimal Docker image built from scratch (< 10MB)
- **Request Details**: Shows complete HTTP request information
-  **System Info**: Displays OS, architecture, and runtime details
- **Network Info**: Local IPs, real client IP detection
- **Multiple Formats**: Text and JSON output
- **Fast**: Built with Go for high performance
- **Container Ready**: Multi-architecture Docker images
- **Configurable**: Environment variables and command-line flags

## Endpoints

| Endpoint | Description | Content-Type |
|----------|-------------|--------------|
| `/` | Returns request and network information as text | `text/plain` |
| `/api` | Returns request and network information in JSON format | `application/json` |
| `/health` | Health check endpoint | `application/json` |

## Configuration

### Command Line Flags

| Flag | Environment Variable | Description | Default |
|------|---------------------|-------------|---------|
| `-port` | `WHOAMI_PORT_NUMBER` | The port number | `80` |
| `-name` | `WHOAMI_NAME` | The name/identifier | (empty) |
| `-verbose` | | Enable verbose logging | `false` |
| `-version` | | Show version information | |

### Environment Variables

```bash
export WHOAMI_PORT_NUMBER=8080
export WHOAMI_NAME=my-server
```

## Installation

### Download Binary

Download the latest release from the [releases page](https://github.com/denga/whoami/releases):

```bash
# Linux amd64
curl -L https://github.com/denga/whoami/releases/latest/download/whoami_Linux_x86_64.tar.gz | tar xz

# macOS amd64
curl -L https://github.com/denga/whoami/releases/latest/download/whoami_Darwin_x86_64.tar.gz | tar xz

# Windows amd64
curl -L https://github.com/denga/whoami/releases/latest/download/whoami_Windows_x86_64.zip -o whoami.zip
unzip whoami.zip
```

### Build from Source

```bash
git clone https://github.com/denga/whoami.git
cd whoami
go build -o whoami main.go
```

### Docker

```bash
docker pull denga/whoami:latest
```

## Usage

### Basic Usage

```bash
# Run on default port 80
./whoami

# Run on custom port
./whoami -port 8080

# Run with custom name and verbose logging
./whoami -port 8080 -name my-server -verbose

# Show version
./whoami -version
```

### Docker Usage

#### Simple Run

```bash
# Run on port 8080
docker run --rm -p 8080:80 denga/whoami:latest

# Run with custom name
docker run --rm -p 8080:80 -e WHOAMI_NAME=my-container denga/whoami:latest

# Run with custom port
docker run --rm -p 8080:8080 denga/whoami:latest -port 8080
```

#### Docker Compose

Create a `docker-compose.yml` file:

```yaml
version: '3.8'

services:
  whoami:
    image: denga/whoami:latest
    ports:
      - "8080:80"
    environment:
      - WHOAMI_NAME=whoami-service
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s

  # Multiple instances with load balancer
  whoami-1:
    image: denga/whoami:latest
    environment:
      - WHOAMI_NAME=whoami-instance-1
    expose:
      - "80"

  whoami-2:
    image: denga/whoami:latest
    environment:
      - WHOAMI_NAME=whoami-instance-2
    expose:
      - "80"

  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
    depends_on:
      - whoami-1
      - whoami-2
```

Example `nginx.conf` for load balancing:

```nginx
events {
    worker_connections 1024;
}

http {
    upstream whoami_backend {
        server whoami-1:80;
        server whoami-2:80;
    }

    server {
        listen 80;
        
        location / {
            proxy_pass http://whoami_backend;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }
    }
}
```

#### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: whoami
spec:
  replicas: 3
  selector:
    matchLabels:
      app: whoami
  template:
    metadata:
      labels:
        app: whoami
    spec:
      containers:
      - name: whoami
        image: denga/whoami:latest
        ports:
        - containerPort: 80
        env:
        - name: WHOAMI_NAME
          value: "k8s-whoami"
        livenessProbe:
          httpGet:
            path: /health
            port: 80
          initialDelaySeconds: 10
          periodSeconds: 30
        readinessProbe:
          httpGet:
            path: /health
            port: 80
          initialDelaySeconds: 5
          periodSeconds: 10
---
apiVersion: v1
kind: Service
metadata:
  name: whoami-service
spec:
  selector:
    app: whoami
  ports:
  - port: 80
    targetPort: 80
  type: LoadBalancer
```

## Examples

### Text Output (/)

```bash
curl http://localhost:8080/
```

```text
Hostname: my-server
Name: my-whoami
IP: 192.168.1.100, 10.0.0.1
RemoteAddr: 172.17.0.1:52460
Host: localhost:8080
URL: /
Method: GET
RealIP: 172.17.0.1
Protocol: HTTP/1.1
OS: linux
Architecture: amd64
Runtime: go1.24.0
Time: 2024-01-15T10:30:45Z
Version: v1.0.0

Headers:
  Accept: */*
  User-Agent: curl/7.68.0

Environment:
  HOME: /
  PATH: /usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
  WHOAMI_NAME: my-whoami
```

### JSON Output (/api)

```bash
curl http://localhost:8080/api
```

```json
{
  "hostname": "my-server",
  "name": "my-whoami",
  "ip": ["192.168.1.100", "10.0.0.1"],
  "remote_addr": "172.17.0.1:52460",
  "host": "localhost:8080",
  "url": "/api",
  "method": "GET",
  "real_ip": "172.17.0.1",
  "protocol": "HTTP/1.1",
  "headers": {
    "Accept": "*/*",
    "User-Agent": "curl/7.68.0"
  },
  "environment": {
    "HOME": "/",
    "PATH": "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
    "WHOAMI_NAME": "my-whoami"
  },
  "os": "linux",
  "architecture": "amd64",
  "runtime": "go1.24.0",
  "time": "2024-01-15T10:30:45Z",
  "version": "v1.0.0"
}
```

### Health Check (/health)

```bash
curl http://localhost:8080/health
```

```json
{
  "status": "ok",
  "time": "2024-01-15T10:30:45Z",
  "version": "v1.0.0"
}
```

## Use Cases

- **Container Debugging**: Understand network and environment configurations
- **Load Balancer Testing**: Verify request distribution and headers
- **Service Mesh**: Test service-to-service communication
- **Network Debugging**: Troubleshoot routing and proxy configurations
- **CI/CD Testing**: Quick health checks and endpoint testing
- **Development**: Local testing and debugging of HTTP clients

## Development

### Prerequisites

- Go 1.22 or later
- Docker (for container builds)
- golangci-lint (for linting)

### Building

```bash
# Build binary
go build -o whoami main.go

# Run tests
go test -v ./...

# Run linter
golangci-lint run

# Build Docker image
docker build -t whoami .
```

### Testing

```bash
# Run all tests
go test -v -race -coverprofile=coverage.out ./...

# Run benchmarks
go test -bench=. -benchmem ./...

# Test Docker image
docker run --rm -p 8080:80 whoami
curl http://localhost:8080/health
```

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -am 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## Security

This tool is intended for debugging and testing purposes. Be aware that it exposes:

- Environment variables (including potential secrets)
- Network configuration details
- System information

Do not expose this service publicly in production environments without proper security considerations.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Inspired by similar tools in the Docker and Kubernetes ecosystem
- Built with the excellent Go standard library
- Thanks to the Go community for the amazing ecosystem 