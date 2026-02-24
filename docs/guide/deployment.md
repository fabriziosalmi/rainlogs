# Deployment

Rainlogs is designed to be easily deployed using Docker Compose. This guide covers the recommended deployment strategy for production environments.

## Prerequisites

- A Linux server (e.g., Ubuntu 22.04 LTS)
- Docker and Docker Compose installed
- A domain name pointing to your server's IP address
- SSL certificates (e.g., Let's Encrypt)

## Production Setup

1. **Clone the repository:**

   ```bash
   git clone https://github.com/fabriziosalmi/rainlogs.git
   cd rainlogs
   ```

2. **Configure Environment Variables:**

   Create a `.env` file in the root directory and configure the necessary variables. See the [Configuration](./configuration.md) guide for details.

   ```bash
   RAINLOGS_ENV=production
   RAINLOGS_PORT=8080
   RAINLOGS_DB_PASSWORD=your-secure-password
   RAINLOGS_REDIS_PASSWORD=your-secure-password
   RAINLOGS_KMS_KEY=your-base64-encoded-key
   ```

3. **Start the Infrastructure:**

   Use Docker Compose to start the services in detached mode.

   ```bash
   docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d
   ```

   This will start PostgreSQL, Redis, Garage S3, Asynqmon, and the Rainlogs API.

4. **Reverse Proxy (Nginx/Traefik):**

   Set up a reverse proxy to handle SSL termination and route traffic to the Rainlogs API and Asynqmon dashboard.

   **Example Nginx Configuration:**

   ```nginx
   server {
       listen 80;
       server_name rainlogs.yourdomain.com;
       return 301 https://$host$request_uri;
   }

   server {
       listen 443 ssl;
       server_name rainlogs.yourdomain.com;

       ssl_certificate /etc/letsencrypt/live/rainlogs.yourdomain.com/fullchain.pem;
       ssl_certificate_key /etc/letsencrypt/live/rainlogs.yourdomain.com/privkey.pem;

       location / {
           proxy_pass http://localhost:8080;
           proxy_set_header Host $host;
           proxy_set_header X-Real-IP $remote_addr;
           proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
           proxy_set_header X-Forwarded-Proto $scheme;
       }

       location /asynqmon/ {
           proxy_pass http://localhost:8081/;
           proxy_set_header Host $host;
           proxy_set_header X-Real-IP $remote_addr;
           proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
           proxy_set_header X-Forwarded-Proto $scheme;
       }
   }
   ```

5. **Monitoring and Logging:**

   Monitor the health of your services using tools like Prometheus and Grafana. Ensure that logs from the Rainlogs API and other containers are collected and analyzed.

## High Availability

For high availability, consider deploying Rainlogs across multiple servers. This involves setting up a PostgreSQL cluster, a Redis cluster, and a distributed Garage S3 cluster. You will also need a load balancer to distribute traffic across the Rainlogs API instances.

## Kubernetes / K3s

Rainlogs includes a set of Kubernetes manifests for deployment on K8s or lightweight distributions like K3s. These are located in the `k8s/` directory.

### Prerequisites

- A Kubernetes cluster (k8s v1.24+ or K3s)
- `kubectl` configured
- Persistent Storage Class (Longhorn, OpenEBS, or default hostPath for K3s)

### Deployment Steps

1. **Apply Base Configurations**
   Create namespaces and base RBAC roles.
   ```bash
   kubectl apply -f k8s/00-base.yaml
   ```

2. **Run Database Migrations**
   Deploy a Job to initialize the database schema.
   ```bash
   kubectl apply -f k8s/05-migrations.yaml
   ```

3. **Deploy Dependencies**
   Deploy PostgreSQL and Redis (if not using managed cloud services).
   ```bash
   kubectl apply -f k8s/10-dependencies.yaml
   ```

4. **Deploy Object Storage (Garage)**
   Deploy the Garage S3-compatible object storage.
   ```bash
   kubectl apply -f k8s/15-garage.yaml
   ```

5. **Deploy Application**
   Deploy the API and Worker components.
   ```bash
   kubectl apply -f k8s/20-app.yaml
   ```

6. **Configure Ingress**
   Expose the service via Ingress (Nginx/Traefik).
   ```bash
   kubectl apply -f k8s/25-ingress.yaml
   ```

7. **Optional Components**
   - **HPA**: Enable horizontal pod autoscaling for the API.
     ```bash
     kubectl apply -f k8s/30-hpa.yaml
     ```
   - **External Secrets**: If using AWS Secrets Manager or Vault.
     ```bash
     kubectl apply -f k8s/35-external-secrets.yaml
     ```

## Bare Metal Deployment

For direct installation on Linux verify (Ubuntu/Debian/RHEL) without containers.

### 1. Build from Source

```bash
# Install Go 1.24+ first
make build

# Binaries will be in ./bin/
ls -l bin/
# - rainlogs-api
# - rainlogs-worker
```

### 2. Database Setup

Install and configure PostgreSQL (v14+) and Redis (v6+).

```bash
# Create Database
sudo -u postgres psql -c "CREATE DATABASE rainlogs;"
sudo -u postgres psql -c "CREATE USER rainlogs WITH ENCRYPTED PASSWORD 'secure_password';"
sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE rainlogs TO rainlogs;"

# Run Migrations
export RAINLOGS_DATABASE_URL="postgres://rainlogs:secure_password@localhost:5432/rainlogs?sslmode=disable"
make migrate-up
```

### 3. Systemd Configuration

Create a systemd unit for the **API Service**: `/etc/systemd/system/rainlogs-api.service`

```ini
[Unit]
Description=Rainlogs API
After=network.target postgresql.service redis.service

[Service]
User=rainlogs
Group=rainlogs
ExecStart=/opt/rainlogs/rainlogs-api
WorkingDirectory=/opt/rainlogs
EnvironmentFile=/etc/rainlogs/config.env
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Create a systemd unit for the **Worker Service**: `/etc/systemd/system/rainlogs-worker.service`

```ini
[Unit]
Description=Rainlogs Worker
After=network.target postgresql.service redis.service

[Service]
User=rainlogs
Group=rainlogs
ExecStart=/opt/rainlogs/rainlogs-worker
WorkingDirectory=/opt/rainlogs
EnvironmentFile=/etc/rainlogs/config.env
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

### 4. Enable Services

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now rainlogs-api
sudo systemctl enable --now rainlogs-worker
```
