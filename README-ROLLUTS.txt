# XDP47 Rollouts MVP Patch

## Files in this patch
- internal/db/rollouts.go  (DB schema + CRUD for rollouts)
- cmd/xdp47-control/rollouts_patch_main.go  (handlers for /api/rollouts + UI)

## Install
1) Extract over your project root (allow overwrite/merge).
2) Run:
   go mod tidy
   mingw32-make build

## Run
   $env:XDP47_DB_URL="postgres://postgres:drishlyoto1@localhost:5432/xdp47?sslmode=disable"
   $env:XDP47_LISTEN_ADDR=":8080"
   .\bin\xdp47-control.exe

## Use
- UI:  GET http://127.0.0.1:8080/ui/rollouts
- API: POST http://127.0.0.1:8080/api/rollouts
        body: {"tenant":"demo-tenant","artifact":"app:v1.2.3","channel":"canary","selector":{"role":"kiosk"},"waves":3}
  Simulate:
     POST http://127.0.0.1:8080/api/rollouts/<id>:simulate
     or: POST with ?waves=3 to try different split

## Notes
- This is an MVP: no scheduler yet; it's for planning and preview.
- Devices are taken from DB (or memory fallback).
