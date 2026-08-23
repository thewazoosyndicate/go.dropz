VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "0.1.0")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-X main.appVersion=$(VERSION) -X main.buildTime=$(BUILD_TIME)"

.PHONY: all build build-darwin-universal clean proto test appimage dmg app frontend-deps frontend-build

all: proto build

build:
	@echo "Building dropz version $(VERSION)..."
	@mkdir -p bin
	go build $(LDFLAGS) -o bin/dropz cmd/dropz/main.go

proto:
	@echo "Generating protobuf code..."
	@mkdir -p internal/protocol
	go get -u google.golang.org/protobuf/cmd/protoc-gen-go
	go get -u google.golang.org/grpc/cmd/protoc-gen-go-grpc
	protoc -I=proto -I=include -I=/usr/local/include -I=/usr/include \
        --go_out=internal/protocol/ --go_opt=paths=source_relative \
		--go-grpc_out=internal/protocol/ --go-grpc_opt=paths=source_relative \
		--go_opt=Mservice.proto=github.com/dropz/dropz/internal/protocol \
		--go_opt=Mgopro.proto=github.com/dropz/dropz/internal/protocol \
		--go_opt=Mconfig.proto=github.com/dropz/dropz/internal/protocol \
		--go_opt=Mvideo.proto=github.com/dropz/dropz/internal/protocol \
		--go_opt=Mcommon.proto=github.com/dropz/dropz/internal/protocol \
		--go-grpc_opt=Mservice.proto=github.com/dropz/dropz/internal/protocol \
		--go-grpc_opt=Mgopro.proto=github.com/dropz/dropz/internal/protocol \
		--go-grpc_opt=Mconfig.proto=github.com/dropz/dropz/internal/protocol \
		--go-grpc_opt=Mvideo.proto=github.com/dropz/dropz/internal/protocol \
		--go-grpc_opt=Mcommon.proto=github.com/dropz/dropz/internal/protocol \
		proto/*.proto
	@if [ -f proto/*_grpc.pb.go ] || [ -f proto/*.pb.go ]; then \
		mv proto/*.pb.go internal/protocol/ 2>/dev/null || true; \
		mv proto/*_grpc.pb.go internal/protocol/ 2>/dev/null || true; \
	fi
	@rm -rf github.com
	@echo "Generating JavaScript protobuf code..."
	cd frontend && npx grpc_tools_node_protoc \
		--js_out=import_style=commonjs:./src/proto \
		--grpc_out=grpc_js:./src/proto \
		--proto_path=../proto \
		../proto/*.proto
	@echo "Protobuf code generation complete."

build-darwin-universal:
	@echo "Building universal macOS binary..."
	@mkdir -p bin
	CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/dropz-amd64 cmd/dropz/main.go
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bin/dropz-arm64 cmd/dropz/main.go
	lipo -create bin/dropz-amd64 bin/dropz-arm64 -output bin/dropz
	@rm bin/dropz-amd64 bin/dropz-arm64

appimage: build frontend-build
	@echo "Building AppImage..."
	cd frontend && npx electron-builder --linux AppImage
	@echo "AppImage built: frontend/dist/Dropz-1.0.0.AppImage"

dmg: build-darwin-universal frontend-build
	@echo "Building macOS DMG..."
	cd frontend && npx electron-builder --mac dmg
	@echo "DMG built in frontend/dist/"

app: build frontend-build
	@echo "Building desktop app..."
	cd frontend && npx electron-builder

frontend-deps:
	cd frontend && npm install

frontend-build:
	@echo "Building Svelte frontend..."
	cd frontend && npx vite build

test:
	@echo "Running tests..."
	go test -v ./...

clean:
	@echo "Cleaning up..."
	rm -rf bin/
	rm -f internal/protocol/*.pb.go
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
