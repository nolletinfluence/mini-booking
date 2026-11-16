package httpapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"minibooking/backend/internal/booking"
	"minibooking/backend/internal/httpapi"
	"minibooking/backend/internal/sqlite"
)

type fixture struct {
	store    *sqlite.Store
	handler  http.Handler
	location *time.Location
	now      time.Time
}

type listedSlot struct {
	ID        int64     `json:"id"`
	StartsAt  time.Time `json:"starts_at"`
	EndsAt    time.Time `json:"ends_at"`
	Status    string    `json:"status"`
	BookingID int64     `json:"booking_id"`
}

type listedSchedule struct {
	TimeZone string       `json:"time_zone"`
	Today    string       `json:"today"`
	Slots    []listedSlot `json:"slots"`
}

func TestCreateBooking(t *testing.T) {
	f := newFixture(t)
	slot := getSchedule(t, f.handler).Slots[1]
	if slot.Status != "available" {
		t.Fatalf("status = %q, want available", slot.Status)
	}

	response := postBooking(f.handler, slot.ID)
	assertStatus(t, response, http.StatusCreated)
	var created struct{ ID int64 }
	decodeResponse(t, response, &created)
	if created.ID == 0 {
		t.Fatal("booking ID is missing")
	}

	response = serve(f.handler, http.MethodGet, "/api/slots", nil)
	assertStatus(t, response, http.StatusOK)
	if strings.Contains(response.Body.String(), "Тест") || strings.Contains(response.Body.String(), "example.com") {
		t.Fatal("private booking data must not appear in the schedule")
	}
	var schedule listedSchedule
	decodeResponse(t, response, &schedule)
	if got := schedule.Slots[1]; got.Status != "booked" || got.BookingID != created.ID {
		t.Fatalf("listed booking = %+v, want booked with ID %d", got, created.ID)
	}

	path := "/api/bookings/" + strconv.FormatInt(created.ID, 10)
	assertStatus(t, serve(f.handler, http.MethodDelete, path, nil), http.StatusNoContent)
	if got := getSchedule(t, f.handler).Slots[1]; got.Status != "available" || got.BookingID != 0 {
		t.Fatalf("cancelled slot = %+v, want available without booking", got)
	}
	response = postBooking(f.handler, slot.ID)
	assertStatus(t, response, http.StatusCreated)
	var rebooked struct{ ID int64 }
	decodeResponse(t, response, &rebooked)
	if rebooked.ID <= created.ID {
		t.Fatalf("replacement booking ID = %d, must be greater than %d", rebooked.ID, created.ID)
	}
	assertStatus(t, serve(f.handler, http.MethodDelete, path, nil), http.StatusNotFound)
	if got := getSchedule(t, f.handler).Slots[1]; got.BookingID != rebooked.ID {
		t.Fatalf("booking after repeated cancellation = %d, want %d", got.BookingID, rebooked.ID)
	}
}

func TestRejectsSecondBooking(t *testing.T) {
	f := newFixture(t)
	slotID := getSchedule(t, f.handler).Slots[1].ID

	first := postBooking(f.handler, slotID)
	assertStatus(t, first, http.StatusCreated)
	var created struct{ ID int64 }
	decodeResponse(t, first, &created)
	assertStatus(t, postBooking(f.handler, slotID), http.StatusConflict)
	assertSingleBooking(t, f.handler, slotID, created.ID)
}

func TestConcurrentBooking(t *testing.T) {
	f := newFixture(t)
	slotID := getSchedule(t, f.handler).Slots[1].ID

	const attempts = 12
	start := make(chan struct{})
	responses := make(chan *httptest.ResponseRecorder, attempts)
	for range attempts {
		go func() {
			<-start
			responses <- postBooking(f.handler, slotID)
		}()
	}
	close(start)

	var successes, conflicts int
	var created struct{ ID int64 }
	for range attempts {
		response := <-responses
		switch response.Code {
		case http.StatusCreated:
			successes++
			decodeResponse(t, response, &created)
		case http.StatusConflict:
			conflicts++
		default:
			t.Errorf("unexpected response %d: %s", response.Code, response.Body)
		}
	}
	if successes != 1 || conflicts != attempts-1 {
		t.Fatalf("responses: %d created and %d conflicts, want 1 and %d", successes, conflicts, attempts-1)
	}
	assertSingleBooking(t, f.handler, slotID, created.ID)
}

