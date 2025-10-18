package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "math/rand"
    "net/http"
    "os"
    "sort"
    "strings"
    "time"

    "github.com/go-chi/chi/v5"
    "github.com/go-chi/chi/v5/middleware"

    xdb "github.com/example/xdp47/internal/db"
    scheduler "github.com/example/xdp47/internal/scheduler"
)

type Device struct {
    ID       string            `json:"id"`
    Tenant   string            `json:"tenant"`
    Labels   map[string]string `json:"labels"`
    LastSeen time.Time         `json:"last_seen"`
    Health   string            `json:"health"` // "ok" | "warn" | "crit" | "unknown"
    Location string            `json:"location"`
    Version  string            `json:"version"`
    Channel  string            `json:"channel"`
}

// in-memory fallback
var devices = map[string]*Device{}
var store *xdb.Store

func main() {
    // DB connect (optional, with retry)
    dbURL := os.Getenv("XDP47_DB_URL")
    if dbURL != "" {
        ctx := context.Background()
        var err error
        for i := 0; i < 6; i++ {
            store, err = xdb.Connect(ctx, dbURL)
            if err == nil {
                if err = store.Migrate(ctx); err == nil {
                    _ = store.MigrateRollouts(ctx)
                    _ = store.MigrateScheduler(ctx)
                    log.Printf("[db] connected & migrated: %s", redacted(dbURL))
                    break
                }
            }
            log.Printf("[db] connect/migrate failed: %v (retrying...)", err)
            time.Sleep(time.Duration(1<<i) * time.Second)
        }
        if store == nil || !store.Enabled {
            log.Printf("[db] giving up; running in memory mode")
        }
    } else {
        log.Printf("[db] XDP47_DB_URL not set; running in memory mode")
    }

    addr := os.Getenv("XDP47_LISTEN_ADDR")
    if addr == "" {
        addr = ":8080"
    }

    r := chi.NewRouter()
    r.Use(middleware.RequestID)
    r.Use(middleware.Logger)
    r.Use(middleware.Recoverer)
    r.Get ("/api/rollouts/{id}/runs",  getRolloutRuns)
    r.Post("/api/rollouts/{id}:retry", retryRollout)


    r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write([]byte("ok"))
    })

    // Devices
    r.Get("/api/devices", listDevices)
    r.Post("/api/devices/claim", claimHandler)
    r.Post("/api/devices/{id}/heartbeat", heartbeatHandler)
    r.Get("/api/devices/{id}/metrics/stream", sseMetrics)

    // Rollouts
    r.Get("/api/rollouts", listRollouts)
    r.Post("/api/rollouts", createRollout)
    r.Post("/api/rollouts/{id}:simulate", simulateRollout)

    // Scheduler start (РїРѕРґРґСЉСЂР¶Р°РјРµ Рё РґРІР°С‚Р° РїСЉС‚СЏ)
    r.Post("/api/rollouts/{id}:start", startRollout)
    r.Post("/api/rollouts/{id}/start", startRollout)

    // UI
    r.Get("/ui/devices", uiDevices)
    r.Get("/ui/rollouts", uiRollouts)

    log.Printf("xdp47-control listening on %s", addr)
    log.Fatal(http.ListenAndServe(addr, r))
}

// --- helpers ---

func redacted(url string) string {
    // postgres://user:pass@host:5432/db -> postgres://user:***@host:5432/db
    if i := index(url, "://"); i >= 0 {
        if j := index(url, "@"); j > i {
            k := i + 3
            if q := index(url[k:j], ":"); q >= 0 {
                return url[:k+q+1] + "***" + url[k+q+1:]
            }
        }
    }
    return url
}
func index(s, sub string) int {
    for i := 0; i+len(sub) <= len(s); i++ {
        if s[i:i+len(sub)] == sub {
            return i
        }
    }
    return -1
}

