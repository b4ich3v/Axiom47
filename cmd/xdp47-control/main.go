package main

import (
    "encoding/json"
    "fmt"
    "log"
    "math/rand"
    "net/http"
    "os"
    "sort"
    "time"

    "github.com/go-chi/chi/v5"
    "github.com/go-chi/chi/v5/middleware"
)

type Device struct {
    ID       string            `json:"id"`
    Tenant   string            `json:"tenant"`
    Labels   map[string]string `json:"labels"`
    LastSeen time.Time         `json:"last_seen"`
    Health   string            `json:"health"` // "ok" | "warn" | "crit" | "unknown"
}

// naive in-memory registry
var devices = map[string]*Device{}

func main() {
    addr := os.Getenv("XDP47_LISTEN_ADDR")
    if addr == "" {
        addr = ":8080"
    }

    r := chi.NewRouter()
    r.Use(middleware.RequestID)
    r.Use(middleware.Logger)
    r.Use(middleware.Recoverer)

    r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write([]byte("ok"))
    })

    // Devices API
    r.Get("/api/devices", listDevices)
    r.Post("/api/devices/claim", claimHandler)
    r.Post("/api/devices/{id}/heartbeat", heartbeatHandler)
    r.Get("/api/devices/{id}/metrics/stream", sseMetrics)

    log.Printf("xdp47-control listening on %s", addr)
    log.Fatal(http.ListenAndServe(addr, r))
}

func listDevices(w http.ResponseWriter, r *http.Request) {
    type devOut struct {
        ID       string            `json:"id"`
        Tenant   string            `json:"tenant"`
        Labels   map[string]string `json:"labels"`
        LastSeen time.Time         `json:"last_seen"`
        Health   string            `json:"health"`
    }
    out := make([]devOut, 0, len(devices))
    for _, d := range devices {
        out = append(out, devOut{
            ID: d.ID, Tenant: d.Tenant, Labels: d.Labels, LastSeen: d.LastSeen, Health: d.Health,
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
        Tenant string                 `json:"tenant"`
        Labels map[string]interface{} `json:"labels"`
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
    d := &Device{ID: id, Tenant: q.Tenant, Labels: labels, LastSeen: time.Now(), Health: "unknown"}
    devices[id] = d

    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(map[string]string{"device_id": id})
}

func heartbeatHandler(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    dv, ok := devices[id]
    if !ok {
        http.Error(w, "not found", http.StatusNotFound)
        return
    }
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
    dv.LastSeen = q.TS
    if q.Stat != "" {
        dv.Health = q.Stat
    }
    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
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

            // Write SSE frame safely in chunks (avoid quoted newlines)
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