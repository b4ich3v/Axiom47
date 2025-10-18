package db

import (
    "context"
    "errors"
)

// ApplyVersionChannel sets version and channel for a device.
func (s *Store) ApplyVersionChannel(ctx context.Context, deviceID, version, channel string) error {
    if s == nil || !s.Enabled {
        return errors.New("store disabled")
    }
    _, err := s.pool.Exec(ctx, `
        UPDATE devices
        SET version = $1, channel = $2
        WHERE id = $3;
    `, version, channel, deviceID)
    return err
}