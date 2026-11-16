package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"minibooking/backend/internal/booking"
)

type handler struct {
	service *booking.Service
}

type slotResponse struct {
	ID        int64     `json:"id"`
	StartsAt  time.Time `json:"starts_at"`
	EndsAt    time.Time `json:"ends_at"`
	Status    string    `json:"status"`
	BookingID int64     `json:"booking_id,omitempty"`
}

type scheduleResponse struct {
	TimeZone string         `json:"time_zone"`
	Today    string         `json:"today"`
	Slots    []slotResponse `json:"slots"`
}

type bookingRequest struct {
	SlotID json.RawMessage `json:"slot_id"`
	Name   json.RawMessage `json:"customer_name"`
	Email  json.RawMessage `json:"customer_email"`
}

type bookingResponse struct {
	ID int64 `json:"id"`
}

type errorResponse struct {
	Message string                  `json:"message"`
	Errors  booking.ValidationError `json:"errors,omitempty"`
}

func NewHandler(service *booking.Service) http.Handler {
	h := handler{service: service}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/slots", h.listSlots)
	mux.HandleFunc("POST /api/bookings", h.createBooking)
	mux.HandleFunc("DELETE /api/bookings/{id}", h.cancelBooking)
	return mux
}

func (h handler) listSlots(w http.ResponseWriter, r *http.Request) {
	schedule, err := h.service.ListSlots(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	response := scheduleResponse{
		TimeZone: schedule.TimeZone,
		Today:    schedule.Today,
		Slots:    make([]slotResponse, 0, len(schedule.Slots)),
	}
	for _, slot := range schedule.Slots {
		response.Slots = append(response.Slots, slotResponse{
			ID: slot.ID, StartsAt: slot.StartsAt, EndsAt: slot.EndsAt,
			Status: slot.Status, BookingID: slot.BookingID,
		})
	}
	writeJSON(w, http.StatusOK, response)
}

func (h handler) createBooking(w http.ResponseWriter, r *http.Request) {
	var request bookingRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10))
	if err := decoder.Decode(&request); err != nil || !errors.Is(decoder.Decode(new(json.RawMessage)), io.EOF) {
		writeJSON(w, http.StatusBadRequest, errorResponse{Message: "Отправьте один корректный JSON-объект"})
		return
	}

	input := booking.BookingInput{
		SlotID: decodeField[int64](request.SlotID),
		Name:   decodeField[string](request.Name),
		Email:  decodeField[string](request.Email),
	}
	id, err := h.service.CreateBooking(r.Context(), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, bookingResponse{ID: id})
}

func (h handler) cancelBooking(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, booking.ErrBookingNotFound)
		return
	}
	if err := h.service.CancelBooking(r.Context(), id); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func decodeField[T any](raw json.RawMessage) T {
	var value T
	if json.Unmarshal(raw, &value) != nil {
		var zero T
		return zero
	}
	return value
}

func writeError(w http.ResponseWriter, err error) {
	var validation booking.ValidationError
	switch {
	case errors.As(err, &validation):
		writeJSON(w, http.StatusUnprocessableEntity, errorResponse{
			Message: "Проверьте поля формы",
			Errors:  validation,
		})
	case errors.Is(err, booking.ErrSlotPast):
		writeJSON(w, http.StatusConflict, errorResponse{Message: "Этот слот уже начался. Выберите другое время"})
	case errors.Is(err, booking.ErrSlotBooked):
		writeJSON(w, http.StatusConflict, errorResponse{Message: "Этот слот уже забронирован. Выберите другое время"})
	case errors.Is(err, booking.ErrBookingNotFound):
		writeJSON(w, http.StatusNotFound, errorResponse{Message: "Бронирование не найдено"})
	default:
		slog.Error("booking API", "error", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Message: "Ошибка сервера. Попробуйте ещё раз"})
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
