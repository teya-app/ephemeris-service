# Build stage: cgo requires a C toolchain.
FROM golang:1.24-bookworm AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -trimpath -ldflags="-s -w" -o /out/ephemeris-service ./cmd/ephemeris-service

# Runtime: distroless with glibc (dynamic cgo binary), non-root.
FROM gcr.io/distroless/base-debian12:nonroot
COPY --from=build /out/ephemeris-service /ephemeris-service
# Swiss Ephemeris .se1 data files can be mounted and pointed to via EPHE_PATH;
# without them the built-in Moshier approximation is used.
EXPOSE 8080
ENTRYPOINT ["/ephemeris-service"]
