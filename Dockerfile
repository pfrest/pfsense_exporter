# Set up the build environment
FROM golang:1.25-alpine AS builder
WORKDIR /app

# Allow the build version to be set at build time
ARG DOCKER_IMAGE_VERSION
ENV DOCKER_IMAGE_VERSION=${DOCKER_IMAGE_VERSION:-"v0.0.0-dev"}

# Copy source code into the container and build the exporter
COPY . .
RUN go mod download
RUN go build -o pfsense_exporter -ldflags="-X 'main.buildString=$DOCKER_IMAGE_VERSION'" ./cmd/pfsense_exporter

# Setup the final container on Alpine Linux
FROM alpine:latest
WORKDIR /pfsense_exporter

COPY --from=builder /app/pfsense_exporter /usr/local/bin/pfsense_exporter

# Run the exporter
CMD ["/usr/local/bin/pfsense_exporter", "--config", "/pfsense_exporter/config.yml"]