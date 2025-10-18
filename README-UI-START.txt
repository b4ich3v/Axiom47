XDP47 UI "Start" Button Patch
=============================

This patch replaces the uiRollouts() function in cmd/xdp47-control/main.go
to add a **Start** button per rollout row (calls POST /api/rollouts/{id}/start,
with :start fallback).

How to apply:
1) Open: D:\XDP47\cmd\xdp47-control\main.go
2) Find the existing function:
     func uiRollouts(w http.ResponseWriter, r *http.Request) { ... }
3) Replace the entire function body with the contents of:
     cmd/xdp47-control/ui_rollouts_patch_snippet.go.txt
   (You can copy/paste it; it includes the full function definition.)
4) Save the file.
5) Rebuild and run:
     go mod tidy
     mingw32-make build
     set XDP47_DB_URL=postgres://postgres:***@localhost:5432/xdp47?sslmode=disable
     set XDP47_LISTEN_ADDR=:8080
     .\bin\xdp47-control.exe
6) Open http://127.0.0.1:8080/ui/rollouts and click **Start** on a row.

Notes:
- Server must have both routes registered (we already added both):
    r.Post("/api/rollouts/{id}:start", startRollout)
    r.Post("/api/rollouts/{id}/start", startRollout)
- The UI tries /start first, then :start.
