# Define a stage for kube_mon image to retrieve necessary files
#FROM whatap/kube_mon_dev:1.7.15-sec AS whatap_kube_mon

# ===Build cadvisor_helper Binary ===
FROM --platform=$BUILDPLATFORM public.ecr.aws/docker/library/golang:1.22.7 AS whatap_cadvisor_helper_build

# Build arguments for cross-platform compilation
ARG TARGETOS
ARG TARGETARCH
RUN echo "Kubernetes Cadvisor Helper Build is running"

# Set working directory
WORKDIR /data/agent/node
COPY . .
RUN go mod download
# Install build dependencies
RUN echo "(1)Install base GCC"
RUN apt-get update && \
    if [ "$TARGETARCH" = "arm64" ]; then \
      apt-get install -y gcc-aarch64-linux-gnu libc6-dev-arm64-cross ; \
    else \
      apt-get install -y gcc ; \
    fi && \
    apt-get install -y build-essential ca-certificates
# RUN apk add build-base
RUN echo "(2)Build cadvisor Binary"
RUN pwd

# Build cadvisor_helper binary with specified OS and architecture
# RUN --mount=type=cache,target="/root/.cache/go-build" CGO_ENABLED=1 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags='-w -extldflags "-static"' -o cadvisor_helper ./cadvisor/cmd/cadvisor-helper/cadvisor_helper.go
RUN --mount=type=cache,target="/root/.cache/go-build" \
    sh -c 'CC=$( [ "$TARGETARCH" = "arm64" ] && echo "aarch64-linux-gnu-gcc" || echo "gcc" ) && \
           CGO_ENABLED=1 GOOS=$TARGETOS GOARCH=$TARGETARCH CC=$CC \
           go build -ldflags="-w -extldflags \"-static\"" \
           -o cadvisor_helper ./cadvisor/cmd/cadvisor-helper/cadvisor_helper.go'

RUN ls /data/agent/node

# ===Build debugger Binary ===
FROM --platform=$BUILDPLATFORM public.ecr.aws/docker/library/golang:1.22.7 AS whatap_debugger_build
ARG TARGETOS
ARG TARGETARCH
RUN echo "Kubernetes Node Debugger Build is running"
WORKDIR /data/agent/tools
COPY . .
RUN go mod download
RUN pwd

# Install build dependencies
RUN echo "(1)Install base GCC"
RUN apt-get update && \
    if [ "$TARGETARCH" = "arm64" ]; then \
      apt-get install -y gcc-aarch64-linux-gnu libc6-dev-arm64-cross ; \
    else \
      apt-get install -y gcc ; \
    fi && \
    apt-get install -y build-essential ca-certificates

# RUN --mount=type=cache,target="/root/.cache/go-build" GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags='-w -extldflags "-static"' -o whatap_debugger ./debugger/cmd/whatap-debugger/whatap_debugger.go
RUN --mount=type=cache,target="/root/.cache/go-build" \
    sh -c 'CC=$( [ "$TARGETARCH" = "arm64" ] && echo "aarch64-linux-gnu-gcc" || echo "gcc" ) && \
           CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH CC=$CC \
           go build -ldflags="-w -extldflags \"-static\"" \
           -o whatap_debugger ./debugger/cmd/whatap-debugger/whatap_debugger.go'

RUN ls /data/agent/tools

# === Build whatap_control_plane_helper Binary ===
FROM --platform=${BUILDPLATFORM} public.ecr.aws/docker/library/golang:1.22-alpine3.21 AS whatap_control_plane_helper_build

# Build arguments for cross-platform compilation
ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

RUN echo "Kubernetes Metrics Helper Build is running on $BUILDPLATFORM, building for $TARGETPLATFORM"

# Set working directory
WORKDIR /data/agent/master
COPY . .
RUN go mod download
# Build whatap_control_plane_helper binary with specified OS and architecture
RUN --mount=type=cache,target="/root/.cache/go-build" CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags='-w -extldflags "-static"' -o whatap_control_plane_helper ./controlplane/cmd/whatap-control-plane-helper/whatap_control_plane_helper.go
RUN ls /data/agent/master

# === Final Packaging ===
FROM --platform=${TARGETPLATFORM} public.ecr.aws/docker/library/alpine:3.21 AS packaging

ARG BUILDPLATFORM
ARG BUILDARCH
ARG TARGETPLATFORM
ARG TARGETOS
ARG TARGETARCH

WORKDIR /data/agent
RUN mkdir /data/agent/node
RUN mkdir /data/agent/master
RUN mkdir /data/agent/tools
RUN mkdir /data/agent/sidecar
COPY --from=whatap_cadvisor_helper_build /data/agent/node/cadvisor_helper ./node
COPY --from=whatap_control_plane_helper_build /data/agent/master/whatap_control_plane_helper ./master
COPY --from=whatap_debugger_build /data/agent/tools/whatap_debugger ./tools
#COPY --from=whatap_kube_mon /data/agent/sidecar/whatap_sidecar ./sidecar
RUN apk update && apk upgrade --no-cache
RUN apk add --no-cache bash
RUN apk add --no-cache curl
RUN apk add --no-cache jq
CMD []