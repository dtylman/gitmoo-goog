FROM golang:1.12 as builder

WORKDIR /project
COPY main.go go.mod go.sum ./
COPY downloader ./downloader
ADD vendor ./vendor
ADD version ./version

# Production-ready build, without debug information specifically for linux
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o=gitmoo-goog -mod=vendor .


FROM alpine:3.12

# Add CA certificates required for SSL connections
RUN apk add --update --no-cache ca-certificates

COPY --from=builder /project/gitmoo-goog /usr/local/bin/gitmoo-goog

RUN mkdir /app
WORKDIR /app
ENTRYPOINT ["/usr/local/bin/gitmoo-goog"]