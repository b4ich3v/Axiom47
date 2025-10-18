package db

import (
    "context"
    "encoding/json"
    "fmt"
    "time"
)

// Rollout represents a simple rollout plan persisted in DB.
type Rollout struct {
    ID        string                 `json:"id"`
    Tenant    string                 `json:"tenant"`
    Artifact  string                 `json:"artifact"`
    Channel   string                 `json:"channel"`
    Selector  map[string]string      `json:"selector"` // match labels
    Waves     int                    `json:"waves"`
    Status    string                 `json:"status"`   // draft|running|paused|completed|failed
    CreatedAt time.Time              `json:"created_at"`
}

// MigrateRollouts ensures the rollouts table exists.
func (s *Store) MigrateRollouts(ctx context.Context) error {
    if s == nil || !s.Enabled { return fmt.Errorf("store disabled") }
    sql := `
    CREATE TABLE IF NOT EXISTS rollouts (
        id TEXT PRIMARY KEY,
        tenant TEXT NOT NULL,
        artifact TEXT,
        channel TEXT,
        selector JSONB,
        waves INT,
        status TEXT,
        created_at TIMESTAMPTZ DEFAULT now()
    );
    CREATE INDEX IF NOT EXISTS idx_rollouts_tenant ON rollouts(tenant);
    `
    _, err := s.pool.Exec(ctx, sql)
    return err
}

func (s *Store) CreateRollout(ctx context.Context, r Rollout) error {
    if s == nil || !s.Enabled { return fmt.Errorf("store disabled") }
    sel, _ := json.Marshal(r.Selector)
    _, err := s.pool.Exec(ctx, `
        INSERT INTO rollouts (id, tenant, artifact, channel, selector, waves, status, created_at)
        VALUES ($1,$2,$3,$4,$5,$6,$7,COALESCE($8, now()));
    `, r.ID, r.Tenant, r.Artifact, r.Channel, sel, r.Waves, r.Status, r.CreatedAt)
    return err
}

func (s *Store) ListRollouts(ctx context.Context, tenant string) ([]Rollout, error) {
    if s == nil || !s.Enabled { return nil, fmt.Errorf("store disabled") }
    rows, err := s.pool.Query(ctx, `
        SELECT id, tenant, artifact, channel, selector, waves, status, created_at
        FROM rollouts
        WHERE ($1 = '' OR tenant = $1)
        ORDER BY created_at DESC, id ASC;
    `, tenant)
    if err != nil { return nil, err }
    defer rows.Close()
    var out []Rollout
    for rows.Next() {
        var r Rollout
        var sel []byte
        if err := rows.Scan(&r.ID, &r.Tenant, &r.Artifact, &r.Channel, &sel, &r.Waves, &r.Status, &r.CreatedAt); err != nil { return nil, err }
        if sel != nil { _ = json.Unmarshal(sel, &r.Selector) }
        out = append(out, r)
    }
    return out, rows.Err()
}
