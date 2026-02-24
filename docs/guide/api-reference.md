# API Reference

The Rainlogs API is built with Go 1.24 and leverages the standard library's `net/http` package with enhanced routing capabilities introduced in Go 1.22.

## Endpoints

### Health Check

- **GET** `/health`
  - Returns a 200 OK response indicating the service is healthy.

### Logs

- **POST** `/logs`
  - Ingests logs from external sources (e.g., Cloudflare).
  - Requires authentication.
  - Payload: JSON array of log entries.

- **GET** `/logs`
  - Retrieves logs based on query parameters (e.g., time range, source).
  - Requires authentication.
  - Query Parameters:
    - `start`: Start time (ISO 8601 format).
    - `end`: End time (ISO 8601 format).
    - `source`: Log source (e.g., `cloudflare`).

### Configuration

- **GET** `/config`
  - Retrieves the current configuration settings.
  - Requires authentication.

- **PUT** `/config`
  - Updates the configuration settings.
  - Requires authentication.
  - Payload: JSON object containing the updated configuration.

## Authentication

The API uses Bearer token authentication. You must include the token in the `Authorization` header of your requests.

```http
Authorization: Bearer your-token
```

## Error Handling

The API returns standard HTTP status codes to indicate success or failure.

- **200 OK**: The request was successful.
- **400 Bad Request**: The request was invalid or malformed.
- **401 Unauthorized**: The request requires authentication.
- **403 Forbidden**: The authenticated user does not have permission to access the resource.
- **404 Not Found**: The requested resource was not found.
- **500 Internal Server Error**: An unexpected error occurred on the server.

Error responses include a JSON payload with a `message` field describing the error.

```json
{
  "message": "Invalid request payload"
}
```
