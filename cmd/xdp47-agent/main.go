package main

import (
    "bytes"
    "encoding/json"
    "log"
    "math/rand"
    "net/http"
    "os"
    "strings"
    "time"
)

func getenv(k, def string) string {
    v := os.Getenv(k)
    if v == "" { return def }
    return v
}

func main() {
    control := getenv("XDP47_CONTROL_URL", "http://127.0.0.1:8080")
    tenant := getenv("XDP47_TENANT", "demo-tenant")
    deviceID := getenv("XDP47_DEVICE_ID", "")
    labelStr := getenv("XDP47_DEVICE_LABELS", "store=demo,role=kiosk")

    labels := map[string]string{}
    for _, kv := range strings.Split(labelStr, ",") {
        if kv == "" { continue }
        parts := strings.SplitN(kv, "=", 2)
        if len(parts) == 2 {
            labels[parts[0]] = parts[1]
        }
    }

    if deviceID == "" {
        // claim
        body := map[string]interface{}{"tenant": tenant, "labels": labels}
        buf, _ := json.Marshal(body)
        resp, err := http.Post(control+"/api/devices/claim", "application/json", bytes.NewReader(buf))
        if err != nil { log.Fatalf("claim error: %v", err) }
        defer resp.Body.Close()
        var out map[string]string
        json.NewDecoder(resp.Body).Decode(&out)
        deviceID = out["device_id"]
        if deviceID == "" { log.Fatal("empty device_id after claim") }
        log.Printf("claimed device_id=%s", deviceID)
    }

    // heartbeat loop (every 5s)
    client := &http.Client{ Timeout: 5 * time.Second }
    for {
        hb := map[string]interface{}{
            "ts":   time.Now().UTC().Format(time.RFC3339Nano),
            "cpu":  5 + rand.Float64()*30,
            "mem":  50 + rand.Float64()*100,
            "status": "ok",
            "tags": map[string]string{"agent":"xdp47"},
        }
        buf, _ := json.Marshal(hb)
        req, _ := http.NewRequest("POST", control+"/api/devices/"+deviceID+"/heartbeat", bytes.NewReader(buf))
        req.Header.Set("Content-Type", "application/json")
        resp, err := client.Do(req)
        if err != nil {
            log.Printf("heartbeat error: %v", err)
        } else {
            resp.Body.Close()
        }
        time.Sleep(5 * time.Second)
    }
}
