# syntax=docker/dockerfile:1.6
# Build stage
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY *.go ./

# Build the binary with static linking
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags="-w -s -extldflags '-static'" -o cloudflare_exporter .

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS calls to Cloudflare API
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/cloudflare_exporter .

# Expose metrics port
EXPOSE 8080

# Run the exporter
ENTRYPOINT ["./cloudflare_exporter"]
