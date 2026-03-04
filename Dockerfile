FROM golang:1.21-alpine AS build

RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY . .

RUN BUILD_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ) && \
    CGO_ENABLED=0 GOOS=linux go build -mod=vendor -a -installsuffix cgo \
    -ldflags="-X 'github.com/alvarolobato/iptv-proxy/cmd.BuildDate=${BUILD_DATE}'" \
    -o iptv-proxy2 .

FROM alpine:3

COPY --from=build /app/iptv-proxy2 /usr/local/bin/iptv-proxy2

# /data for replacements.json and other data; mount a volume and use --json-folder /data
VOLUME ["/data"]

ENTRYPOINT ["/usr/local/bin/iptv-proxy2"]
