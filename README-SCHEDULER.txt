# XDP47 Scheduler + Auto-rollback (MVP) Patch

## Adds
- internal/db/rollouts_ex.go     (extra migrations + rollout ops + device filtering)
- internal/scheduler/scheduler.go (wave scheduler with auto-rollback)
- cmd/xdp47-control/scheduler_patch.go (endpoint /api/rollouts/{id}:start)

## Usage
1) Extract into your project root (allow merge).
2) Run:
     go mod tidy
     mingw32-make build
3) Start control-plane with DB envs as before.
4) Create a rollout (POST /api/rollouts).
5) Start it:
     POST /api/rollouts/<id>:start
   The scheduler will:
     - split devices into <waves> buckets (round-robin)
     - for each wave: check last_seen freshness and status == "ok"
     - if any device fails a wave → rollout status becomes "failed" (auto-rollback)
     - otherwise, after all waves → "completed"

## Notes
- Heartbeat grace = 20s, wave interval = 8s, RequireOK=true (can be tuned in code).
- History stored in table rollout_runs.
