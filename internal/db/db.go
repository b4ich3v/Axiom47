package db

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "time"

    "github.com/jackc/pgx/v5/pgxpool"
)

// Device mirrors the control-plane view and is stored in Postgres.
type Device struct {
    ID       string            `json:"id"`
    Tenant   string            `json:"tenant"`
    Labels   map[string]string `json:"labels"`
    Location string            `json:"location"`
    Version  string            `json:"version"`
    Channel  string            `json:"channel"`
    Status   string            `json:"status"`   // aka health
    LastSeen time.Time         `json:"last_seen"`
    CreatedAt time.Time        `json:"created_at"`
}

type Store struct {
    pool    *pgxpool.Pool
    Enabled bool
}

// Connect creates a pgx pool with sane defaults and validates the connection.
func Connect(ctx context.Context, url string) (*Store, error) {
    cfg, err := pgxpool.ParseConfig(url)
    if err != nil {
        return nil, fmt.Errorf("parse config: %w", err)
    }
    // Reasonable defaults
    cfg.MaxConns = 10
    cfg.MinConns = 0
    cfg.HealthCheckPeriod = 30 * time.Second
    cfg.MaxConnLifetime = 30 * time.Minute
    cfg.MaxConnIdleTime = 5 * time.Minute

    pool, err := pgxpool.NewWithConfig(ctx, cfg)
    if err != nil {
        return nil, fmt.Errorf("new pool: %w", err)
    }
    // Validate
    pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()
    if err := pool.Ping(pingCtx); err != nil {
        pool.Close()
        return nil, fmt.Errorf("ping db: %w", err)
    }
    s := &Store{pool: pool, Enabled: true}
    return s, nil
}

func (s *Store) Close() {
    if s != nil && s.pool != nil {
        s.pool.Close()
    }
}

// Migrate ensures the devices table exists.
func (s *Store) Migrate(ctx context.Context) error {
    if s == nil || !s.Enabled {
        return errors.New("store disabled")
    }
    sql := `
    CREATE TABLE IF NOT EXISTS devices (
        id TEXT PRIMARY KEY,
        tenant TEXT NOT NULL,
        labels JSONB,
        location TEXT,
        version TEXT,
        channel TEXT,
        status TEXT,
        last_seen TIMESTAMPTZ,
        created_at TIMESTAMPTZ DEFAULT now()
    );
    CREATE INDEX IF NOT EXISTS idx_devices_tenant ON devices(tenant);
    CREATE INDEX IF NOT EXISTS idx_devices_last_seen ON devices(last_seen DESC);
    `
    ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
    defer cancel()
    _, err := s.pool.Exec(ctx, sql)
    if err != nil {
        return fmt.Errorf("migrate: %w", err)
    }
    return nil
}

// UpsertDevice inserts or updates a device row.
func (s *Store) UpsertDevice(ctx context.Context, d Device) error {
    if s == nil || !s.Enabled {
        return errors.New("store disabled")
    }
    lb, _ := json.Marshal(d.Labels)
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()
    _, err := s.pool.Exec(ctx, `
        INSERT INTO devices (id, tenant, labels, location, version, channel, status, last_seen)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
        ON CONFLICT (id) DO UPDATE SET
            tenant=EXCLUDED.tenant,
            labels=EXCLUDED.labels,
            location=EXCLUDED.location,
            version=EXCLUDED.version,
            channel=EXCLUDED.channel,
            status=EXCLUDED.status,
            last_seen=EXCLUDED.last_seen;
    `, d.ID, d.Tenant, lb, d.Location, d.Version, d.Channel, d.Status, d.LastSeen)
    if err != nil {
        return fmt.Errorf("upsert device: %w", err)
    }
    return nil
}

// UpdateHeartbeat updates status and last_seen for a device.
func (s *Store) UpdateHeartbeat(ctx context.Context, id string, status string, ts time.Time) error {
    if s == nil || !s.Enabled {
        return errors.New("store disabled")
    }
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()
    _, err := s.pool.Exec(ctx, `
        UPDATE devices SET status=$1, last_seen=$2 WHERE id=$3;
    `, status, ts, id)
    if err != nil {
        return fmt.Errorf("update heartbeat: %w", err)
    }
    return nil
}

// ListDevices returns devices ordered by last_seen desc.
func (s *Store) ListDevices(ctx context.Context) ([]Device, error) {
    if s == nil || !s.Enabled {
        return nil, errors.New("store disabled")
    }
    ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
    defer cancel()
    rows, err := s.pool.Query(ctx, `
        SELECT id, tenant, labels, location, version, channel, status, last_seen, created_at
        FROM devices ORDER BY last_seen DESC NULLS LAST, id ASC;
    `)
    if err != nil {
        return nil, fmt.Errorf("list devices: %w", err)
    }
    defer rows.Close()

    out := []Device{}
    for rows.Next() {
        var d Device
        var lb []byte
        if err := rows.Scan(&d.ID, &d.Tenant, &lb, &d.Location, &d.Version, &d.Channel, &d.Status, &d.LastSeen, &d.CreatedAt); err != nil {
            return nil, fmt.Errorf("scan: %w", err)
        }
        if lb != nil {
            _ = json.Unmarshal(lb, &d.Labels)
        }
        out = append(out, d)
    }
    return out, rows.Err()
}