func TestSeedResumesCalendarWithoutLosingBookings(t *testing.T) {
	f := newFixture(t)
	schedule := getSchedule(t, f.handler)
	if schedule.TimeZone != "Asia/Bishkek" || schedule.Today != "2026-11-16" || len(schedule.Slots) != 63 {
		t.Fatalf("initial schedule zone=%q date=%q slots=%d", schedule.TimeZone, schedule.Today, len(schedule.Slots))
	}

	first := schedule.Slots[0]
	if first.StartsAt.Format(time.RFC3339) != "2026-11-16T09:00:00+06:00" || first.Status != "past" {
		t.Fatalf("first slot = %+v, want 09:00 +06:00 and past", first)
	}
	assertStatus(t, postBooking(f.handler, first.ID), http.StatusConflict)

	future := schedule.Slots[1]
	response := postBooking(f.handler, future.ID)
	assertStatus(t, response, http.StatusCreated)
	var created struct{ ID int64 }
	decodeResponse(t, response, &created)

	for _, now := range []time.Time{f.now, f.now.AddDate(0, 0, 1)} {
		if err := f.store.SeedSlots(t.Context(), booking.WeekSlots(now, f.location)); err != nil {
			t.Fatal(err)
		}
	}
	schedule = getSchedule(t, f.handler)
	if len(schedule.Slots) != 72 {
		t.Fatalf("slots after reseed = %d, want 72", len(schedule.Slots))
	}
	last := schedule.Slots[71]
	if last.StartsAt.Format(time.RFC3339) != "2026-11-23T17:00:00+06:00" ||
		last.EndsAt.Format(time.RFC3339) != "2026-11-23T18:00:00+06:00" {
		t.Fatalf("last slot = %+v, want 17:00–18:00 on November 23", last)
	}
	assertSingleBooking(t, f.handler, future.ID, created.ID)
}

func TestInvalidBookingFields(t *testing.T) {
	f := newFixture(t)
	cases := []struct {
		body   string
		fields []string
	}{
		{
			body:   `{"slot_id":"invalid","customer_name":"  ","customer_email":"invalid"}`,
			fields: []string{"slot_id", "customer_name", "customer_email"},
		},
		{
			body:   `{"slot_id":1000,"customer_name":"Тест","customer_email":"test@example.com"}`,
			fields: []string{"slot_id"},
		},
		{
			body:   `{"slot_id":1000,"customer_name":" ","customer_email":"invalid"}`,
			fields: []string{"slot_id", "customer_name", "customer_email"},
		},
	}
	for _, test := range cases {
		response := serve(f.handler, http.MethodPost, "/api/bookings", []byte(test.body))
		assertStatus(t, response, http.StatusUnprocessableEntity)
		var result struct{ Errors map[string][]string }
		decodeResponse(t, response, &result)
		for _, field := range test.fields {
			if len(result.Errors[field]) == 0 {
				t.Fatalf("missing error for %s in %v", field, result.Errors)
			}
		}
	}
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	location, err := time.LoadLocation("Asia/Bishkek")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, time.November, 16, 3, 0, 0, 0, time.UTC)
	store, err := sqlite.Open(t.Context(), filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.Migrate(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := store.SeedSlots(t.Context(), booking.WeekSlots(now, location)); err != nil {
		t.Fatal(err)
	}
	service := booking.NewService(store, location, func() time.Time { return now })
	return fixture{store: store, handler: httpapi.NewHandler(service), location: location, now: now}
}

func serve(handler http.Handler, method, path string, body []byte) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, bytes.NewReader(body))
	if len(body) != 0 {
		request.Header.Set("Content-Type", "application/json")
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func postBooking(handler http.Handler, slotID int64) *httptest.ResponseRecorder {
	body := `{"slot_id":` + strconv.FormatInt(slotID, 10) + `,"customer_name":"Тест","customer_email":"test@example.com"}`
	return serve(handler, http.MethodPost, "/api/bookings", []byte(body))
}

func getSchedule(t *testing.T, handler http.Handler) listedSchedule {
	t.Helper()
	response := serve(handler, http.MethodGet, "/api/slots", nil)
	assertStatus(t, response, http.StatusOK)
	var schedule listedSchedule
	decodeResponse(t, response, &schedule)
	return schedule
}

func assertSingleBooking(t *testing.T, handler http.Handler, slotID, bookingID int64) {
	t.Helper()
	var count int
	for _, slot := range getSchedule(t, handler).Slots {
		if slot.ID == slotID && slot.BookingID != 0 {
			count++
			if slot.BookingID != bookingID {
				t.Fatalf("booking ID = %d, want %d", slot.BookingID, bookingID)
			}
		}
	}
	if count != 1 {
		t.Fatalf("active bookings = %d, want 1", count)
	}
}

func assertStatus(t *testing.T, response *httptest.ResponseRecorder, status int) {
	t.Helper()
	if response.Code != status {
		t.Fatalf("status = %d, want %d: %s", response.Code, status, response.Body)
	}
}

func decodeResponse(t *testing.T, response *httptest.ResponseRecorder, dst any) {
	t.Helper()
	if err := json.Unmarshal(response.Body.Bytes(), dst); err != nil {
		t.Fatalf("decode response %s: %v", response.Body, err)
	}
}
