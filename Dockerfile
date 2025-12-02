# Build stage
FROM golang:1.24.1-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy all source files
COPY . .

# Build the application
RUN go build -ldflags="-w -s" -o out

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

