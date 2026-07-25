# syntax=docker/dockerfile:1

# K6_VERSION must stay on the same module major as go.mod: the extension imports
# go.k6.io/k6/v2, so building it against a k6 v1 release fails with
# "conflicting k6 versions".
ARG GO_VERSION=1.25
ARG K6_VERSION=v2.1.0

# Pinned to the build platform and cross-compiled via GOOS/GOARCH, so multi-arch
# builds do not run the Go toolchain under QEMU emulation.
FROM --platform=${BUILDPLATFORM} golang:${GO_VERSION}-alpine AS builder

ARG K6_VERSION
ARG TARGETOS
ARG TARGETARCH
# xk6 v1.4+ reads K6_VERSION as the --k6-version flag. Passing it as a
# positional argument as well fails with "k6 version was specified with both
# flag and argument", so it is only ever set as an env var.
ENV K6_VERSION=${K6_VERSION}
ENV CGO_ENABLED=0

RUN apk add --no-cache git

WORKDIR /build

RUN go install go.k6.io/xk6/cmd/xk6@latest

# Warm the module cache from the manifests alone, so editing sources does not
# re-download the k6 dependency tree.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# GOOS/GOARCH are set here rather than as ENV so that `go install xk6` above
# still produces a binary that runs on the build platform.
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    xk6 build --with github.com/moko-poi/xk6-valkey=. --output /build/k6

FROM alpine:3.22

RUN apk add --no-cache ca-certificates && \
    adduser -D -u 12345 -g 12345 k6

COPY --from=builder /build/k6 /usr/bin/k6

USER k6
WORKDIR /home/k6

ENTRYPOINT ["k6"]
CMD ["--help"]
