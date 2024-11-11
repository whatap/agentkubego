FROM golang:1.22.7-alpine3.20 AS whatap_cadvisor_helper_build

ARG TARGETOS
ARG TARGETARCH
RUN echo "Kubernetes Cadvisor Helper Build is running"

WORKDIR /data/agent/node
COPY ./cadvisor .
RUN echo "(1)Install base GCC"
RUN apk add build-base
RUN echo "(2)Build cadvisor Binary"
RUN pwd
RUN CGO_ENABLED=1 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags='-w -extldflags "-static"' -o cadvisor_helper ./cadvisor/cmd/cadvisor-helper/cadvisor_helper.go

RUN ls /data/agent/node

FROM --platform=${TARGETPLATFORM} alpine AS packaging

ARG BUILDPLATFORM
ARG BUILDARCH
ARG TARGETPLATFORM
ARG TARGETOS
ARG TARGETARCH

WORKDIR /data/agent
RUN mkdir /data/agent/node
COPY --from=whatap_cadvisor_helper_build /data/agent/node/cadvisor_helper ./node
RUN apk add --no-cache bash
RUN apk add --no-cache curl
RUN apk add --no-cache jq
CMD []