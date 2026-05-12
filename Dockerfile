FROM ghcr.io/grpc-ecosystem/grpc-health-probe:v0.4.48 AS grpc_health_probe

# Use official golang image: supports amd64/arm64 multi-arch manifest,
# and we cross-compile so the builder always runs on the host architecture.
FROM golang:1.26 AS builder

ARG GOPROXY=https://goproxy.cn,direct
# Injected automatically by `docker buildx build --platform`
ARG TARGETOS=linux
ARG TARGETARCH=amd64

# Prevent Go from downloading a newer toolchain (honours go.mod toolchain directive locally)
ENV GOTOOLCHAIN=local

WORKDIR /app

RUN --mount=type=cache,target=/root/go/pkg/mod \
    --mount=type=bind,source=go.sum,target=go.sum \
    --mount=type=bind,source=go.mod,target=go.mod \
    GOPROXY=${GOPROXY} go mod download -x

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/root/go/pkg/mod \
    --mount=type=bind,target=. \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} GOPROXY=${GOPROXY} go build -o /bin/openfga ./cmd/openfga

# Runtime: use tag without SHA so buildx selects the correct arch manifest
FROM cgr.dev/chainguard/static:latest

EXPOSE 8081
EXPOSE 8080
EXPOSE 3000

COPY --from=grpc_health_probe /ko-app/grpc-health-probe /usr/local/bin/grpc_health_probe
COPY --from=builder /bin/openfga /openfga

HEALTHCHECK --interval=5s --timeout=30s --retries=3 CMD ["/usr/local/bin/grpc_health_probe", "-addr=:8081"]

ENTRYPOINT ["/openfga"]
