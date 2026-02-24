---
layout: home

hero:
  name: "RainLogs"
  text: "Cloudflare Log Archiving & Compliance"
  tagline: "High-performance, EU-sovereign log retention for NIS2 requirements."
  image:
    src: /logo.svg
    alt: RainLogs Logo
  actions:
    - theme: brand
      text: Documentation
      link: /guide/getting-started
    - theme: alt
      text: GitHub
      link: https://github.com/fabriziosalmi/rainlogs

features:
  - title: Infrastructure
    details: Complete containerized stack including API, Worker, PostgreSQL, Redis, and Garage S3.
  - title: Storage Sovereignty
    details: Native integration with S3-compatible providers (Garage, Hetzner) strictly within EU jurisdiction.
  - title: Encryption
    details: AES-256-GCM encryption for credentials at rest with centralized key management.
  - title: Verification
    details: Cryptographically verifiable log integrity using SHA-256 hash chaining (WORM).
  - title: GDPR Compliance
    details: Automated lifecycle management enforcing strict data retention policies.
  - title: Architecture
    details: Built with Go 1.24, employing Hexagonal Architecture and CQRS for maintainability.
  - title: Observability
    details: Comprehensive metrics and structured logging via Prometheus/OpenTelemetry standards.
---
