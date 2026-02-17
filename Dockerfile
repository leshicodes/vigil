# ===========================================================================
# Stage 1: Build frontend
# ===========================================================================
FROM node:22-alpine AS frontend

WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npx vite build

# ===========================================================================
# Stage 2: Build Go binary
# ===========================================================================
FROM golang:alpine AS builder

RUN apk add --no-cache git

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags="-s -w" -o /vigil .

# ===========================================================================
# Stage 3: Runtime
# Using python:3.12-slim instead of Alpine because numpy/OpenCV wheels
# are pre-built for Debian but need compilation on Alpine. This also gives
# us Python out of the box for the analysis hook.
# ===========================================================================
FROM python:3.12-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ffmpeg \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && useradd --create-home vigil

WORKDIR /home/vigil

# Install Python hook dependencies.
COPY hooks/requirements.txt /tmp/requirements.txt
RUN pip install --no-cache-dir -r /tmp/requirements.txt && rm /tmp/requirements.txt

# Copy the Go binary, frontend, and hooks.
COPY --from=builder /vigil /usr/local/bin/vigil
COPY --from=frontend /web/dist ./web/dist
COPY hooks/ ./hooks/

RUN mkdir -p /data/captures && chown -R vigil:vigil /data /home/vigil

USER vigil

EXPOSE 8080

VOLUME ["/data"]

ENTRYPOINT ["vigil"]
CMD ["--data-dir=/data", "--camera=mock", "--static-dir=./web/dist", "--port=8080"]
