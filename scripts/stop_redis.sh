#!/usr/bin/env bash
set -euo pipefail
DIR=$(cd "$(dirname "$0")/.." && pwd)
cd "$DIR"
docker compose -f docker-compose.redis.yml down