func parseDurationEnv(key string, def time.Duration) time.Duration {
    if v := os.Getenv(key); v != "" {
        if d, err := time.ParseDuration(v); err == nil {
            return d
        }
    }
    return def
}
func parseBoolEnv(key string, def bool) bool {
    if v := os.Getenv(key); v != "" {
        if v == "1" || strings.EqualFold(v, "true") {
            return true
        }
        if v == "0" || strings.EqualFold(v, "false") {
            return false
        }
    }
    return def
}

// --- devices handlers ---

func listDevices(w http.ResponseWriter, r *http.Request) {
    if store != nil && store.Enabled {
        rows, err := store.ListDevices(r.Context())
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        w.Header().Set("Content-Type", "application/json")
        _ = json.NewEncoder(w).Encode(rows)
        return
    }

    // fallback (memory)
    type devOut struct {
        ID       string            `json:"id"`
        Tenant   string            `json:"tenant"`
        Labels   map[string]string `json:"labels"`
        LastSeen time.Time         `json:"last_seen"`
        Health   string            `json:"health"`
        Version  string            `json:"version"`
        Channel  string            `json:"channel"`
    }
    out := make([]devOut, 0, len(devices))
    for _, d := range devices {
        out = append(out, devOut{
            ID: d.ID, Tenant: d.Tenant, Labels: d.Labels, LastSeen: d.LastSeen, Health: d.Health,
            Version: d.Version, Channel: d.Channel,
        })
    }
    sort.Slice(out, func(i, j int) bool {
        if out[i].LastSeen.Equal(out[j].LastSeen) {
            return out[i].ID < out[j].ID
        }
        return out[i].LastSeen.After(out[j].LastSeen)
    })
    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(out)
}

func claimHandler(w http.ResponseWriter, r *http.Request) {
    type req struct {
        Tenant   string                 `json:"tenant"`
        Labels   map[string]interface{} `json:"labels"`
        Location string                 `json:"location"`
        Version  string                 `json:"version"`
        Channel  string                 `json:"channel"`
    }
    var q req
    if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    if q.Tenant == "" {
        http.Error(w, "tenant required", http.StatusBadRequest)
        return
    }
    id := fmt.Sprintf("dev-%d", time.Now().UnixNano())
    labels := map[string]string{}
    for k, v := range q.Labels {
        labels[k] = fmt.Sprint(v)
    }
    now := time.Now().UTC()

    if store != nil && store.Enabled {
        dv := xdb.Device{
            ID: id, Tenant: q.Tenant, Labels: labels,
            Location: q.Location, Version: q.Version, Channel: q.Channel,
            Status: "unknown", LastSeen: now,
        }
        if err := store.UpsertDevice(r.Context(), dv); err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
    } else {
        devices[id] = &Device{ID: id, Tenant: q.Tenant, Labels: labels, LastSeen: now, Health: "unknown",
            Location: q.Location, Version: q.Version, Channel: q.Channel}
    }

    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(map[string]string{"device_id": id})
}

func heartbeatHandler(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    type hb struct {
        TS   time.Time         `json:"ts"`
        CPU  float64           `json:"cpu"`
        MEM  float64           `json:"mem"`
        Stat string            `json:"status"` // "ok"|"warn"|"crit"
        Tags map[string]string `json:"tags"`   // optional
    }
    var q hb
    if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    if q.TS.IsZero() {
        q.TS = time.Now().UTC()
    }
    status := "ok"
    if q.Stat != "" {
        status = q.Stat
    }

    if store != nil && store.Enabled {
        if err := store.UpdateHeartbeat(r.Context(), id, status, q.TS); err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
    } else {
        dv, ok := devices[id]
        if !ok {
            http.Error(w, "not found", http.StatusNotFound)
            return
        }
        dv.LastSeen = q.TS
        dv.Health = status
    }

    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
}

// dobavih gi tuk

// --- rollout details & retry ---

