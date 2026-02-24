# Storage

Rainlogs uses [Garage S3](https://garagehq.deuxfleurs.fr/) as its primary storage backend for logs. Garage is an open-source, distributed object storage system that implements the S3 API. It is designed for self-hosting and provides a sovereign alternative to AWS S3.

## Why Garage S3?

- **Sovereignty**: By using Garage S3, data remains within your control, ensuring compliance with data sovereignty regulations.
- **Distributed**: Garage is designed to be distributed across multiple nodes, providing high availability and fault tolerance.
- **S3 Compatible**: Garage implements the S3 API, making it compatible with existing tools and libraries.
- **Zero-Config**: Rainlogs features a "Zero-Config" initialization process that automatically connects to the Garage S3 instance, creates the necessary buckets, and generates access keys.

## Configuration

The storage configuration is managed via environment variables or a `config.yaml` file. See the [Configuration](./configuration.md) guide for more details.

### Zero-Config Initialization

If the `RAINLOGS_STORAGE_ACCESS_KEY` and `RAINLOGS_STORAGE_SECRET_KEY` are not provided, Rainlogs will attempt to automatically generate them by connecting to the Garage S3 instance and creating the necessary buckets. This is the recommended approach for local development and testing.

## Data Retention

Rainlogs enforces data retention policies (e.g., GDPR Art. 17, NIS2 requirements) by periodically scanning the S3 buckets and deleting logs that exceed the configured retention period. This is handled by background workers powered by [Asynq](https://github.com/hibiken/asynq).

## Multi-provider Failover

Rainlogs supports S3 failover (e.g., Contabo + Hetzner) to ensure high availability and data durability. This is achieved by configuring multiple S3 endpoints and automatically switching to a secondary endpoint if the primary one becomes unavailable.
