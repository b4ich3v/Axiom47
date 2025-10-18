package db

import (
    "context"
    "database/sql"
    "errors"
    "time"
)

// Минимален модел за детайли на вълните
type RolloutRun struct {
    RolloutID  string     `json:"rollout_id"`
    WaveIndex  int        `json:"wave_index"`
    Status     string     `json:"status"`
    StartedAt  time.Time  `json:"started_at"`
    FinishedAt *time.Time `json:"finished_at,omitempty"`
}

// ========== DETAILS (history) ==========

func (s *Store) ListRolloutRuns(ctx context.Context, rolloutID string) ([]RolloutRun, error) {
    if s == nil || !s.Enabled {
        return []RolloutRun{}, nil
    }
    rows, err := s.pool.Query(ctx, `
        SELECT rollout_id, wave_index, status, started_at, finished_at
        FROM rollout_runs
        WHERE rollout_id = $1
        ORDER BY wave_index ASC`, rolloutID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    out := make([]RolloutRun, 0, 8)
    for rows.Next() {
        var r RolloutRun
        var finished sql.NullTime
        if err := rows.Scan(&r.RolloutID, &r.WaveIndex, &r.Status, &r.StartedAt, &finished); err != nil {
            return nil, err
        }
        if finished.Valid {
            t := finished.Time
            r.FinishedAt = &t
        }
        out = append(out, r)
    }
    return out, rows.Err()
}

func (s *Store) CompleteRolloutRun(ctx context.Context, runID, status string, finished time.Time) error {
    if s == nil || !s.Enabled {
        return nil
    }
    _, err := s.pool.Exec(ctx, `
        UPDATE rollout_runs SET status = $1, finished_at = $2
        WHERE id = $3`, status, finished, runID)
    return err
}

// ========== METHODS, които очаква scheduler ==========

// FilterDevicesBySelector: практичен филтър по tenant + selector (labels JSONB).
// Поддържаме (tenant, selector) или само (selector).
func (s *Store) FilterDevicesBySelector(ctx context.Context, args ...any) ([]Device, error) {
    if s == nil || !s.Enabled {
        return []Device{}, errors.New("store disabled")
    }
    var tenant string
    var selector map[string]string

    if len(args) == 1 {
        if m, ok := args[0].(map[string]string); ok {
            selector = m
        }
    } else if len(args) >= 2 {
        if t, ok := args[0].(string); ok {
            tenant = t
        }
        if m, ok := args[1].(map[string]string); ok {
            selector = m
        }
    }
    if selector == nil {
        selector = map[string]string{}
    }

    // Ще мачнем по всяка двойка key=value от selector в labels JSONB (@>)
    type kv struct{ k, v string }
    var pairs []kv
    for k, v := range selector {
        pairs = append(pairs, kv{k, v})
    }

    q := `
        SELECT id, tenant, labels, location, version, channel, status, last_seen, created_at
        FROM devices
        WHERE ($1 = '' OR tenant = $1)
    `
    // динамично изграждаме AND условията
    params := []any{tenant}
    i := 2
    for _, p := range pairs {
        q += "\n  AND (labels ->> $" + itoa(i) + ") = $" + itoa(i+1)
        params = append(params, p.k, p.v)
        i += 2
    }
    q += "\nORDER BY last_seen DESC NULLS LAST, id ASC"

    rows, err := s.pool.Query(ctx, q, params...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var out []Device
    for rows.Next() {
        var d Device
        if err := rows.Scan(&d.ID, &d.Tenant, &d.Labels, &d.Location, &d.Version, &d.Channel, &d.Status, &d.LastSeen, &d.CreatedAt); err != nil {
            return nil, err
        }
        out = append(out, d)
    }
    return out, rows.Err()
}

// UpdateRolloutStatus: обновява статуса (и по желание finished_at).
func (s *Store) UpdateRolloutStatus(ctx context.Context, args ...any) error {
    if s == nil || !s.Enabled {
        return errors.New("store disabled")
    }
    if len(args) < 2 {
        return errors.New("missing args")
    }
    id, _ := args[0].(string)
    status, _ := args[1].(string)
    var finishedAt *time.Time
    if len(args) >= 3 {
        if t, ok := args[2].(*time.Time); ok {
            finishedAt = t
        }
    }
    if id == "" {
        return errors.New("missing rollout id")
    }

    if finishedAt != nil {
        _, err := s.pool.Exec(ctx, `UPDATE rollouts SET status=$1, finished_at=$2 WHERE id=$3`, status, *finishedAt, id)
        return err
    }
    _, err := s.pool.Exec(ctx, `UPDATE rollouts SET status=$1 WHERE id=$2`, status, id)
    return err
}

// InsertRolloutRun: съвместима с извикването от scheduler – връща само error.
// Очакваме най-често: (runID string, rolloutID string, waveIndex int, ... , status string, startedAt *time.Time)
func (s *Store) InsertRolloutRun(ctx context.Context, args ...any) error {
    if s == nil || !s.Enabled {
        return errors.New("store disabled")
    }
    if len(args) < 5 {
        return errors.New("missing args")
    }
    runID, _ := args[0].(string)
    rolloutID, _ := args[1].(string)
    waveIndex, _ := args[2].(int)

    // status и startedAt най-често са последните 2 аргумента
    var status string
    var startedAt time.Time
    if len(args) >= 5 {
        if s1, ok := args[len(args)-2].(string); ok {
            status = s1
        }
        if tptr, ok := args[len(args)-1].(*time.Time); ok && tptr != nil {
            startedAt = *tptr
        }
    }
    if startedAt.IsZero() {
        startedAt = time.Now().UTC()
    }
    if runID == "" || rolloutID == "" || waveIndex <= 0 || status == "" {
        return errors.New("invalid InsertRolloutRun args")
    }

    _, err := s.pool.Exec(ctx, `
        INSERT INTO rollout_runs (id, rollout_id, wave_index, status, started_at)
        VALUES ($1,$2,$3,$4,$5)`,
        runID, rolloutID, waveIndex, status, startedAt)
    return err
}

// малки помощници

func itoa(i int) string {
    // без strconv, за да държим импорти минимални
    if i == 0 {
        return "0"
    }
    neg := false
    if i < 0 {
        neg = true
        i = -i
    }
    var b [20]byte
    pos := len(b)
    for i > 0 {
        pos--
        b[pos] = byte('0' + i%10)
        i /= 10
    }
    if neg {
        pos--
        b[pos] = '-'
    }
    return string(b[pos:])
}