# XDP47 – Docker on Ubuntu (Control + Agents)

This pack lets you run the control plane, PostgreSQL, and demo agents as Ubuntu-based containers.

## Quick start (development, localhost)
```bash
# from project root (where go.mod is), copy `docker/` and `scripts/`
cp -r docker ./docker.bak 2>/dev/null || true
cp -r /mnt/data/xdp47-docker-pack/docker .
cp -r /mnt/data/xdp47-docker-pack/scripts .
cp /mnt/data/xdp47-docker-pack/docker/.env.example docker/.env

# bring up DB + control + demo agents
./scripts/dev_bootstrap.sh
# UI:
#   http://127.0.0.1:8080/ui/devices
#   http://127.0.0.1:8080/ui/rollouts
```

## Build images explicitly
```bash
docker build -t xdp47/control:latest -f docker/control/Dockerfile .
docker build -t xdp47/agent:latest   -f docker/agent/Dockerfile   .
```

## Edge agent (on customer machine)
```bash
export XDP47_CONTROL_URL="https://control.example.com"
export XDP47_TENANT="demo-tenant"
export XDP47_DEVICE_LABELS="store=sofia-01,role=kiosk,track=canary-only"

docker compose -f docker/docker-compose.edge-agent.yml up -d
# or
docker run -d --restart unless-stopped   -e XDP47_CONTROL_URL -e XDP47_TENANT -e XDP47_DEVICE_LABELS   xdp47/agent:latest
```

## Notes
- Control connects to Postgres **inside compose** via service name `db`.
- Scheduler knobs are in `docker/.env`:
  - `XDP47_SCHED_INTERVAL`, `XDP47_SCHED_GRACE`, `XDP47_SCHED_REQUIRE_OK`, `XDP47_SCHED_SKIP_OFFLINE`
- Images are Ubuntu-based.
