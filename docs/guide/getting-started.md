# Getting Started

This guide will walk you through setting up Rainlogs locally using Docker Compose.

## Prerequisites

- [Docker](https://docs.docker.com/get-docker/)
- [Docker Compose](https://docs.docker.com/compose/install/)
- [Go 1.24+](https://go.dev/dl/) (if building from source)

## Quick Start

The easiest way to get Rainlogs running is via the provided `docker-compose.yml`.

1. **Clone the repository:**

   ```bash
   git clone https://github.com/your-org/rainlogs.git
   cd rainlogs
   ```

2. **Start the infrastructure:**

   ```bash
   docker compose up -d
   ```

   This command will start:
   - **PostgreSQL**: The primary database.
   - **Redis**: Used by Asynq for background job queues.
   - **Garage S3**: The distributed object storage.
   - **Asynqmon**: The web UI for monitoring background jobs.
   - **Rainlogs API**: The core Go application.

3. **Verify the installation:**

   The Rainlogs API should now be running on `http://localhost:8080`.
   You can check the health endpoint:

   ```bash
   curl http://localhost:8080/health
   ```

   You should see a response indicating the service is healthy.

## Zero-Config Initialization

Rainlogs features a "Zero-Config" initialization process. When the API starts, it automatically:

1. Connects to the Garage S3 instance.
2. Creates the necessary S3 buckets (e.g., `rainlogs-data`).
3. Generates S3 access keys if they don't exist.
4. Configures the application to use these keys seamlessly.

You do not need to manually configure S3 credentials or create buckets before starting the application.

## Next Steps

- Read about the [Architecture](./architecture.md) to understand how the components interact.
- Check the [Configuration](./configuration.md) guide to customize your setup.
