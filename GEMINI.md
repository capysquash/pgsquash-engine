# Project: pgsquash - PostgreSQL Migration Squasher

## Project Overview

`pgsquash` is a command-line tool written in Go that consolidates and squashes multiple PostgreSQL migration files into a single, clean, and optimized migration. It is designed to preserve data integrity and dependency order while reducing the number of migration files. The tool features an interactive terminal user interface (TUI), accurate SQL parsing using `pg_query_go`, and various safety levels to control the optimization process. It also includes a Docker-based schema validation feature to ensure that the consolidated migration is equivalent to the original migrations. Additionally, it has optional AI-powered semantic analysis for function deduplication and dead code detection.

The project is structured as a Go CLI application with a clear separation of concerns. The `cmd/pgsquash` directory contains the entry point of the application, while the `internal` directory holds the core logic for parsing, squashing, and validating migrations. The project also includes a `docker` directory with all the necessary files to build and run the application in a containerized environment.

## Building and Running

### Building from source

To build the `pgsquash` binary from source, you can use the following command:

```bash
go build -o pgsquash cmd/pgsquash/main.go
```

### Running with Docker

The project includes a `docker-compose.yml` file that sets up a development environment with the `pgsquash` application and a PostgreSQL database. To run the application with Docker, you can use the following command:

```bash
docker-compose up
```

This will build the `pgsquash` image and start the `pgsquash` and `postgres-primary` services. The `pgsquash` service will have access to the migration files in the `migrations` directory.

### Testing

To run the tests, you can use the following command:

```bash
go test ./...
```

## Development Conventions

The project follows standard Go development conventions. The code is organized into packages within the `internal` directory, and the entry point of the application is in the `cmd/pgsquash` directory. The project uses Go modules for dependency management.

The project also makes extensive use of Docker for development and testing. The `docker-compose.yml` file defines the development environment, and the `Dockerfile` is used to build the application image. The validation feature also relies on Docker to create a temporary PostgreSQL container to test the consolidated migration.
