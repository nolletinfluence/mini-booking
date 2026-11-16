package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"minibooking/backend/internal/booking"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func Open(ctx context.Context, path string) (*Store, error) {
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve database path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(absolutePath), 0755); err != nil {
		return nil, fmt.Errorf("create database directory: %w", err)
	}

	params := url.Values{"_pragma": {"busy_timeout(5000)", "foreign_keys(1)"}}
	dsn := url.URL{Scheme: "file", Path: absolutePath, RawQuery: params.Encode()}
	db, err := sql.Open("sqlite", dsn.String())
	if err != nil {
		return nil, fmt.Errorf("open SQLite: %w", err)
	}
	db.SetMaxOpenConns(4)
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("connect to SQLite: %w", err)
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) ListSlots(ctx context.Context) ([]booking.Slot, error) {
	const query = `
		SELECT slots.id, slots.starts_at, slots.ends_at, bookings.id
		FROM slots
		LEFT JOIN bookings ON bookings.slot_id = slots.id
		ORDER BY slots.starts_at
	`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list slots: %w", err)
	}
	defer rows.Close()

	var slots []booking.Slot
	for rows.Next() {
		var slot booking.Slot
		var startsAt, endsAt int64
		var bookingID sql.NullInt64
		if err := rows.Scan(&slot.ID, &startsAt, &endsAt, &bookingID); err != nil {
			return nil, fmt.Errorf("read slot: %w", err)
		}
		slot.StartsAt = time.Unix(startsAt, 0).UTC()
		slot.EndsAt = time.Unix(endsAt, 0).UTC()
		slot.BookingID = bookingID.Int64
		slots = append(slots, slot)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate slots: %w", err)
	}
	return slots, nil
}

func (s *Store) SlotExists(ctx context.Context, id int64) (bool, error) {
	var exists bool
	if err := s.db.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM slots WHERE id = ?)", id).Scan(&exists); err != nil {
		return false, fmt.Errorf("check slot existence: %w", err)
	}
	return exists, nil
}

func (s *Store) CreateBooking(ctx context.Context, input booking.BookingInput, now time.Time) (int64, error) {
	const insert = `
		INSERT INTO bookings (slot_id, customer_name, customer_email)
		SELECT id, ?, ? FROM slots WHERE id = ? AND starts_at > ?
		ON CONFLICT(slot_id) DO NOTHING
		RETURNING id
	`
	var id int64
	err := s.db.QueryRowContext(ctx, insert, input.Name, input.Email, input.SlotID, now.Unix()).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("insert booking: %w", err)
	}

	var startsAt int64
	err = s.db.QueryRowContext(ctx, "SELECT starts_at FROM slots WHERE id = ?", input.SlotID).Scan(&startsAt)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return 0, booking.ErrSlotNotFound
	case err != nil:
		return 0, fmt.Errorf("read slot after insert: %w", err)
	case startsAt <= now.Unix():
		return 0, booking.ErrSlotPast
	default:
		return 0, booking.ErrSlotBooked
	}
}

func (s *Store) DeleteBooking(ctx context.Context, id int64) error {
	result, err := s.db.ExecContext(ctx, "DELETE FROM bookings WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete booking: %w", err)
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("confirm booking deletion: %w", err)
	}
	if deleted == 0 {
		return booking.ErrBookingNotFound
	}
	return nil
}
