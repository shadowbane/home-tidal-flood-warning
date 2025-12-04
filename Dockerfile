# Stage 1: Build
FROM golang:1.23 AS builder

LABEL maintainer="Adli I. Ifkar <adly.shadowbane@gmail.com>"

# Install UPX
RUN apt-get update && apt-get install -y --no-install-recommends upx && rm -rf /var/lib/apt/lists/*

WORKDIR /build

# Copy go mod files first for better caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build for linux amd64 (CGO enabled for SQLite support)
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o bin/home-tidal-flood-warning cmd/api/main.go

# Compress with UPX
RUN upx --best --lzma bin/home-tidal-flood-warning

# Stage 2: Runtime
FROM debian:bookworm-slim

# Install ca-certificates for HTTPS requests
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates tzdata && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/bin/home-tidal-flood-warning /app/tidal-flood-warning

ENTRYPOINT ["/app/tidal-flood-warning"]