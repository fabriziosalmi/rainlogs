# Configuration

Rainlogs uses [Viper](https://github.com/spf13/viper) for configuration management. It supports reading from environment variables, configuration files (e.g., `config.yaml`), and command-line flags.

## Environment Variables

The primary way to configure Rainlogs is through environment variables. The application expects variables to be prefixed with `RAINLOGS_`.

### Core Settings

| Variable | Description | Default |
|---|---|---|
| `RAINLOGS_ENV` | The environment (e.g., `development`, `production`). | `development` |
| `RAINLOGS_PORT` | The port the API listens on. | `8080` |
| `RAINLOGS_LOG_LEVEL` | The logging level (e.g., `debug`, `info`, `warn`, `error`). | `info` |

### Database

| Variable | Description | Default |
|---|---|---|
| `RAINLOGS_DB_HOST` | The PostgreSQL host. | `localhost` |
| `RAINLOGS_DB_PORT` | The PostgreSQL port. | `5432` |
| `RAINLOGS_DB_USER` | The PostgreSQL user. | `postgres` |
| `RAINLOGS_DB_PASSWORD` | The PostgreSQL password. | `postgres` |
| `RAINLOGS_DB_NAME` | The PostgreSQL database name. | `rainlogs` |

### Redis

| Variable | Description | Default |
|---|---|---|
| `RAINLOGS_REDIS_HOST` | The Redis host. | `localhost` |
| `RAINLOGS_REDIS_PORT` | The Redis port. | `6379` |
| `RAINLOGS_REDIS_PASSWORD` | The Redis password. | `""` |

### Storage (Garage S3)

| Variable | Description | Default |
|---|---|---|
| `RAINLOGS_STORAGE_ENDPOINT` | The S3 endpoint URL. | `http://localhost:3900` |
| `RAINLOGS_STORAGE_REGION` | The S3 region. | `us-east-1` |
| `RAINLOGS_STORAGE_BUCKET` | The S3 bucket name. | `rainlogs-logs` |
| `RAINLOGS_STORAGE_ACCESS_KEY` | The S3 access key ID. | `""` |
| `RAINLOGS_STORAGE_SECRET_KEY` | The S3 secret access key. | `""` |

### Security

| Variable | Description | Default |
|---|---|---|
| `RAINLOGS_KMS_KEY` | The 32-byte base64-encoded KMS key used for encryption. | `""` |

### Cloudflare

| Variable | Description | Default |
|---|---|---|
| `RAINLOGS_CLOUDFLARE_RATE_LIMIT` | The rate limit for Cloudflare API requests (requests per second). | `0` (unlimited) |
| `RAINLOGS_CLOUDFLARE_MAX_WINDOW_SIZE` | Max log pull window per request. | `1h` |

## Configuration File

You can also provide a `config.yaml` file in the root directory of the application. The structure mirrors the environment variables.

```yaml
env: development
port: 8080
log_level: info

db:
  host: localhost
  port: 5432
  user: postgres
  password: password
  name: rainlogs

redis:
  host: localhost
  port: 6379
  password: ""

storage:
  endpoint: http://localhost:3900
  region: us-east-1
  bucket: rainlogs-logs
  access_key: your-access-key
  secret_key: your-secret-key

kms:
  key: your-base64-encoded-key
```

## Zero-Config Initialization

If the `RAINLOGS_STORAGE_ACCESS_KEY` and `RAINLOGS_STORAGE_SECRET_KEY` are not provided, Rainlogs will attempt to automatically generate them by connecting to the Garage S3 instance and creating the necessary buckets. This is the recommended approach for local development and testing.
