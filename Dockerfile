# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    --ldflags '-w -s -extldflags "-static"' \
    -o /cloudflare_exporter .

FROM alpine:3.22
RUN apk add --no-cache ca-certificates
COPY --from=builder /cloudflare_exporter /cloudflare_exporter
USER 65534:65534
ENTRYPOINT ["/cloudflare_exporter"]
