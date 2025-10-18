#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
export COMPOSE_FILE=docker/docker-compose.dev.yml
docker compose build
docker compose up -d
echo "UI:   http://127.0.0.1:8080/ui/devices"
echo "Roll: http://127.0.0.1:8080/ui/rollouts"
