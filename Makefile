VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "0.1.0")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-X main.appVersion=$(VERSION) -X main.buildTime=$(BUILD_TIME)"

.PHONY: all build clean proto test

all: proto build

build:
	@echo "Building dropz version $(VERSION)..."
	@mkdir -p bin
	go build $(LDFLAGS) -o bin/dropz cmd/dropz/main.go

proto:
	@echo "Generating protobuf code..."
	@mkdir -p pkg/protocol
	go get -u google.golang.org/protobuf/cmd/protoc-gen-go
	go get -u google.golang.org/grpc/cmd/protoc-gen-go-grpc
	protoc -I=proto -I=include -I=/usr/local/include -I=/usr/include \
        --go_out=pkg/protocol/ --go_opt=paths=source_relative \
		--go-grpc_out=pkg/protocol/ --go-grpc_opt=paths=source_relative \
		--go_opt=Mservice.proto=github.com/dropz/dropz/pkg/protocol \
		--go_opt=Mgopro.proto=github.com/dropz/dropz/pkg/protocol \
		--go_opt=Mconfig.proto=github.com/dropz/dropz/pkg/protocol \
		--go_opt=Mlogs.proto=github.com/dropz/dropz/pkg/protocol \
		--go_opt=Mvideo.proto=github.com/dropz/dropz/pkg/protocol \
		--go_opt=Mcommon.proto=github.com/dropz/dropz/pkg/protocol \
		--go-grpc_opt=Mservice.proto=github.com/dropz/dropz/pkg/protocol \
		--go-grpc_opt=Mgopro.proto=github.com/dropz/dropz/pkg/protocol \
		--go-grpc_opt=Mconfig.proto=github.com/dropz/dropz/pkg/protocol \
		--go-grpc_opt=Mlogs.proto=github.com/dropz/dropz/pkg/protocol \
		--go-grpc_opt=Mvideo.proto=github.com/dropz/dropz/pkg/protocol \
		--go-grpc_opt=Mcommon.proto=github.com/dropz/dropz/pkg/protocol \
		proto/*.proto
	@if [ -f proto/*_grpc.pb.go ] || [ -f proto/*.pb.go ]; then \
		mv proto/*.pb.go pkg/protocol/ 2>/dev/null || true; \
		mv proto/*_grpc.pb.go pkg/protocol/ 2>/dev/null || true; \
	fi
	@rm -rf github.com
	@echo "Generating JavaScript protobuf code..."
	cd frontend && npx grpc_tools_node_protoc \
		--js_out=import_style=commonjs:./src/proto \
		--grpc_out=grpc_js:./src/proto \
		--proto_path=../proto \
		../proto/*.proto
	@echo "Protobuf code generation complete."

test:
	@echo "Running tests..."
	go test -v ./...

clean:
	@echo "Cleaning up..."
	rm -rf bin/
	rm -f pkg/protocol/*.pb.go
	rm -f frontend/src/proto/*_pb.js
	rm -f frontend/src/proto/*_grpc_pb.js

run:
	@echo "Running dropz..."
	./bin/dropz

deps:
	@echo "Installing dependencies..."
	go get -u google.golang.org/protobuf/cmd/protoc-gen-go
	go get -u google.golang.org/grpc/cmd/protoc-gen-go-grpc
	go mod tidy 