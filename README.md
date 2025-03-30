# Syncer Playground

A demonstration of bidirectional data synchronization using gRPC, PostgreSQL, and Redis. Also to demonstrate how bad AI is at these challenging technical implementation, to then raise a PR to fix it all.

## State

- Stop project canot be proceeded, AI not able to fix this repo or even get it remotely working after the tagged `vibev1`, loop steps and failures.

## Project Structure

- `cmd/postgres-only/`: Server implementation using only PostgreSQL
- `cmd/postgres-redis/`: Server implementation using PostgreSQL and Redis for event synchronization
- `cmd/client/`: Test client application
- `proto/`: Protocol buffer definitions
- `pkg/chat/`: Generated protocol buffer code
- `pkg/config/`: Configuration management package
- `misc/`: Docker Compose and deployment configurations

## Prerequisites

- Go 1.23 or later
- PostgreSQL
- Redis (for the postgres-redis version)
- Protocol Buffers compiler (protoc)
- Docker and Docker Compose (optional, for containerized deployment)

## Configuration

The application can be configured using environment variables or a `.env` file. All configuration options are prefixed with `SYNCER_`.

### Environment Variables

```bash
# PostgreSQL Configuration
SYNCER_POSTGRES_HOST=localhost
SYNCER_POSTGRES_PORT=5432
SYNCER_POSTGRES_USER=postgres
SYNCER_POSTGRES_PASSWORD=postgres
SYNCER_POSTGRES_DBNAME=chat
SYNCER_POSTGRES_SSLMODE=disable

# Redis Configuration (for postgres-redis version)
SYNCER_REDIS_HOST=localhost
SYNCER_REDIS_PORT=6379
SYNCER_REDIS_PASSWORD=
SYNCER_REDIS_DB=0

# Server Configuration
SYNCER_SERVER_PORT=50051
```

Copy `.env.example` to `.env` and modify the values as needed:

```bash
cp .env.example .env
```

## Running the Application

### Local Development

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

### Docker Deployment

#### Using Docker Compose (Recommended)

1. Navigate to the misc directory:

```bash
cd misc
```

2. Copy the environment file:

```bash
cp .env.example .env
```

3. Start all services:

```bash
docker compose up -d
```

This will start:

- PostgreSQL-only server with its own database on port 50051
- PostgreSQL + Redis server with its own database and Redis on port 50052
- A test client that can connect to both servers

4. View logs:

```bash
docker compose logs -f
```

5. Stop all services:

```bash
docker compose down
```

#### Using Individual Docker Commands

1. Build Docker images:

```bash
# Build both server images
make docker

# Or build individual images
make docker-postgres
make docker-postgres-redis
```

2. Run the containers:

```bash
# PostgreSQL-only version
docker run -d \
  --name postgres-only \
  -p 50051:50051 \
  -e SYNCER_POSTGRES_HOST=postgres \
  -e SYNCER_POSTGRES_USER=postgres \
  -e SYNCER_POSTGRES_PASSWORD=postgres \
  --network syncer-network \
  localhost/postgres-only:latest

# PostgreSQL + Redis version
docker run -d \
  --name postgres-redis \
  -p 50052:50051 \
  -e SYNCER_POSTGRES_HOST=postgres \
  -e SYNCER_POSTGRES_USER=postgres \
  -e SYNCER_POSTGRES_PASSWORD=postgres \
  -e SYNCER_REDIS_HOST=redis \
  -e SYNCER_REDIS_PORT=6379 \
  --network syncer-network \
  localhost/postgres-redis:latest
```

## Features

- Bidirectional streaming using gRPC
- PostgreSQL database integration
- Redis event synchronization (postgres-redis version)
- Automatic schema migration
- Docker support for containerized deployment
- Modern configuration management with environment variables and .env file support
- Docker Compose setup with separate infrastructure for each server version
