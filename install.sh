#!/usr/bin/env bash
# install.sh – One-shot RainLogs setup for Docker Compose
# Usage: bash install.sh  OR  curl -fsSL https://raw.githubusercontent.com/fabriziosalmi/rainlogs/main/install.sh | bash
set -euo pipefail

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; NC='\033[0m'
info()  { echo -e "${GREEN}[rainlogs]${NC} $*"; }
warn()  { echo -e "${YELLOW}[rainlogs]${NC} $*"; }
fatal() { echo -e "${RED}[rainlogs] ERROR:${NC} $*" >&2; exit 1; }

# ── Prerequisites ──────────────────────────────────────────────────────────────
command -v docker  >/dev/null 2>&1 || fatal "docker is required – https://docs.docker.com/get-docker/"
command -v openssl >/dev/null 2>&1 || fatal "openssl is required"

if docker compose version >/dev/null 2>&1; then
  COMPOSE="docker compose"
elif command -v docker-compose >/dev/null 2>&1; then
  COMPOSE="docker-compose"
else
  fatal "Docker Compose v2 is required – https://docs.docker.com/compose/install/"
fi

# ── Clone if running via pipe (no docker-compose.yml in cwd) ──────────────────
if [[ ! -f "docker-compose.yml" ]]; then
  command -v git >/dev/null 2>&1 || fatal "git is required when running outside the repo"
  info "Cloning rainlogs repository..."
  git clone https://github.com/fabriziosalmi/rainlogs.git
  cd rainlogs
fi

# ── Generate secrets into .env ─────────────────────────────────────────────────
if [[ -f ".env" ]]; then
  warn ".env already exists – skipping secret generation. Remove it to regenerate."
else
  info "Generating secrets and writing .env..."
  JWT_SECRET=$(openssl rand -hex 32)
  KMS_KEY=$(openssl rand -hex 32)
  REDIS_PASS=$(openssl rand -hex 16)
  DB_PASS=$(openssl rand -hex 16)
  RPC_SECRET=$(openssl rand -hex 32)

  cp .env.example .env

  # Cross-platform sed (-i '' on macOS, -i on Linux)
  sedi() { sed -i${OSTYPE+.bak} "$@"; rm -f "${!#}.bak" 2>/dev/null || true; }

  sedi "s|^RAINLOGS_JWT_SECRET=.*|RAINLOGS_JWT_SECRET=${JWT_SECRET}|" .env
  sedi "s|^RAINLOGS_KMS_KEY=.*|RAINLOGS_KMS_KEY=${KMS_KEY}|" .env
  sedi "s|^RAINLOGS_REDIS_PASSWORD=.*|RAINLOGS_REDIS_PASSWORD=${REDIS_PASS}|" .env
  sedi "s|^POSTGRES_PASSWORD=.*|POSTGRES_PASSWORD=${DB_PASS}|" .env
  sedi "s|^GARAGE_RPC_SECRET=.*|GARAGE_RPC_SECRET=${RPC_SECRET}|" .env

  info ".env created with generated secrets."
fi

# ── Start stack ────────────────────────────────────────────────────────────────
info "Starting services (this may take a minute on first run)..."
$COMPOSE up -d --build

# ── Wait for API health ────────────────────────────────────────────────────────
info "Waiting for API to become healthy..."
MAX_TRIES=30
for i in $(seq 1 $MAX_TRIES); do
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/health 2>/dev/null || true)
  if [[ "$STATUS" == "200" ]]; then
    info "API is healthy."
    break
  fi
  if [[ "$i" == "$MAX_TRIES" ]]; then
    warn "API did not become healthy in time. Check logs: $COMPOSE logs api"
  fi
  sleep 3
done

# ── Done ───────────────────────────────────────────────────────────────────────
echo ""
info "RainLogs is running!"
echo ""
echo "  API endpoint:   http://localhost:8080"
echo "  Health check:   http://localhost:8080/health"
echo "  Queue monitor:  http://localhost:8383  (Asynqmon)"
echo ""
echo "Quick start:"
echo "  1. POST /customers          – register your account"
echo "  2. POST /api/v1/api-keys    – issue a bearer token"
echo "  3. POST /api/v1/zones       – add a Cloudflare zone"
echo "  4. GET  /api/v1/zones       – list zones (includes health field)"
echo ""
warn "Never commit .env to version control – it contains your secrets."
