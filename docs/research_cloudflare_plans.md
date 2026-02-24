# Cloudflare Data Access Research & Technical Proposal

## 1. Research on Cloudflare Data Access Options

### Instant Logs
*   **Availability:**
    *   **Free:** No
    *   **Pro:** No
    *   **Business:** **Yes** (Confirmed via documentation)
    *   **Enterprise:** Yes
*   **Protocol:** WebSocket (`wss://logs.cloudflare.com/instant-logs/ws/sessions/<SESSION_ID>`)
*   **Sampling:**
    *   Yes, server-side sampling is enforced for high-volume domains.
    *   The `sampleInterval` field in the log JSON indicates the sampling rate applied (e.g., `10` means 1 in 10 requests).
*   **Method:** Requires creating a Job via API to get a session ID, then connecting via WebSocket.

### GraphQL Analytics API
*   **Datasets Availability (Non-Enterprise):**
    *   **Free/Pro/Business:** Access is generally available for **Zone Analytics** (e.g., `httpRequests1mGroups`) and **Firewall Analytics** (e.g., `firewallEventsAdaptiveGroups`).
    *   **Data Granularity:** These are *aggregated* datasets. You query for counts, sums, or groupings (e.g., "count of requests per country").
    *   **Security Logs (Blocked IPs):**
        *   **Firewall Events:** You *can* query `firewallEventsAdaptiveGroups` and group by `clientIP` (subject to cardinality limits). This allows extracting a list of IPs that triggered firewall rules (e.g., `action: "block"`).
        *   **HTTP Requests:** Grouping by `clientIP` is typically restricted or highly sampled for non-Enterprise to prevent high-cardinality queries.
*   **Meaningful Security Logs:** Yes, specific to *blocked/challenged* requests via Firewall/Security analytics.

### Audit Logs
*   **Availability:**
    *   **Dashboard:** Viewable for all plans (retention varies: typically 18 months for Ent, less for others).
    *   **API Access:**
        *   **Enterprise:** Full access via API and Logpush.
        *   **Free/Pro/Business:** API access to `/accounts/<id>/audit_logs` is generally **available** but limited by rate limits and retention (often 12 months). However, automated export (Logpush) is Enterprise-only.

## 2. NIS2 Compliance Comparison

**Requirement:** "Who accessed what, when" (Detailed Access Logs).

| Option | "Who" (Client IP/User) | "What" (Resource) | "When" (Timestamp) | Verdict for NIS2 |
| :--- | :--- | :--- | :--- | :--- |
| **GraphQL Analytics** | **Partial** (Aggregated). Can see "top IPs" or "blocked IPs" but not *every* IP for every request. | **Partial**. Aggregated by path/host. | **No**. Time buckets (e.g., 1 min), not precise timestamps. | **Insufficient**. Cannot reconstruct an incident timeline or prove specific access for a specific request. |
| **Instant Logs** | **Yes**. Full request details (IP, User Agent). | **Yes**. Full URI, Method. | **Yes**. Precise timestamp. | **Viable but Risky**. Provides the necessary data granularity. However, it is a *stream* (WebSocket). If the connection drops or the buffer overflows, data is **lost forever**. It lacks the "guaranteed delivery" of Logpush. |

**Conclusion:**
*   **GraphQL** is useful for *trends* and *identifying attackers* (blocked IPs), but fails "incident forensics" for successful access (e.g., "did IP X access file Y at time Z?").
*   **Instant Logs** is the *only* viable option for detailed logging on Business plans, but requires a robust collector to minimize data loss. It is not "audit-grade" reliable like Logpush (S3 storage), but it's the best available for non-Enterprise.

## 3. Technical Proposal for `rainlogs`

To support Business (and potentially Pro) users, `rainlogs` should implement a multi-tiered collection strategy:

### A. New Collector: "Instant Logs Streamer" (Business Plan)
*   **Target:** Business Plan Users.
*   **Mechanism:**
    1.  **Job Creation:** Uses Cloudflare API to create a temporary Instant Logs job.
    2.  **WebSocket Client:** Connects to the returned `wss://` URL.
    3.  **Keep-Alive:** Handles network interruptions and automatically creates new jobs (sessions expire after 60m or 5m inactivity).
    4.  **Buffering:** Buffers incoming JSON lines and batches them for writing to the configured storage (S3/Filesystem), mimicking the Logpush file format.
*   **Benefit:** Provides near-real-time access logs for Business users who cannot use Logpush.

### B. New Collector: "Security Events Poller" (Pro/Business Plan)
*   **Target:** Pro and Business Users (where Instant Logs might be overkill or unavailable/unwanted).
*   **Mechanism:**
    *   Periodically queries **GraphQL API** (`firewallEventsAdaptiveGroups`).
    *   Query: Group by `clientIP`, `action`, `ruleId` for the last X minutes.
    *   Result: Generates a "Security Audit" log file containing the list of blocked/challenged IPs and reasons.
*   **Benefit:** Provides "Who was blocked and why" visibility, which satisfies the security monitoring aspect of NIS2, even if full access logs are missing.

### C. Architecture Updates
*   **Worker:** Update `internal/worker` to support multiple `Collector` types (currently only `LogPull`).
*   **Configuration:** Add `mode` to specific Cloudflare zone config (e.g., `mode: logpush` (default/current), `mode: instant_logs`, `mode: security_poll`).
