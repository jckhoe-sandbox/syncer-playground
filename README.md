# Syncer Playground

A demonstration of bidirectional data synchronization using gRPC, PostgreSQL, and Redis.

## Project Structure

- `cmd/postgres-only/`: Server implementation using only PostgreSQL
- `cmd/postgres-redis/`: Server implementation using PostgreSQL and Redis for event synchronization
- `cmd/client/`: Test client application
- `proto/`: Protocol buffer definitions
- `pkg/chat/`: Generated protocol buffer code

## Prerequisites

- Go 1.23 or later
- PostgreSQL
- Redis (for the postgres-redis version)
- Protocol Buffers compiler (protoc)

## Running the Application

1. Start PostgreSQL:
```bash
docker run -d --name postgres -e POSTGRES_PASSWORD=postgres -p 5432:5432 postgres
```

2. Start Redis (for postgres-redis version):
```bash
docker run -d --name redis -p 6379:6379 redis
```

3. Build the project:
```bash
make build
```

4. Run either server version:
```bash
# PostgreSQL-only version
make run-postgres

# PostgreSQL + Redis version
make run-postgres-redis
```

5. Run the client:
```bash
make run-client
```

## Features

- Bidirectional streaming using gRPC
- PostgreSQL database integration
- Redis event synchronization (postgres-redis version)
- Automatic schema migration
