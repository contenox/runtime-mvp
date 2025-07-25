module github.com/contenox/runtime-mvp/core

go 1.24.4

toolchain go1.24.5

// libauth libbus  libcipher  libdb  libkv  libollama  libroutine libtestenv
replace github.com/contenox/runtime-mvp/libs/libauth => ../libs/libauth

replace github.com/contenox/runtime-mvp/libs/libbus => ../libs/libbus

replace github.com/contenox/runtime-mvp/libs/libcipher => ../libs/libcipher

replace github.com/contenox/runtime-mvp/libs/libdb => ../libs/libdb

replace github.com/contenox/runtime-mvp/libs/libroutine => ../libs/libroutine

replace github.com/contenox/runtime-mvp/libs/libtestenv => ../libs/libtestenv

replace github.com/contenox/runtime-mvp/libs/libkv => ../libs/libkv

replace github.com/contenox/runtime-mvp/libs/libmodelprovider => ../libs/libmodelprovider

require (
	dario.cat/mergo v1.0.2
	github.com/contenox/runtime-mvp/libs/libauth v0.0.0-20250626094131-df93eea0ce6a
	github.com/contenox/runtime-mvp/libs/libbus v0.0.0-00010101000000-000000000000
	github.com/contenox/runtime-mvp/libs/libcipher v0.0.0-00010101000000-000000000000
	github.com/contenox/runtime-mvp/libs/libdb v0.0.0-00010101000000-000000000000
	github.com/contenox/runtime-mvp/libs/libkv v0.0.0-20250714091341-746b94b8a904
	github.com/contenox/runtime-mvp/libs/libmodelprovider v0.0.0-00010101000000-000000000000
	github.com/contenox/runtime-mvp/libs/libroutine v0.0.0-00010101000000-000000000000
	github.com/contenox/runtime-mvp/libs/libtestenv v0.0.0-00010101000000-000000000000
	github.com/go-telegram-bot-api/telegram-bot-api/v5 v5.5.1
	github.com/google/uuid v1.6.0
	github.com/lib/pq v1.10.9
	github.com/ollama/ollama v0.9.6
	github.com/stretchr/testify v1.10.0
	github.com/testcontainers/testcontainers-go v0.37.0
	github.com/vdaas/vald-client-go v1.7.17
	google.golang.org/grpc v1.73.0
	google.golang.org/protobuf v1.36.6
	gopkg.in/yaml.v3 v3.0.1
)

require github.com/google/go-querystring v1.1.0 // indirect

require (
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.36.6-20250625184727-c923a0c2a132.1 // indirect
	github.com/Azure/go-ansiterm v0.0.0-20250102033503-faa5f7b0171c // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/containerd/log v0.1.0 // indirect
	github.com/containerd/platforms v0.2.1 // indirect
	github.com/cpuguy83/dockercfg v0.3.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/docker/docker v28.0.4+incompatible // indirect
	github.com/docker/go-connections v0.5.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/ebitengine/purego v0.8.2 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-jwt/jwt/v5 v5.2.2 // indirect
	github.com/google/go-github/v58 v58.0.0
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/lufia/plan9stats v0.0.0-20250317134145-8bc96cf8fc35 // indirect
	github.com/magiconair/properties v1.8.10 // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/moby/patternmatcher v0.6.0 // indirect
	github.com/moby/sys/sequential v0.6.0 // indirect
	github.com/moby/sys/user v0.4.0 // indirect
	github.com/moby/sys/userns v0.1.0 // indirect
	github.com/moby/term v0.5.2 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/nats-io/nats.go v1.41.1 // indirect
	github.com/nats-io/nkeys v0.4.9 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/planetscale/vtprotobuf v0.6.1-0.20240319094008-0393e58bdf10 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/power-devops/perfstat v0.0.0-20240221224432-82ca36839d55 // indirect
	github.com/shirou/gopsutil/v4 v4.25.3 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/testcontainers/testcontainers-go/modules/nats v0.36.0 // indirect
	github.com/testcontainers/testcontainers-go/modules/postgres v0.36.0 // indirect
	github.com/testcontainers/testcontainers-go/modules/valkey v0.36.0 // indirect
	github.com/tklauser/go-sysconf v0.3.15 // indirect
	github.com/tklauser/numcpus v0.10.0 // indirect
	github.com/valkey-io/valkey-go v1.0.62 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.60.0 // indirect
	go.opentelemetry.io/otel v1.35.0 // indirect
	go.opentelemetry.io/otel/metric v1.35.0 // indirect
	go.opentelemetry.io/otel/trace v1.35.0 // indirect
	golang.org/x/crypto v0.37.0 // indirect
	golang.org/x/net v0.38.0 // indirect
	golang.org/x/oauth2 v0.30.0
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.26.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20250324211829-b45e905df463 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250528174236-200df99c418a // indirect
)
