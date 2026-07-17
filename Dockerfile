# Build stage: cgo requires a C toolchain.
FROM golang:1.24-bookworm AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -trimpath -ldflags="-s -w" -o /out/ephemeris-service ./cmd/ephemeris-service

# Runtime: slim glibc image (dynamic cgo binary), non-root.
# docker.io-hosted on purpose: gcr.io (distroless) is unreliable from some
# networks, and this image must build anywhere.
FROM debian:bookworm-slim
COPY --from=build /out/ephemeris-service /ephemeris-service
USER 65534:65534
# Swiss Ephemeris .se1 data files can be mounted and pointed to via EPHE_PATH;
# without them the built-in Moshier approximation is used.
EXPOSE 8080
# The binary probes its own /healthz: the slim image has no curl/wget.
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s \
  CMD ["/ephemeris-service", "healthcheck"]
ENTRYPOINT ["/ephemeris-service"]
