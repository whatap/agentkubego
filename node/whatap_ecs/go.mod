module whatap.io/aws/ecs

go 1.22.7

require (
	github.com/aws/aws-sdk-go v1.35.24
	github.com/docker/docker v27.3.1+incompatible
	github.com/natefinch/lumberjack v2.0.0+incompatible
	github.com/whatap/go-api v0.1.10
	github.com/whatap/kube/node v0.0.0
	gitlab.whatap.io/hsnam/focus-agent v0.0.0
)

require (
	github.com/Azure/go-ansiterm v0.0.0-20210617225240-d185dfc1b5a1 // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/docker/distribution v2.8.3+incompatible // indirect
	github.com/docker/go-connections v0.5.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/hako/durafmt v0.0.0-20210608085754-5c1018a4e16b // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/stretchr/testify v1.9.0 // indirect
	github.com/whatap/golib v0.0.29 // indirect
	golang.org/x/mod v0.17.0 // indirect
	golang.org/x/net v0.28.0 // indirect
	golang.org/x/sys v0.26.0 // indirect
	golang.org/x/text v0.19.0 // indirect
	golang.org/x/tools v0.21.1-0.20240508182429-e35e4ccd0d2d // indirect
)

replace (
	github.com/whatap/kube/node v0.0.0 => ../../golang
	gitlab.whatap.io/hsnam/focus-agent v0.0.0 => ../focus-agent
)
