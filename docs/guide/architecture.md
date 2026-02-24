# Architecture

Rainlogs is built on a modern, robust stack designed for high performance, security, and compliance.

## Core Components

### 1. Go 1.24 Backend

The core API and background workers are written in Go 1.24. It leverages the standard library's `net/http` package with enhanced routing capabilities introduced in Go 1.22.

### 2. PostgreSQL

The primary relational database used for storing metadata, user accounts, and configuration.

### 3. Redis & Asynq

Redis is used as the backing store for [Asynq](https://github.com/hibiken/asynq), a robust task queue library for Go. Asynq handles background jobs such as:
- Fetching logs from Cloudflare.
- Processing and encrypting logs.
- Uploading logs to S3.
- Enforcing data retention policies (e.g., deleting logs older than 395 days).

### 4. Garage S3

[Garage](https://garagehq.deuxfleurs.fr/) is an open-source, distributed object storage system that implements the S3 API. It is designed for self-hosting and provides a sovereign alternative to AWS S3. Rainlogs uses Garage as its primary storage backend for logs.

### 5. Asynqmon

A web UI for monitoring and managing Asynq queues and tasks. It provides real-time visibility into the background processing pipeline.

## Data Flow

1. **Ingestion**: Logs are fetched from external sources (e.g., Cloudflare) via background jobs.
2. **Processing**: The logs are processed, and sensitive data is encrypted using AES-256-GCM.
3. **Storage**: The processed logs are uploaded to the Garage S3 cluster.
4. **Retention**: Background workers periodically scan the S3 buckets and delete logs that exceed the configured retention period to ensure compliance (e.g., GDPR, NIS2).

## Security

- **Encryption at Rest**: Sensitive data, such as API keys, are encrypted before being stored in the database.
- **KMS**: A built-in Key Management Service (KMS) handles encryption and decryption operations.
- **Sovereignty**: By using Garage S3, data remains within your control, ensuring compliance with data sovereignty regulations.
