#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REDIS_CONTAINER="main-dashboard-local-redis"
REDIS_STARTED_BY_SCRIPT=0
SEARXNG_CONTAINER="main-dashboard-local-searxng"
SEARXNG_STARTED_BY_SCRIPT=0
CRAWLER_CONTAINER="main-dashboard-local-crawler"
CRAWLER_IMAGE="intellisearch-crawler"
CRAWLER_STARTED_BY_SCRIPT=0
BACKEND_PID=""
FRONTEND_PID=""

if [[ ! -f "$ROOT/.env" ]]; then
  echo "-> creating .env from .env.example"
  cp "$ROOT/.env.example" "$ROOT/.env"
fi

# Make the shared root environment available to both host-run processes.
set -a
# shellcheck disable=SC1091
source "$ROOT/.env"
set +a

export APP_ENV="${APP_ENV:-development}"
export PORT="${PORT:-8088}"
export CORS_ORIGINS="${CORS_ORIGINS:-http://localhost:5173}"
# Host-run development uses local Redis + SearXNG + crawler containers and SQLite.
export REDIS_ADDR="127.0.0.1:6379"
export SEARXNG_BASE_URL="http://localhost:8081"
export CRAWLER_BASE_URL="http://localhost:3001"
export DB_DRIVER="sqlite"
export DB_SQLITE_PATH="${DB_SQLITE_PATH:-$ROOT/backend/data/intellisearch.db}"

if ! command -v podman >/dev/null 2>&1; then
	 echo "ERROR: podman is required because local Redis, SearXNG, and the crawler are managed by run-local.sh." >&2
  exit 1
fi


if [[ -n "$(podman ps -q -f "name=^/${REDIS_CONTAINER}$")" ]]; then
  echo "-> Redis is already running ($REDIS_CONTAINER)"
elif [[ -n "$(podman ps -aq -f "name=^/${REDIS_CONTAINER}$")" ]]; then
  echo "-> starting existing Redis container ($REDIS_CONTAINER)"
  podman start "$REDIS_CONTAINER" >/dev/null
  REDIS_STARTED_BY_SCRIPT=1
else
  echo "-> creating local Redis container ($REDIS_CONTAINER)"
  podman run -d --name "$REDIS_CONTAINER" -p 6379:6379 -v main_dashboard_local_redis_data:/data redis:7-alpine redis-server --appendonly yes >/dev/null
  REDIS_STARTED_BY_SCRIPT=1
fi

for _ in $(seq 1 30); do
  if podman exec "$REDIS_CONTAINER" redis-cli ping 2>/dev/null | grep -q PONG; then
    break
  fi
  sleep 1
done
if ! podman exec "$REDIS_CONTAINER" redis-cli ping 2>/dev/null | grep -q PONG; then
  echo "ERROR: local Redis did not become ready." >&2
  exit 1
fi

# SearXNG (self-hosted metasearch) — host port 8081 -> container port 8080 so it
# never collides with the API's :8088. The JSON format is enabled in
# searxng/settings.yml (search.formats: [html, json]).
if [[ -n "$(podman ps -q -f "name=^/${SEARXNG_CONTAINER}$")" ]]; then
  echo "-> SearXNG is already running ($SEARXNG_CONTAINER)"
elif [[ -n "$(podman ps -aq -f "name=^/${SEARXNG_CONTAINER}$")" ]]; then
  echo "-> starting existing SearXNG container ($SEARXNG_CONTAINER)"
  podman start "$SEARXNG_CONTAINER" >/dev/null
  SEARXNG_STARTED_BY_SCRIPT=1
else
  echo "-> creating local SearXNG container ($SEARXNG_CONTAINER)"
  podman run -d --name "$SEARXNG_CONTAINER" -p 8081:8080 -v "$ROOT/searxng/settings.yml:/etc/searxng/settings.yml:ro" searxng/searxng:latest >/dev/null
  SEARXNG_STARTED_BY_SCRIPT=1
fi

SEARXNG_READY=0
for _ in $(seq 1 60); do
  if curl -fsS "http://localhost:8081/healthz" >/dev/null 2>&1 || curl -fsS -o /dev/null "http://localhost:8081/" >/dev/null 2>&1; then
    SEARXNG_READY=1
    break
  fi
  sleep 1
