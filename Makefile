GO = $(shell which go 2>/dev/null)
DOCKER = $(shell which docker 2>/dev/null)

POSTGRES_SERVER_APP := postgres-only
POSTGRES_REDIS_SERVER_APP := postgres-redis
CLIENT_APP := client
VERSION := v0.1.0
LDFLAGS := -ldflags "-X main.AppVersion=$(VERSION)"

# Docker image names
DOCKER_REGISTRY := localhost
POSTGRES_IMAGE := $(DOCKER_REGISTRY)/$(POSTGRES_SERVER_APP)
POSTGRES_REDIS_IMAGE := $(DOCKER_REGISTRY)/$(POSTGRES_REDIS_SERVER_APP)

.PHONY: all build clean run test proto docker docker-postgres docker-postgres-redis

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

# Docker build commands
docker: docker-postgres docker-postgres-redis

docker-postgres:
	$(DOCKER) build \
		--build-arg VERSION=$(VERSION) \
		--build-arg APP_NAME=$(POSTGRES_SERVER_APP) \
		-t $(POSTGRES_IMAGE):$(VERSION) \
		-t $(POSTGRES_IMAGE):latest \
		.

docker-postgres-redis:
	$(DOCKER) build \
		--build-arg VERSION=$(VERSION) \
		--build-arg APP_NAME=$(POSTGRES_REDIS_SERVER_APP) \
		-t $(POSTGRES_REDIS_IMAGE):$(VERSION) \
		-t $(POSTGRES_REDIS_IMAGE):latest \
		.
