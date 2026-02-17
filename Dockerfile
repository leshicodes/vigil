# --- Build Stage ---
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /vigil .

# --- Runtime Stage ---
FROM alpine:3.19

RUN apk add --no-cache \
    ca-certificates \
    libcamera \
    fswebcam \
    && adduser -D -h /home/vigil vigil

WORKDIR /home/vigil
COPY --from=builder /vigil /usr/local/bin/vigil

RUN mkdir -p /data/captures && chown -R vigil:vigil /data

USER vigil

EXPOSE 8080

VOLUME ["/data"]

ENTRYPOINT ["vigil"]
CMD ["--data-dir=/data", "--camera=mock", "--port=8080"]