done
if [[ "$SEARXNG_READY" != "1" ]]; then
  echo "ERROR: local SearXNG did not become ready on http://localhost:8081." >&2
  exit 1
fi
echo "-> SearXNG ready on http://localhost:8081"

# Crawler (Playwright page fetcher) — host port 3001 -> container port 3002.
# The image is built from crawler/ (Playwright base image + npm ci). Building it
# the first time pulls the Playwright base image and can take a few minutes.
if ! podman image exists "$CRAWLER_IMAGE" >/dev/null 2>&1; then
  echo "-> building crawler image ($CRAWLER_IMAGE)..."
  podman build -t "$CRAWLER_IMAGE" "$ROOT/crawler"
fi
if [[ -n "$(podman ps -q -f "name=^/${CRAWLER_CONTAINER}$")" ]]; then
  echo "-> Crawler is already running ($CRAWLER_CONTAINER)"
elif [[ -n "$(podman ps -aq -f "name=^/${CRAWLER_CONTAINER}$")" ]]; then
  echo "-> starting existing crawler container ($CRAWLER_CONTAINER)"
  podman start "$CRAWLER_CONTAINER" >/dev/null
  CRAWLER_STARTED_BY_SCRIPT=1
else
  echo "-> creating local crawler container ($CRAWLER_CONTAINER)"
  podman run -d --name "$CRAWLER_CONTAINER" -p 3001:3002 "$CRAWLER_IMAGE" >/dev/null
  CRAWLER_STARTED_BY_SCRIPT=1
fi

CRAWLER_READY=0
for _ in $(seq 1 60); do
  if curl -fsS "http://localhost:3001/health" >/dev/null 2>&1; then
    CRAWLER_READY=1
    break
  fi
  sleep 1
done
if [[ "$CRAWLER_READY" != "1" ]]; then
  echo "ERROR: local crawler did not become ready on http://localhost:3001." >&2
  exit 1
fi
echo "-> Crawler ready on http://localhost:3001"

cleanup() {
  echo ""
  echo "-> shutting down..."
  [[ -n "$BACKEND_PID" ]] && kill "$BACKEND_PID" 2>/dev/null || true
  [[ -n "$FRONTEND_PID" ]] && kill "$FRONTEND_PID" 2>/dev/null || true
  [[ -n "$BACKEND_PID" ]] && wait "$BACKEND_PID" 2>/dev/null || true
  [[ -n "$FRONTEND_PID" ]] && wait "$FRONTEND_PID" 2>/dev/null || true
  if [[ "$REDIS_STARTED_BY_SCRIPT" == "1" ]]; then
    podman stop "$REDIS_CONTAINER" >/dev/null 2>&1 || true
  fi
  if [[ "$SEARXNG_STARTED_BY_SCRIPT" == "1" ]]; then
    podman stop "$SEARXNG_CONTAINER" >/dev/null 2>&1 || true
  fi
  if [[ "$CRAWLER_STARTED_BY_SCRIPT" == "1" ]]; then
    podman stop "$CRAWLER_CONTAINER" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT INT TERM

echo "-> starting backend (APP_ENV=$APP_ENV)"
(cd "$ROOT/backend" && go run ./cmd/server) &
BACKEND_PID=$!

for _ in $(seq 1 60); do
  if curl -fsS "http://localhost:${PORT}/health" >/dev/null 2>&1; then
    break
  fi
  sleep 1
done
if ! curl -fsS "http://localhost:${PORT}/health" >/dev/null 2>&1; then
  echo "ERROR: backend did not become ready." >&2
  exit 1
fi
echo "-> backend ready on http://localhost:${PORT}"

if [[ "${1:-}" == "--no-frontend" ]]; then
  echo "-> skipping frontend (dev) server"
  wait "$BACKEND_PID"
  exit 0
fi

echo "-> starting frontend (dev) on http://localhost:5173"
(cd "$ROOT/frontend" && npm run dev) &
FRONTEND_PID=$!
wait "$BACKEND_PID" "$FRONTEND_PID"