func getRolloutRuns(w http.ResponseWriter, r *http.Request) {
	if store == nil || !store.Enabled {
		http.Error(w, "db store required", http.StatusPreconditionFailed)
		return
	}
	id := chi.URLParam(r, "id")
	rows, err := store.ListRolloutRuns(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(rows)
}

func retryRollout(w http.ResponseWriter, r *http.Request) {
	if store == nil || !store.Enabled {
		http.Error(w, "db store required", http.StatusPreconditionFailed)
		return
	}
	id := chi.URLParam(r, "id")
	old, err := store.GetRollout(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	newID := fmt.Sprintf("ro-%d", time.Now().UnixNano())
	rec := xdb.Rollout{
		ID:        newID,
		Tenant:    old.Tenant,
		Artifact:  old.Artifact,
		Channel:   old.Channel,
		Selector:  old.Selector,
		Waves:     old.Waves,
		Status:    "draft",
		CreatedAt: time.Now().UTC(),
	}
	if err := store.CreateRollout(r.Context(), rec); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// СЃС‚Р°СЂС‚РёСЂР°РјРµ РЅРѕРІРёСЏ
	go func() {
		opt := scheduler.Options{
			WaveInterval:   parseDurationEnv("XDP47_SCHED_INTERVAL", 8*time.Second),
			HeartbeatGrace: parseDurationEnv("XDP47_SCHED_GRACE", 2*time.Minute),
			RequireOK:      parseBoolEnv("XDP47_SCHED_REQUIRE_OK", false),
			SkipOffline:    parseBoolEnv("XDP47_SCHED_SKIP_OFFLINE", true),
		}
		_ = scheduler.StartRollout(context.Background(), store, rec, opt)
	}()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"id": newID, "status": "running"})
}

func sseMetrics(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    if id == "" {
        http.Error(w, "missing id", http.StatusBadRequest)
        return
    }
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")

    flusher, ok := w.(http.Flusher)
    if !ok {
        http.Error(w, "stream unsupported", http.StatusInternalServerError)
        return
    }

    ticker := time.NewTicker(1 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-r.Context().Done():
            return
        case t := <-ticker.C:
            payload := map[string]interface{}{
                "ts":    t.UTC().Format(time.RFC3339Nano),
                "cpu":   5 + rand.Float64()*30,
                "mem":   50 + rand.Float64()*100,
                "psi":   map[string]float64{"cpu": rand.Float64() * 0.1, "io": rand.Float64() * 0.1},
                "event": randomEvent(),
            }
            b, _ := json.Marshal(payload)
            w.Write([]byte("data: "))
            w.Write(b)
            w.Write([]byte("\n\n"))
            flusher.Flush()
        }
    }
}

func randomEvent() string {
    events := []string{"", "process_exit", "tcp_retransmit", ""}
    return events[rand.Intn(len(events))]
}

// --- rollouts handlers (MVP) ---

type rolloutReq struct {
    Tenant   string            `json:"tenant"`   // required
    Artifact string            `json:"artifact"` // optional
    Channel  string            `json:"channel"`  // e.g. "dev"|"canary"|"prod"
    Selector map[string]string `json:"selector"` // match labels
    Waves    int               `json:"waves"`    // number of waves
}

