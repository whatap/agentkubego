FROM --platform=${BUILDPLATFORM} public.ecr.aws/docker/library/golang:1.22.7-alpine3.21 AS whatap_control_plane_helper_build

ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

RUN echo "Kubernetes Metrics Helper Build is running on $BUILDPLATFORM, building for $TARGETPLATFORM"

WORKDIR /data/agent/master
COPY . .

RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags='-w -extldflags "-static"' -o whatap_control_plane_helper ./controlplane/cmd/whatap-control-plane-helper/whatap_control_plane_helper.go

RUN ls /data/agent/master

FROM --platform=${TARGETPLATFORM} public.ecr.aws/docker/library/alpine:3.21 AS packaging

ARG BUILDPLATFORM
ARG BUILDARCH
ARG TARGETPLATFORM
ARG TARGETOS
ARG TARGETARCH

WORKDIR /data/agent
RUN mkdir /data/agent/master
COPY --from=whatap_control_plane_helper_build /data/agent/master/whatap_control_plane_helper ./master
RUN apk update && apk upgrade --no-cache
RUN apk add --no-cache bash
RUN apk add curl
CMD []