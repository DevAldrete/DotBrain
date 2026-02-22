# syntax=docker/dockerfile:1

# -- Build Stage --
FROM golang:1.25-alpine AS builder

# Set the working directory
WORKDIR /app

# Install necessary build tools (git is often needed for Go modules)
RUN apk add --no-cache git ca-certificates

# Copy go.mod and go.sum files first to leverage Docker cache
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the application binary
# CGO_ENABLED=0 ensures a static build, which is better for minimal alpine images
# -trimpath removes file system paths from the compiled executable
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /app/bin/dotbrain ./cmd/dotbrain

# -- Final Stage --
FROM alpine:3.19

# Set up runtime environment
WORKDIR /app

# Install ca-certificates for HTTPS requests (e.g., calling OpenAI)
RUN apk add --no-cache ca-certificates tzdata

# Create a non-root user for security
RUN adduser -D -g '' dotbrain
USER dotbrain

# Copy the built binary from the builder stage
COPY --from=builder /app/bin/dotbrain /app/dotbrain

# Expose the application port
EXPOSE 8080

# Command to run the executable
CMD ["/app/dotbrain"]
