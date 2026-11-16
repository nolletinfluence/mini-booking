package sqlite

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"

	"minibooking/backend/internal/booking"
)

//go:embed migrations/001_slots.sql
var createSlots string

//go:embed migrations/002_bookings.sql
var createBookings string

var migrations = [...]string{createSlots, createBookings}

func (s *Store) Migrate(ctx context.Context) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin migration: %w", err)
	}
	defer tx.Rollback()

	version, err := schemaVersion(ctx, tx)
	if err != nil {
		return err
	}
	if version < 0 || version > len(migrations) {
		return fmt.Errorf("unsupported schema version: %d", version)
	}
	for index := version; index < len(migrations); index++ {
		if _, err := tx.ExecContext(ctx, migrations[index]); err != nil {
			return fmt.Errorf("apply migration %d: %w", index+1, err)
		}
	}
	if version != len(migrations) {
		if _, err := tx.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version = %d", len(migrations))); err != nil {
			return fmt.Errorf("set schema version: %w", err)
		}
	}
	return tx.Commit()
}

func (s *Store) CheckSchema(ctx context.Context) error {
	version, err := schemaVersion(ctx, s.db)
	if err != nil {
		return err
	}
	if version != len(migrations) {
		return fmt.Errorf("schema version %d, expected %d; run booking migrate", version, len(migrations))
	}
	return nil
}

func schemaVersion(ctx context.Context, db interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}) (int, error) {
	var version int
	if err := db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		return 0, fmt.Errorf("read schema version: %w", err)
	}
	return version, nil
}

func (s *Store) SeedSlots(ctx context.Context, slots []booking.Slot) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin seed: %w", err)
	}
	defer tx.Rollback()

	const insert = `
		INSERT INTO slots (starts_at, ends_at) VALUES (?, ?)
		ON CONFLICT(starts_at) DO NOTHING
	`
	for _, slot := range slots {
		if _, err := tx.ExecContext(ctx, insert, slot.StartsAt.Unix(), slot.EndsAt.Unix()); err != nil {
			return fmt.Errorf("seed slot %s: %w", slot.StartsAt, err)
		}
	}
	return tx.Commit()
}
