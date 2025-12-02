# Build stage
FROM golang:1.24.1-alpine AS builder

WORKDIR /app

# Copy go mod files first
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy all source files
COPY . .

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

# Check if modules exists, if not, fail with clear error
RUN if [ ! -d "modules" ]; then \
        echo "ERROR: modules/ directory is missing from build context!" && \
        echo "This usually means Railway is building from the wrong directory." && \
        echo "Please check Railway Settings -> Root Directory" && \
        exit 1; \
    fi && \
    echo "=== modules/ directory found ===" && \
    ls -la modules/ && \
    echo "=== modules/Auth/ contents ===" && \
    ls -la modules/Auth/ || echo "WARNING: modules/Auth/ not found"

# Ensure module is properly set up
RUN go mod tidy

# Build the application
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

