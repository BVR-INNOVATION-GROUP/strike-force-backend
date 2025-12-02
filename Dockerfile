# Build stage
FROM golang:1.24.1-alpine AS builder

WORKDIR /app

# Copy go mod files first
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy all source files (modules, config, main.go, etc.)
COPY main.go ./
COPY config/ ./config/
COPY modules/ ./modules/

# Verify modules were copied correctly
RUN echo "=== Verifying copied files ===" && \
    echo "main.go exists:" && test -f main.go && echo "✓" || echo "✗" && \
    echo "config/ exists:" && test -d config && echo "✓" || echo "✗" && \
    echo "modules/ exists:" && test -d modules && echo "✓" || echo "✗" && \
    echo "=== Contents of modules/ ===" && \
    ls -la modules/ && \
    echo "=== Checking for Auth ===" && \
    (test -d modules/Auth && echo "modules/Auth/ EXISTS" && ls -la modules/Auth/ || echo "modules/Auth/ MISSING - this is the problem!")

# Debug: Show what was actually copied
RUN echo "=== Build Context Contents ===" && \
    echo "Current directory:" && pwd && \
    echo "Files in /app:" && ls -la && \
    echo "=== Looking for modules ===" && \
    find . -name "modules" -type d 2>/dev/null || echo "modules directory not found" && \
    echo "=== Looking for main.go ===" && \
    find . -name "main.go" 2>/dev/null || echo "main.go not found" && \
    echo "=== Looking for config ===" && \
    find . -name "config" -type d 2>/dev/null || echo "config directory not found"

# Verify modules/Auth exists (critical check)
RUN if [ ! -d "modules/Auth" ]; then \
        echo "========================================" && \
        echo "ERROR: modules/Auth/ is missing!" && \
        echo "========================================" && \
        echo "This means the modules subdirectories are not in the build context." && \
        echo "" && \
        echo "Possible causes:" && \
        echo "1. Railway Root Directory is set incorrectly" && \
        echo "2. modules/ directory is not committed to git" && \
        echo "3. .dockerignore is excluding modules (check .dockerignore)" && \
        echo "" && \
        echo "Current modules/ contents:" && \
        ls -la modules/ 2>/dev/null || echo "modules/ doesn't exist" && \
        echo "" && \
        echo "Please check:" && \
        echo "- Railway Settings -> Root Directory (should be '.' or empty)" && \
        echo "- Git repo has modules/ committed" && \
        exit 1; \
    fi && \
    echo "✓ modules/Auth/ found" && \
    ls -la modules/Auth/

# Build the application (skip go mod tidy - it tries to fetch from git)
# Local packages should be resolved automatically
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o out .

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/out .

# Create uploads directory structure for static file serving
RUN mkdir -p uploads/applications uploads/organizations/logos uploads/projects

# Expose port (Railway will set PORT env var)
EXPOSE 8080

# Run the application
CMD ["./out"]

