package handler

import (
	"errors"
	"net/http"

	"event-booking-system/internal/domain"
	"event-booking-system/internal/middleware"
	"event-booking-system/internal/service"
)

// BookingHandler handles booking-related HTTP endpoints.
type BookingHandler struct {
	bookingService *service.BookingService
}

func NewBookingHandler(bookingService *service.BookingService) *BookingHandler {
	return &BookingHandler{bookingService: bookingService}
}

func (h *BookingHandler) Book(w http.ResponseWriter, r *http.Request) {
	eventID := r.PathValue("id")
	var input service.BookTicketInput
	if !decode(w, r, &input) {
		return
	}
	customerID := middleware.GetUserID(r)
	customerEmail := middleware.GetUserEmail(r)
	customerName := middleware.GetUserName(r)

	booking, err := h.bookingService.Book(eventID, customerID, customerEmail, customerName, input)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrCapacity):
			writeError(w, http.StatusConflict, "not enough tickets available")
		case errors.Is(err, domain.ErrNotFound):
			writeError(w, http.StatusNotFound, "event not found")
		case errors.Is(err, domain.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, "quantity must be at least 1")
		default:
			writeError(w, http.StatusInternalServerError, "booking failed")
		}
		return
	}
	writeJSON(w, http.StatusCreated, booking)
}

func (h *BookingHandler) MyBookings(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetUserID(r)
	bookings, err := h.bookingService.MyBookings(customerID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list bookings")
		return
	}
	writeJSON(w, http.StatusOK, bookings)
}
