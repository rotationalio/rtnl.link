# Dynamic Builds
ARG BUILDER_IMAGE=golang:1.21-bookworm
ARG FINAL_IMAGE=debian:bookworm-slim

# Build stage
FROM --platform=${BUILDPLATFORM} ${BUILDER_IMAGE} AS builder

# Build Args
ARG GIT_REVISION=""

# Ensure ca-certificates are up to date on the image
RUN update-ca-certificates

# Use modules for dependencies
WORKDIR $GOPATH/src/github.com/rotationalio/rtnl.link

COPY go.mod .
COPY go.sum .

ENV CGO_ENABLED=1
ENV GO111MODULE=on
RUN go mod download
RUN go mod verify

# Copy package
COPY . .

# Build binary
ARG TARGETOS
ARG TARGETARCH
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -v -o /go/bin/rtnl -ldflags="-X 'github.com/rotationalio/rtnl.link/pkg.GitVersion=${GIT_REVISION}'" ./cmd/rtnl

# Final Stage
FROM --platform=${BUILDPLATFORM} ${FINAL_IMAGE} AS final

LABEL maintainer="Rotational Labs <support@rotational.io>"
LABEL description="Rotational Labs' url shortner and click tracking service"

# Ensure ca-certificates are up to date and install sqlite3
RUN set -x && apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y ca-certificates sqlite3 && \
    rm -rf /var/lib/apt/lists/*

# Copy the binary to the production image from the builder stage
COPY --from=builder /go/bin/rtnl /usr/local/bin/rtnl

# Create a user so that we don't run as root
RUN groupadd -r rtnl && useradd -m -r -g rtnl rtnl
USER rtnl

CMD [ "/usr/local/bin/rtnl", "serve" ]