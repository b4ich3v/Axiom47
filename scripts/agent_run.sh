#!/usr/bin/env bash
set -euo pipefail
docker run --rm -e XDP47_CONTROL_URL="${XDP47_CONTROL_URL:?}"   -e XDP47_TENANT="${XDP47_TENANT:-demo-tenant}"   -e XDP47_DEVICE_LABELS="${XDP47_DEVICE_LABELS:-store=unknown,role=kiosk}"   xdp47/agent:latest