func listRollouts(w http.ResponseWriter, r *http.Request) {
    if store != nil && store.Enabled {
        rows, err := store.ListRollouts(r.Context(), "")
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        w.Header().Set("Content-Type", "application/json")
        _ = json.NewEncoder(w).Encode(rows)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode([]any{})
}

func createRollout(w http.ResponseWriter, r *http.Request) {
    var q rolloutReq
    if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    if q.Tenant == "" {
        http.Error(w, "tenant required", http.StatusBadRequest)
        return
    }
    if q.Waves <= 0 {
        q.Waves = 1
    }
    if q.Selector == nil {
        q.Selector = map[string]string{}
    }
    id := fmt.Sprintf("ro-%d", time.Now().UnixNano())
    rec := xdb.Rollout{
        ID: id, Tenant: q.Tenant, Artifact: q.Artifact, Channel: q.Channel,
        Selector: q.Selector, Waves: q.Waves, Status: "draft", CreatedAt: time.Now().UTC(),
    }
    if store != nil && store.Enabled {
        if err := store.CreateRollout(r.Context(), rec); err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
    }
    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(map[string]any{"id": id, "status": "draft"})
}

func simulateRollout(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    _ = id

    // Build candidate set from devices
    var list []Device
    if store != nil && store.Enabled {
        rows, err := store.ListDevices(r.Context())
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        for _, d := range rows {
            list = append(list, Device{
                ID: d.ID, Tenant: d.Tenant, Labels: d.Labels, LastSeen: d.LastSeen,
                Health: d.Status, Location: d.Location, Version: d.Version, Channel: d.Channel,
            })
        }
    } else {
        for _, d := range devices {
            list = append(list, *d)
        }
    }

    type wave struct {
        Index     int      `json:"index"`
        DeviceIDs []string `json:"device_ids"`
    }
    plan := []wave{}

    waves := 1
    if q := r.URL.Query().Get("waves"); q != "" {
        var tmp int
        if _, err := fmt.Sscanf(q, "%d", &tmp); err == nil && tmp > 0 {
            waves = tmp
        }
    }
    buckets := make([][]string, waves)
    for i, d := range list {
        buckets[i%waves] = append(buckets[i%waves], d.ID)
    }
    for i := 0; i < waves; i++ {
        plan = append(plan, wave{Index: i + 1, DeviceIDs: buckets[i]})
    }

    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(map[string]any{
        "rollout_id":    id,
        "waves":         plan,
        "total_devices": len(list),
    })
}

// --- scheduler start ---

func startRollout(w http.ResponseWriter, r *http.Request) {
    if store == nil || !store.Enabled {
        http.Error(w, "db store required for scheduler", http.StatusPreconditionFailed)
        return
    }
    id := chi.URLParam(r, "id")
    ro, err := store.GetRollout(r.Context(), id)
    if err != nil {
        http.Error(w, err.Error(), http.StatusNotFound)
        return
    }
    go func() {
        opt := scheduler.Options{
            WaveInterval:   parseDurationEnv("XDP47_SCHED_INTERVAL", 8*time.Second),
            HeartbeatGrace: parseDurationEnv("XDP47_SCHED_GRACE", 2*time.Minute),
            RequireOK:      parseBoolEnv("XDP47_SCHED_REQUIRE_OK", false),
            SkipOffline:    parseBoolEnv("XDP47_SCHED_SKIP_OFFLINE", true),
        }
        _ = scheduler.StartRollout(context.Background(), store, ro, opt)
    }()
    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(map[string]any{"id": id, "status": "running"})
}

// --- UI ---

func uiDevices(w http.ResponseWriter, r *http.Request) {
    html := `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>XDP47 Devices</title>
  <style>
    body{font-family:system-ui,-apple-system,Segoe UI,Roboto,Ubuntu,Helvetica,Arial,sans-serif;margin:24px;background:#0b0e14;color:#e6e6e6}
    h1{font-weight:600;margin:0 0 12px}
    .muted{color:#9aa0a6}
    table{border-collapse:collapse;width:100%;margin-top:12px}
    th,td{border-bottom:1px solid #2a2f3a;padding:10px 8px;text-align:left}
    th{font-weight:600;color:#cbd5e1}
    .pill{display:inline-block;padding:2px 8px;border-radius:999px;font-size:12px}
    .ok{background:#063f30;color:#7ef0c1}
    .warn{background:#3f2e06;color:#f8d27e}
    .crit{background:#3f0606;color:#f08c7e}
    .unknown{background:#2a2f3a;color:#cbd5e1}
    .labels{font-family:ui-monospace,Consolas,monospace;font-size:12px;color:#cbd5e1}
    .toolbar{display:flex;gap:12px;align-items:center;margin-top:8px}
    .chip{border:1px solid #2a2f3a;border-radius:8px;padding:6px 10px}
  </style>
</head>
<body>
  <h1>XDP47 вЂ” Devices</h1>
  <div class="toolbar">
    <div class="chip muted">Auto-refresh: <span id="refint">5s</span></div>
    <div class="chip muted">Now: <span id="now"></span></div>
  </div>
  <table>
    <thead>
      <tr>
        <th>ID</th>
        <th>Tenant</th>
        <th>Labels</th>
        <th>Last seen</th>
        <th>Health</th>
        <th>Version</th>
        <th>Channel</th>
      </tr>
    </thead>
    <tbody id="tbody"></tbody>
  </table>
  <script>
    const $tbody = document.getElementById('tbody');
    const $now = document.getElementById('now');
    const REFRESH = 5000;
    function pill(h){
      const cls = (h||'unknown').toLowerCase();
      return '<span class="pill '+cls+'">'+(h||'unknown')+'</span>';
    }
    function fmtLabels(labels){
      if(!labels) return '';
      return '<span class="labels">'+Object.entries(labels).map(([k,v])=>k+'='+v).join(' ')+'</span>';
    }
    function fmtTime(s){
      try{ const d=new Date(s); return d.toLocaleString(); }catch(e){ return s; }
    }
    async function load(){
      $now.textContent = new Date().toLocaleTimeString();
      const res = await fetch('/api/devices');
      const arr = await res.json();
      $tbody.innerHTML = arr.map(d => (
        '<tr>'+
        '<td>'+d.id+'</td>'+
        '<td>'+d.tenant+'</td>'+
        '<td>'+fmtLabels(d.labels)+'</td>'+
        '<td>'+fmtTime(d.last_seen)+'</td>'+
        '<td>'+pill(d.health||d.status)+'</td>'+
        '<td>'+(d.version||'')+'</td>'+
        '<td>'+(d.channel||'')+'</td>'+
        '</tr>'
      )).join('');
    }
    load();
    setInterval(load, REFRESH);
  </script>
</body>
</html>`
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    _, _ = w.Write([]byte(html))
}

func uiRollouts(w http.ResponseWriter, r *http.Request) {
    html := `<!doctype html><html><head><meta charset="utf-8"><title>XDP47 Rollouts</title>
<style>
  body{font-family:system-ui;margin:24px;background:#0b0e14;color:#e6e6e6}
  table{border-collapse:collapse;width:100%;margin-top:12px}
  th,td{border-bottom:1px solid #2a2f3a;padding:8px;text-align:left}
  .muted{color:#9aa0a6}.pill{display:inline-block;padding:2px 8px;border-radius:999px;font-size:12px;background:#2a2f3a}
  button{background:#1f6feb;color:#fff;border:0;border-radius:8px;padding:6px 10px;cursor:pointer}
  button[disabled]{opacity:.6;cursor:not-allowed}
  .row-actions{display:flex;gap:8px}
  .toast{position:fixed;right:16px;bottom:16px;background:#1f2937;color:#e5e7eb;padding:10px 14px;border-radius:8px;box-shadow:0 10px 25px rgba(0,0,0,.3);opacity:0;transform:translateY(10px);transition:.2s}
  .toast.show{opacity:1;transform:translateY(0)}
</style>
</head><body><h1>Rollouts</h1>
<div class="muted">Create -> Start -> observe status.</div>
<table><thead><tr><th>ID</th><th>Tenant</th><th>Artifact</th><th>Channel</th><th>Waves</th><th>Status</th><th>Created</th><th>Actions</th></tr></thead>
<tbody id="tbody"></tbody></table>
<div id="toast" class="toast"></div>
<script>
var $tbody = document.getElementById('tbody');
var $toast = document.getElementById('toast');
function toast(msg){ $toast.textContent = msg; $toast.classList.add('show'); setTimeout(function(){ $toast.classList.remove('show'); }, 2000); }

function httpStart(id, btn){
  btn.disabled = true;
  fetch('/api/rollouts/'+id+'/start', {method:'POST'}).then(function(res){
    if(!res.ok){ return fetch('/api/rollouts/'+id+':start', {method:'POST'}); }
    return res;
  }).then(function(res){
    if(!res.ok){ return res.text().then(function(t){ throw new Error(t || ('HTTP '+res.status)); }); }
    toast('Rollout '+id+' started'); load();
  }).catch(function(e){
    toast('Error: '+e.message);
  }).finally(function(){ btn.disabled = false; });
}

function httpRetry(id, btn){
  btn.disabled = true;
  fetch('/api/rollouts/'+id+':retry', {method:'POST'}).then(function(res){
    if(!res.ok){ return res.text().then(function(t){ throw new Error(t || ('HTTP '+res.status)); }); }
    return res.json();
  }).then(function(data){
    toast('Retry created: '+data.id); load();
  }).catch(function(e){
    toast('Error: '+e.message);
  }).finally(function(){ btn.disabled = false; });
}

function details(id){
  fetch('/api/rollouts/'+id+'/runs').then(function(res){ return res.json(); }).then(function(arr){
    var rows = arr.map(function(r){
      return '<tr><td>'+r.wave_index+'</td><td>'+r.status+'</td><td>'+new Date(r.started_at).toLocaleString()+'</td><td>'+(r.finished_at?new Date(r.finished_at).toLocaleString():'')+'</td></tr>';
    }).join('');
    var modal = ''
      + '<div id="dlg" style="position:fixed;left:0;top:0;right:0;bottom:0;background:rgba(0,0,0,.4);display:flex;align-items:center;justify-content:center;">'
      + '  <div style="background:#0f172a;border:1px solid #334155;border-radius:10px;padding:16px;min-width:520px;max-width:80%;">'
      + '    <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:8px;">'
      + '      <b>Rollout '+id+' - Waves</b>'
      + '      <button onclick="document.getElementById(\'dlg\').remove()">X</button>'
      + '    </div>'
      + '    <table style="width:100%;border-collapse:collapse">'
      + '      <thead><tr><th style="text-align:left">Wave</th><th>Status</th><th>Started</th><th>Finished</th></tr></thead>'
      + '      <tbody>'+(rows || '<tr><td colspan="4">No runs yet</td></tr>')+'</tbody>'
      + '    </table>'
      + '  </div>'
      + '</div>';
    document.body.insertAdjacentHTML('beforeend', modal);
  }).catch(function(e){ toast('Error: '+e.message); });
}

function pill(s){return '<span class="pill">'+(s||'draft')+'</span>'}
function row(x){
  var id = x.id;
  return '<tr>'
    + '<td>'+id+'</td>'
    + '<td>'+x.tenant+'</td>'
    + '<td>'+(x.artifact||'')+'</td>'
    + '<td>'+(x.channel||'')+'</td>'
    + '<td>'+x.waves+'</td>'
    + '<td>'+pill(x.status)+'</td>'
    + '<td>'+new Date(x.created_at).toLocaleString()+'</td>'
    + '<td class="row-actions">'
      + '<button onclick="httpStart(\''+id+'\', this)">Start</button> '
      + '<button onclick="httpRetry(\''+id+'\', this)">Retry</button> '
      + '<button onclick="details(\''+id+'\')">Details</button>'
    + '</td>'
  + '</tr>';
}
function load(){
  fetch('/api/rollouts').then(function(r){ return r.json(); }).then(function(arr){
    $tbody.innerHTML = arr.map(row).join('');
  });
}
load(); setInterval(load, 5000);
</script></body></html>`
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    _, _ = w.Write([]byte(html))
}

