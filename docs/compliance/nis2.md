# NIS2 Compliance & Rainlogs

Rainlogs is designed to support organizational compliance with **Directive (EU) 2022/2555** (NIS2 Directive), specifically regarding incident handling, supply chain security, and data integrity.

## Key Directive Articles

The directive mandates specific technical and organizational measures. Rainlogs directly addresses the following requirements:

### Article 21: Cybersecurity Risk-Management Measures

**Requirement:**
> "...policies and procedures to assess the effectiveness of cybersecurity risk-management measures." (Art 21.2.d)

**Rainlogs Capability:**
*   **Immutable Audit Trails:** Every access to logs and configuration changes is recorded in a tamper-proof audit log (`internal/api/middleware/audit.go`).
*   **WORM Storage:** Logs are stored using Write-Once-Read-Many (WORM) technology, ensuring forensic data cannot be altered or deleted before the retention period expires (`pkg/worm/worm.go`).

**Requirement:**
> "...incident handling." (Art 21.2.a)

**Rainlogs Capability:**
*   **Real-time Ingestion:** Logs are pulled from Cloudflare in near real-time, allowing for rapid detection of anomalies.
*   **Centralized Repository:** Aggregates logs from multiple zones into a single, queryable source of truth for incident response teams.

### Article 23: Reporting Obligations

**Requirement:**
> "Entities shall notify, without undue delay... any significant incident." (Art 23.1)

**Rainlogs Capability:**
*   **Alerting Framework:** (Planned) Integration with Slack/Webhook to notify security teams immediately upon detecting critical log patterns or system anomalies.
*   **Availability Monitoring:** Built-in health checks (`/health`) and Prometheus metrics (`/metrics`) ensure the logging infrastructure itself is operational, preventing "silent failures" in monitoring.

## Official Resources

*   **Official Text (EUR-Lex):** [Directive (EU) 2022/2555 (NIS2)](https://eur-lex.europa.eu/eli/dir/2022/2555/oj)
*   **ENISA (European Union Agency for Cybersecurity):** [NIS2 Directive Overview](https://www.enisa.europa.eu/topics/cybersecurity-policy/nis-directive-new)
*   **National Transposition Status:** Consult your local national cybersecurity authority (e.g., BSI in Germany, ANSSI in France) for specific implementation laws.

## Data Sovereignty

Rainlogs is self-hosted software. You retain full control over:
1.  **Storage Location:** Metadata resides in your PostgreSQL database; heavy log data resides in your S3/compatible storage.
2.  **Encryption:** API keys and sensitive customer data are encrypted at rest using AES-GCM (`internal/kms/kms.go`).
3.  **Access Control:** Role-Based Access Control (RBAC) ensures only authorized personnel can access sensitive log data.
