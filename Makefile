GOPATH ?= $(shell go env GOPATH)
ifneq ($(OS),Windows_NT)
EXE =
else
EXE = .exe
endif
PKG = $(shell go env GOOS)_$(shell go env GOARCH)
TAGS ?=
BBLFSH_DEP =

all: ${GOPATH}/bin/hercules${EXE}

test: all
	go test gopkg.in/src-d/hercules.v7

${GOPATH}/bin/protoc-gen-gogo${EXE}:
	go get -v github.com/gogo/protobuf/protoc-gen-gogo

ifneq ($(OS),Windows_NT)
internal/pb/pb.pb.go: internal/pb/pb.proto ${GOPATH}/bin/protoc-gen-gogo
	PATH=${PATH}:${GOPATH}/bin protoc --gogo_out=internal/pb --proto_path=internal/pb internal/pb/pb.proto
else
internal/pb/pb.pb.go: internal/pb/pb.proto ${GOPATH}/bin/protoc-gen-gogo.exe
	export PATH="${PATH};${GOPATH}\bin" && \
	protoc --gogo_out=internal/pb --proto_path=internal/pb internal/pb/pb.proto
endif

internal/pb/pb_pb2.py: internal/pb/pb.proto
	protoc --python_out internal/pb --proto_path=internal/pb internal/pb/pb.proto

cmd/hercules/plugin_template_source.go: cmd/hercules/plugin.template
	cd cmd/hercules && go generate

vendor:
	dep ensure -v

ifeq ($(OS),Windows_NT)
BBLFSH_DEP = vendor/gopkg.in/bblfsh/client-go.v2/tools/include

vendor/gopkg.in/bblfsh/client-go.v2/tools/include:
	cd vendor/gopkg.in/bblfsh/client-go.v2 && make cgo-dependencies
endif

${GOPATH}/bin/hercules${EXE}: vendor *.go */*.go */*/*.go */*/*/*.go internal/pb/pb.pb.go internal/pb/pb_pb2.py cmd/hercules/plugin_template_source.go ${BBLFSH_DEP}
	go get -tags "$(TAGS)" -ldflags "-X gopkg.in/src-d/hercules.v7.BinaryGitHash=$(shell git rev-parse HEAD)" gopkg.in/src-d/hercules.v7/cmd/hercules
