# What is Rainlogs?

**Rainlogs** is a high-performance, self-hosted log management system designed for security, compliance, and ease of use. It is built with Go 1.24 and leverages a modern, robust stack including PostgreSQL, Redis, and Garage S3.

## Core Philosophy

Rainlogs was built with a few core principles in mind:

1. **Zero-Config Onboarding**: The entire infrastructure, including distributed S3 storage, should spin up with a single command. No manual bucket creation, no complex key management.
2. **EU-Sovereignty**: By integrating [Garage S3](https://garagehq.deuxfleurs.fr/), Rainlogs provides a sovereign, distributed object storage solution out of the box.
3. **Security First**: Sensitive data, such as Cloudflare API keys, are encrypted at rest using AES-256-GCM via a built-in KMS service.
4. **Compliance Ready**: Automated log expiry workers ensure that data retention policies (e.g., GDPR Art. 17, NIS2 requirements) are strictly enforced.

## Key Features

- **Cloudflare Log Pulling**: Automatically fetch and store Cloudflare logs.
- **S3 Storage**: Store logs in any S3-compatible storage (defaults to the bundled Garage S3).
- **Bulk Export**: Efficiently move large log volumes to external S3 buckets.
- **WORM Compliance**: Cryptographic chain verification for tamper-evident data integrity.
- **RBAC**: Secure your deployment with Admin and Viewer roles.
- **Background Processing**: Robust job queues powered by [Asynq](https://github.com/hibiken/asynq).
- **Real-time Monitoring**: Built-in Asynqmon dashboard for queue visibility.
- **Multi-provider Failover**: Support for S3 failover (e.g., Contabo + Hetzner).

## Why Rainlogs?

If you need to retain logs for compliance (like the 395 days required by NIS2) but want to avoid the exorbitant costs of SaaS log management platforms, Rainlogs provides a self-hosted, highly efficient alternative that you fully control.
