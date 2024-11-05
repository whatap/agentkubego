module whatap.io/k8s/sidecar

go 1.22.7

require (
	github.com/natefinch/lumberjack v2.0.0+incompatible
	github.com/whatap/golib v0.0.20
	gitlab.whatap.io/hsnam/focus-agent v0.0.0
	k8s.io/api v0.20.0-alpha.2
	k8s.io/apimachinery v0.20.0-alpha.2
	k8s.io/client-go v0.20.0-alpha.2
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-logr/logr v0.2.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/google/gofuzz v1.1.0 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/googleapis/gnostic v0.4.1 // indirect
	github.com/json-iterator/go v1.1.10 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/rogpeppe/go-internal v1.13.1 // indirect
	github.com/whatap/go-api v0.1.10 // indirect
	golang.org/x/crypto v0.28.0 // indirect
	golang.org/x/oauth2 v0.6.0 // indirect
	golang.org/x/sys v0.26.0 // indirect
	golang.org/x/term v0.25.0 // indirect
	golang.org/x/text v0.19.0 // indirect
	golang.org/x/time v0.3.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/protobuf v1.33.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	k8s.io/klog/v2 v2.2.0 // indirect
	k8s.io/utils v0.0.0-20200729134348-d5654de09c73 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.0.2-0.20201001033253-b3cf1e8ff931 // indirect
	sigs.k8s.io/yaml v1.2.0 // indirect
)

require golang.org/x/net v0.23.0 // indirect; CVE-2022-27664, CVE-2022-41721, CVE-2022-41723, CVE-2023-39325 해결

replace (
	github.com/google/gnostic v0.7.0 => github.com/google/gnostic v0.7.0
	gitlab.whatap.io/hsnam/focus-agent v0.0.0 => ../focus-agent
)