#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# Preferred: DEPLOY_ENV_FILE override, then .env.production, then legacy .env_prod.
ENV_FILE="${DEPLOY_ENV_FILE:-$ROOT/.env.production}"
if [[ ! -f "$ENV_FILE" && -f "$ROOT/.env_prod" ]]; then
  ENV_FILE="$ROOT/.env_prod"
fi
if [[ ! -f "$ENV_FILE" ]]; then
  echo "ERROR: production environment file not found. Create .env.production first (see .env.example)." >&2
  exit 1
fi
# Export so docker compose can interpolate ${ENV_FILE:-.env} inside
# docker-compose.prod.yml (env_file) and the API container gets the
# production secrets instead of the dev .env.
export ENV_FILE="$ENV_FILE"

set -a
# shellcheck disable=SC1090
source "$ENV_FILE"
set +a

if [[ -z "${JWT_SECRET:-}" || "$JWT_SECRET" == "change-me-32-chars-min" || "$JWT_SECRET" == "change-me-in-production" ]]; then
  echo "ERROR: JWT_SECRET must be replaced with a strong production value." >&2
  exit 1
fi
if [[ -z "${DB_PASSWORD:-}" || "$DB_PASSWORD" == "aimain" || "$DB_PASSWORD" == "change-me" ]]; then
  echo "ERROR: DB_PASSWORD must be replaced with a strong production value." >&2
  exit 1
fi
if [[ -z "${ENCRYPTION_KEY:-}" || "$ENCRYPTION_KEY" == "change-me-32-byte-key-for-aes-gcm" ]]; then
  echo "ERROR: ENCRYPTION_KEY must be replaced with a strong production value." >&2
  exit 1
fi

COMPOSE_FILE="$ROOT/docker-compose.prod.yml"
echo "-> validating production compose configuration"
docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" config >/dev/null
echo "-> building production stack"
docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" build --pull
echo "-> starting production stack"
docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" up -d --remove-orphans

echo "-> waiting for backend health..."
for _ in $(seq 1 60); do
  if curl -fsS http://localhost:8088/health >/dev/null 2>&1; then
    echo "-> backend healthy: http://localhost:8088"
    echo "-> frontend: http://localhost:${FRONTEND_PORT:-8082}"
    docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" ps
    exit 0
  fi
  sleep 2
done

echo "ERROR: backend did not become healthy. Check 'docker compose logs api' or 'docker compose ps'." >&2
exit 1
