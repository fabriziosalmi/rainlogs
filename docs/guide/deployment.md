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
