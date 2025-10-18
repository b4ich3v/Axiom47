package db

import (
    "context"
    "encoding/json"
    "time"
)

// MigrateScheduler създава таблицата за history на вълните при rollout-и.
func (s *Store) MigrateScheduler(ctx context.Context) error {
    if s == nil || !s.Enabled {
        return nil
    }
    sql := `
CREATE TABLE IF NOT EXISTS rollout_runs (
    id          TEXT PRIMARY KEY,
    rollout_id  TEXT NOT NULL REFERENCES rollouts(id) ON DELETE CASCADE,
    wave_index  INT  NOT NULL,
    status      TEXT NOT NULL,
    started_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at TIMESTAMPTZ NULL
);
CREATE INDEX IF NOT EXISTS idx_rollout_runs_ro ON rollout_runs(rollout_id);
CREATE INDEX IF NOT EXISTS idx_rollout_runs_ro_wave ON rollout_runs(rollout_id, wave_index);
`
    _, err := s.pool.Exec(ctx, sql)
    return err
}

// GetRollout връща един rollout по ID.
func (s *Store) GetRollout(ctx context.Context, id string) (Rollout, error) {
    var out Rollout
    if s == nil || !s.Enabled {
        return out, ErrStoreDisabled
    }
    var selBytes []byte
    // Включваме created_at; finished_at е по желание (може да липсва в модела)
    err := s.pool.QueryRow(ctx, `
        SELECT id, tenant, artifact, channel, selector, waves, status, created_at
        FROM rollouts
        WHERE id = $1
    `, id).Scan(
        &out.ID, &out.Tenant, &out.Artifact, &out.Channel,
        &selBytes, &out.Waves, &out.Status, &out.CreatedAt,
    )
    if err != nil {
        return Rollout{}, err
    }
    if len(selBytes) > 0 {
        _ = json.Unmarshal(selBytes, &out.Selector)
    } else if out.Selector == nil {
        out.Selector = map[string]string{}
    }
    return out, nil
}

// Лека помощна грешка за по-ясни съобщения.
var ErrStoreDisabled = &storeDisabledError{}

type storeDisabledError struct{}

func (*storeDisabledError) Error() string { return "store disabled" }

// (по желание) удобен хелпър – връща pointer към time за UpdateRolloutStatus
func timePtr(t time.Time) *time.Time { return &t }