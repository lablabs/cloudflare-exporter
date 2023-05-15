FROM golang:alpine as builder

WORKDIR /app

COPY cloudflare.go cloudflare.go
COPY main.go main.go
COPY prometheus.go prometheus.go
COPY go.mod go.mod
COPY go.sum go.sum

RUN go get -d -v
RUN CGO_ENABLED=0 GOOS=linux go build -o cloudflare_exporter .

FROM alpine:3.18

RUN apk update && apk add ca-certificates

COPY --from=builder /app/cloudflare_exporter cloudflare_exporter

ENV CF_API_KEY ""
ENV CF_API_EMAIL ""

ENTRYPOINT [ "./cloudflare_exporter" ]
