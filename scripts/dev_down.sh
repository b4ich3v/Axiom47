#!/usr/bin/env bash
set -euo pipefail
export COMPOSE_FILE=docker/docker-compose.dev.yml
docker compose down -v
