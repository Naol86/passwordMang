# syntax=docker/dockerfile:1

FROM golang:1.26.9-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
ARG VERSION=dev
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /out/server ./cmd/server

# Production image: the default target, also used by `make start`.
FROM gcr.io/distroless/static-debian12:nonroot AS runtime
COPY --from=build /out/server /server
EXPOSE 8080 9090
USER nonroot:nonroot
ENTRYPOINT ["/server"]
