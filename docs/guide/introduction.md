# Introduction

**Rainlogs** is an enterprise-grade log management system engineered for security, data sovereignty, and regulatory compliance. Built with Go 1.24, it integrates seamlessly with modern infrastructure stacks including PostgreSQL, Redis, and S3-compatible object storage.

## Core Principles

1. **Automated Provisioning**: Deploys as a complete containerized stack, including distributed S3 storage, via standard orchestration tools.
2. **Data Sovereignty**: Ensures strict data residency by utilizing EU-based storage providers (e.g., Garage, Hetzner, Contabo), avoiding US Cloud Act exposure.
3. **Defense in Depth**: Protecting credentials with AES-256-GCM encryption at rest and enforcing strict Role-Based Access Control (RBAC).
4. **Regulatory Compliance**: Automated retention policies and tamper-evident logging meet stringent requirements such as NIS2 (Article 21) and GDPR.

## Capabilities

- **Automated Ingestion**: Adapts collection strategies to Cloudflare plan capabilities (Logpull, Instant Logs, Security Events).
- **Storage Agnostic**: Native support for any S3-compatible backend, with built-in multi-provider failover.
- **Bulk Export**: High-throughput export pipeline for moving large datasets to cold storage.
- **Data Integrity**: Cryptographic verification using SHA-256 hash chaining (WORM) for forensic auditability.
- **Access Control**: Granular permission management separating administrative and auditor roles.
- **Resilience**: Robust job processing with circuit breakers, exponential backoff, and dead-letter queues.
- **Observability**: Prometheus metrics and OpenTelemetry-compatible structured logging.

## Rationale

Organizations operating under strict regulatory frameworks (NIS2, D.Lgs. 138/2024) require log retention periods that often exceed standard provider limits (e.g., Cloudflare's 7-day Logpull retention). Rainlogs provides a compliant, self-hosted archive solution that guarantees data integrity and sovereignty while eliminating the complexity of building custom ingestion pipelines.
