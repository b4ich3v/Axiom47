# XDP47 — Edge Orchestrator for Retail/Kiosk Fleets (Go)

Minimal starter to begin implementing the control plane and agent step by step.

## What’s here (MVP-0)
- **Control Plane** (`cmd/xdp47-control/`): HTTP API with `/healthz`, device claim stub, and SSE stream for demo metrics.
- **Agent** (`cmd/xdp47-agent/`): Registers (fake claim) and sends periodic heartbeats/metrics to the control plane.
- **OpenAPI** (`api/openapi.yaml`): Minimal spec for the first endpoints.
- **Systemd** (`packaging/systemd/*.service`): Units for agent and control.
- **Makefile**: Simple build/run targets.
- **.env.sample**: Config envs for local dev.

## Quickstart
```bash
# 1) Adjust module path in go.mod if you want (e.g., github.com/yourname/xdp47)
# 2) Build
make build

# 3) Run control plane (terminal A)
./bin/xdp47-control

# 4) Run agent (terminal B)
XDP47_CONTROL_URL=http://127.0.0.1:8080         XDP47_TENANT=demo-tenant         XDP47_DEVICE_LABELS=store=sofia-01,role=kiosk         ./bin/xdp47-agent

# 5) Try the SSE stream in your browser:
# http://127.0.0.1:8080/api/devices/demo-device/metrics/stream
```

## Next steps
- Persist devices and claims in Postgres (replace in-memory store).
- Implement signed artifact registry endpoints.
- Add rollout scheduler skeleton and policy evaluation.
- Replace fake SSE metrics with real eBPF (/proc fallback) collectors.
