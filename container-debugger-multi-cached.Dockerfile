FROM public.ecr.aws/docker/library/golang:1.22.7-alpine3.21 AS whatap_debugger_build

ARG TARGETOS
ARG TARGETARCH
RUN echo "Kubernetes Node Debugger Build is running"
WORKDIR /data/agent/tools
COPY . .
RUN go mod download
RUN pwd
RUN --mount=type=cache,target="/root/.cache/go-build" GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags='-w -extldflags "-static"' -o whatap_debugger ./debugger/cmd/whatap-debugger/whatap_debugger.go

RUN ls /data/agent/tools

FROM --platform=${TARGETPLATFORM} public.ecr.aws/docker/library/alpine:3.21 AS packaging

ARG BUILDPLATFORM
ARG BUILDARCH
ARG TARGETPLATFORM
ARG TARGETOS
ARG TARGETARCH

WORKDIR /data/agent
RUN mkdir -p /data/agent/tools
COPY --from=whatap_debugger_build /data/agent/tools/whatap_debugger ./tools
RUN apk update && apk upgrade --no-cache
RUN apk add --no-cache bash
RUN apk add --no-cache curl
RUN apk add --no-cache jq
CMD []