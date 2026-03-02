FROM golang:1.21-alpine AS build

RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -a -installsuffix cgo -o iptv-proxy2 .

FROM alpine:3

COPY --from=build /app/iptv-proxy2 /usr/local/bin/iptv-proxy2

# /data for replacements.json and other data; mount a volume and use --json-folder /data
VOLUME ["/data"]

ENTRYPOINT ["/usr/local/bin/iptv-proxy2"]
