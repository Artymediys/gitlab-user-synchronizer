# ---------- BUILD STAGE ----------
FROM golang:1.24.4-alpine AS builder

WORKDIR /src
COPY . .

RUN go mod download && \
    CGO_ENABLED=0 \
    go build -trimpath -ldflags="-s -w" \
    -o /gus ./...

# ---------- RUNTIME STAGE ----------
FROM alpine:3.22.0

COPY certs/. /usr/local/share/ca-certificates/
RUN apk add --no-cache ca-certificates && \
    update-ca-certificates && \
    adduser -D -h /home/gus gus

COPY --from=builder /gus /usr/local/bin/gus

USER gus
WORKDIR /home/gus

ENTRYPOINT ["/usr/local/bin/gus"]
