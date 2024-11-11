# Define a stage for whatap/kube_mon image to retrieve necessary files
FROM whatap/kube_mon:latest AS whatap_kube_mon

# ===Build cadvisor_helper Binary ===
FROM golang:1.22.7-alpine3.20 AS whatap_cadvisor_helper_build

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
RUN apk add build-base
RUN echo "(2)Build cadvisor Binary"
RUN pwd

# Build cadvisor_helper binary with specified OS and architecture
RUN --mount=type=cache,target="/root/.cache/go-build" CGO_ENABLED=1 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags='-w -extldflags "-static"' -o cadvisor_helper ./cadvisor/cmd/cadvisor-helper/cadvisor_helper.go

RUN ls /data/agent/node

# === Build whatap_control_plane_helper Binary ===
FROM --platform=${BUILDPLATFORM} golang:1.22.7-alpine3.20 AS whatap_control_plane_helper_build

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
FROM --platform=${TARGETPLATFORM} alpine AS packaging

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
COPY --from=whatap_kube_mon /data/agent/tools/whatap_debugger ./tools
COPY --from=whatap_kube_mon /data/agent/sidecar/whatap_sidecar ./sidecar
RUN apk add --no-cache bash
RUN apk add --no-cache curl
RUN apk add --no-cache jq
CMD []