#!/bin/bash
set -e

# RainLogs "Super Easy" Installer
# Deploys the full stack using Docker Compose with Traefik (HTTPS).

echo "🌧️  Installing RainLogs..."

# 1. Check dependencies
if ! command -v docker &> /dev/null; then
    echo "❌ Docker is not installed. Please install Docker first."
    echo "💡 Tip: curl -fsSL https://get.docker.com | sh"
    exit 1
fi

# 2. Setup Configuration
if [ ! -f .env ]; then
    echo "📝 Creating .env from example..."
    cp .env.example .env
    
    # Generate secrets
    JWT_SECRET=$(openssl rand -hex 32)
    KMS_KEY=$(openssl rand -hex 32)
    RPC_SECRET=$(openssl rand -hex 32)
    S3_ACCESS=$(openssl rand -hex 16)
    S3_SECRET=$(openssl rand -hex 32)
    DB_PASS=$(openssl rand -hex 16)
    REDIS_PASS=$(openssl rand -hex 16)

    # Sed is different on MacOS vs Linux
    if [[ "$OSTYPE" == "darwin"* ]]; then
      SED_CMD="sed -i ''"
    else
      SED_CMD="sed -i"
    fi

    $SED_CMD "s/JWT_SECRET=.*/JWT_SECRET=$JWT_SECRET/" .env
    $SED_CMD "s/KMS_KEY=.*/KMS_KEY=$KMS_KEY/" .env
    $SED_CMD "s/GARAGE_RPC_SECRET=.*/GARAGE_RPC_SECRET=$RPC_SECRET/" .env
    $SED_CMD "s/S3_ACCESS_KEY_ID=.*/S3_ACCESS_KEY_ID=$S3_ACCESS/" .env
    $SED_CMD "s/S3_SECRET_ACCESS_KEY=.*/S3_SECRET_ACCESS_KEY=$S3_SECRET/" .env
    $SED_CMD "s/POSTGRES_PASSWORD=.*/POSTGRES_PASSWORD=$DB_PASS/" .env
    $SED_CMD "s/REDIS_PASSWORD=.*/REDIS_PASSWORD=$REDIS_PASS/" .env
    
    echo "✅ Configuration generated."
else
    echo "ℹ️  .env already exists, skipping generation."
fi

# 3. Create required directories
mkdir -p docker/garage
# Ensure init script exists (it should be in repo)

# 4. Start Stack
echo "🚀 Starting RainLogs stack..."
docker compose up -d

echo ""
echo "🎉 RainLogs is running!"
echo "👉 Dashboard: https://localhost (Traefik self-signed or configured cert)"
echo "👉 Logs: docker compose logs -f"
