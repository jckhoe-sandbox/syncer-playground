GO = $(shell which go 2>/dev/null)
DOCKER = $(shell which docker 2>/dev/null)

POSTGRES_SERVER_APP := postgres-only
POSTGRES_REDIS_SERVER_APP := postgres-redis
CLIENT_APP := client
VERSION := v0.1.0
LDFLAGS := -ldflags "-X main.AppVersion=$(VERSION)"

.PHONY: all build clean run test proto

all: clean build

clean:
	$(RM) -rf bin/*
	$(RM) -rf pkg/chat/*.pb.go

build: proto
	$(GO) build -o bin/postgres-only $(LDFLAGS) cmd/postgres-only/main.go
	$(GO) build -o bin/postgres-redis $(LDFLAGS) cmd/postgres-redis/main.go
	$(GO) build -o bin/client $(LDFLAGS) cmd/client/main.go

run-postgres:
	$(GO) run $(LDFLAGS) cmd/postgres-only/main.go

run-postgres-redis:
	$(GO) run $(LDFLAGS) cmd/postgres-redis/main.go

run-client:
	$(GO) run $(LDFLAGS) cmd/client/main.go

test:
	$(GO) test -v ./...

proto:
	@echo "Generating protobuf code..."
	@mkdir -p pkg/chat
	protoc \
		--proto_path=proto \
		--go_out=pkg/chat \
		--go_opt=paths=source_relative \
		--go-grpc_out=pkg/chat \
		--go-grpc_opt=paths=source_relative \
		proto/chat.proto
