package booking

import (
	"context"
	"errors"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	Available = "available"
	Booked    = "booked"
	Past      = "past"

	DateLayout  = "2006-01-02"
	invalidSlot = "Укажите существующий слот"
)

var (
	ErrSlotNotFound    = errors.New("slot not found")
	ErrSlotPast        = errors.New("slot has started")
	ErrSlotBooked      = errors.New("slot is booked")
	ErrBookingNotFound = errors.New("booking not found")
)

type Slot struct {
	ID        int64
	StartsAt  time.Time
	EndsAt    time.Time
	BookingID int64
}

type ScheduledSlot struct {
	Slot
	Status string
}

type Schedule struct {
	TimeZone string
	Today    string
	Slots    []ScheduledSlot
}

type BookingInput struct {
	SlotID int64
	Name   string
	Email  string
}

type ValidationError map[string][]string

func (ValidationError) Error() string { return "invalid booking fields" }

type Store interface {
	ListSlots(context.Context) ([]Slot, error)
	SlotExists(context.Context, int64) (bool, error)
	CreateBooking(context.Context, BookingInput, time.Time) (int64, error)
	DeleteBooking(context.Context, int64) error
}

type Service struct {
	store    Store
	location *time.Location
	now      func() time.Time
}

func NewService(store Store, location *time.Location, now func() time.Time) *Service {
	return &Service{store: store, location: location, now: now}
}

func (s *Service) ListSlots(ctx context.Context) (Schedule, error) {
	slots, err := s.store.ListSlots(ctx)
	if err != nil {
		return Schedule{}, err
	}

	now := s.now()
	schedule := Schedule{
		TimeZone: s.location.String(),
		Today:    now.In(s.location).Format(DateLayout),
		Slots:    make([]ScheduledSlot, 0, len(slots)),
	}
	for _, slot := range slots {
		status := Available
		switch {
		case !slot.StartsAt.After(now):
			status = Past
		case slot.BookingID != 0:
			status = Booked
		}
		slot.StartsAt = slot.StartsAt.In(s.location)
		slot.EndsAt = slot.EndsAt.In(s.location)
		schedule.Slots = append(schedule.Slots, ScheduledSlot{Slot: slot, Status: status})
	}
	return schedule, nil
}

func (s *Service) CreateBooking(ctx context.Context, input BookingInput) (int64, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Email = strings.TrimSpace(input.Email)
	fields := ValidationError{}
	if input.SlotID <= 0 {
		fields["slot_id"] = []string{invalidSlot}
	}
	if input.Name == "" || utf8.RuneCountInString(input.Name) > 120 {
		fields["customer_name"] = []string{"Введите имя длиной от 1 до 120 символов"}
	}
	email, emailErr := mail.ParseAddress(input.Email)
	if emailErr != nil || email.Address != input.Email || len(input.Email) > 254 {
		fields["customer_email"] = []string{"Введите корректный email"}
	}
	if len(fields) != 0 {
		if input.SlotID > 0 {
			exists, err := s.store.SlotExists(ctx, input.SlotID)
			if err != nil {
				return 0, err
			}
			if !exists {
				fields["slot_id"] = []string{invalidSlot}
			}
		}
		return 0, fields
	}

	id, err := s.store.CreateBooking(ctx, input, s.now())
	if errors.Is(err, ErrSlotNotFound) {
		return 0, ValidationError{"slot_id": {invalidSlot}}
	}
	return id, err
}

func (s *Service) CancelBooking(ctx context.Context, id int64) error {
	return s.store.DeleteBooking(ctx, id)
}
