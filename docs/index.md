---
layout: home

hero:
  name: "Rainlogs"
  text: "High-performance log management"
  tagline: "Self-hosted, zero-config, EU-sovereign log storage and analysis."
  image:
    src: /logo.svg
    alt: Rainlogs Logo
  actions:
    - theme: brand
      text: Get Started
      link: /guide/getting-started
    - theme: alt
      text: View on GitHub
      link: https://github.com/fabriziosalmi/rainlogs

features:
  - title: Zero-Config Onboarding
    details: Spin up the entire stack (API, Worker, PostgreSQL, Redis, Garage S3) with a single `docker compose up -d`.
  - title: EU-Sovereign Storage
    details: Built-in integration with Garage S3 for distributed, sovereign object storage.
  - title: Secure by Design
    details: AES-256-GCM encryption at rest via KMS (with key rotation) for sensitive data.
  - title: Tamper-Proof
    details: SHA-256 WORM hash chaining with `rainlogs-verify` tool for integrity audits.
  - title: GDPR Compliant
    details: Automated log expiry workers enforce retention policies (e.g., 395 days for NIS2).
  - title: High Performance
    details: Written in Go 1.24, utilizing Asynq for robust background job processing.
  - title: Real-time Monitoring
    details: Includes Asynqmon for real-time queue monitoring and task management.
---
