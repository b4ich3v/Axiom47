package scheduler

import (
    "context"
    "fmt"
    "log"
    "time"

    xdb "github.com/example/xdp47/internal/db"
)

type Options struct {
    WaveInterval   time.Duration // time between waves (e.g. 8s)
    HeartbeatGrace time.Duration // consider offline if older (e.g. 2m)
    RequireOK      bool          // fail device if status != ok
    SkipOffline    bool          // if true, skip offline devices instead of failing wave
}

// StartRollout executes waves sequentially based on selector.
// For each OK device in a wave, it applies rollout artifact/channel to the device.
func StartRollout(ctx context.Context, store *xdb.Store, rollout xdb.Rollout, opt Options) error {
    if store == nil || !store.Enabled {
        return fmt.Errorf("scheduler requires db store")
    }
    devs, err := store.FilterDevicesBySelector(ctx, rollout.Tenant, rollout.Selector)
    if err != nil {
        return err
    }
    if len(devs) == 0 {
        log.Printf("[sched] rollout %s: no matching devices", rollout.ID)
        _ = store.UpdateRolloutStatus(ctx, rollout.ID, "failed")
        return nil
    }
    waves := rollout.Waves
    if waves <= 0 {
        waves = 1
    }

    // split devices into waves round-robin
    buckets := make([][]string, waves)
    for i, d := range devs {
        buckets[i%waves] = append(buckets[i%waves], d.ID)
    }
    if err := store.UpdateRolloutStatus(ctx, rollout.ID, "running"); err != nil {
        return err
    }

    for wi := 0; wi < waves; wi++ {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
        }
        waveID := fmt.Sprintf("run-%s-%d", rollout.ID, wi+1)
        now := time.Now().UTC()
        _ = store.InsertRolloutRun(ctx, waveID, rollout.ID, wi+1, buckets[wi], "running", &now)

        applied := 0
        anyFailed := false

        for _, id := range buckets[wi] {
            // lookup device by id
            var dv *xdb.Device
            for i := range devs {
                if devs[i].ID == id {
                    dv = &devs[i]
                    break
                }
            }
            if dv == nil {
                continue
            }
            offline := dv.LastSeen.IsZero() || time.Since(dv.LastSeen) > opt.HeartbeatGrace
            badStatus := opt.RequireOK && dv.Status != "ok"

            if offline {
                if opt.SkipOffline {
                    log.Printf("[sched] rollout %s wave %d: device %s SKIPPED (offline)", rollout.ID, wi+1, id)
                    continue
                }
                anyFailed = true
                log.Printf("[sched] rollout %s wave %d: device %s FAILED (offline)", rollout.ID, wi+1, id)
                continue
            }
            if badStatus {
                anyFailed = true
                log.Printf("[sched] rollout %s wave %d: device %s FAILED (status=%s)", rollout.ID, wi+1, id, dv.Status)
                continue
            }

            // APPLY version/channel
            if err := store.ApplyVersionChannel(ctx, dv.ID, rollout.Artifact, rollout.Channel); err != nil {
                anyFailed = true
                log.Printf("[sched] rollout %s wave %d: device %s APPLY ERROR: %v", rollout.ID, wi+1, dv.ID, err)
                continue
            }
            applied++
            log.Printf("[sched] rollout %s wave %d: device %s APPLY OK (version=%s, channel=%s)",
                rollout.ID, wi+1, dv.ID, rollout.Artifact, rollout.Channel)
        }

        // критерий за вълна:
        // - ако сме приложили поне на 1 устройство и няма критични грешки -> completed
        // - ако нито едно не е приложено -> failed
        if applied > 0 && !anyFailed {
            _ = store.CompleteRolloutRun(ctx, waveID, "completed", time.Now().UTC())
        } else if applied > 0 && anyFailed {
            _ = store.CompleteRolloutRun(ctx, waveID, "partial", time.Now().UTC())
        } else {
            _ = store.CompleteRolloutRun(ctx, waveID, "failed", time.Now().UTC())
            _ = store.UpdateRolloutStatus(ctx, rollout.ID, "failed")
            return nil
        }

        if wi < waves-1 {
            select {
            case <-ctx.Done():
                return ctx.Err()
            case <-time.After(opt.WaveInterval):
            }
        }
    }

    _ = store.UpdateRolloutStatus(ctx, rollout.ID, "completed")
    return nil
}