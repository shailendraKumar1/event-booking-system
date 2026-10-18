package handler

import (
	"errors"
	"net/http"
	"strconv"

	"event-booking-system/internal/domain"
	"event-booking-system/internal/middleware"
	"event-booking-system/internal/service"
)

// EventHandler handles event-related HTTP endpoints.
type EventHandler struct {
	eventService *service.EventService
}

func NewEventHandler(eventService *service.EventService) *EventHandler {
	return &EventHandler{eventService: eventService}
}

func (h *EventHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input service.CreateEventInput
	if !decode(w, r, &input) {
		return
	}
	organizerID := middleware.GetUserID(r)
	event, err := h.eventService.Create(organizerID, input)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			writeError(w, http.StatusBadRequest, "title and capacity (>0) are required")
		} else {
			writeError(w, http.StatusInternalServerError, "failed to create event")
		}
		return
	}
	writeJSON(w, http.StatusCreated, event)
}

func (h *EventHandler) Update(w http.ResponseWriter, r *http.Request) {
	eventID := r.PathValue("id")
	var input service.UpdateEventInput
	if !decode(w, r, &input) {
		return
	}
	organizerID := middleware.GetUserID(r)
	event, err := h.eventService.Update(eventID, organizerID, input)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotFound):
			writeError(w, http.StatusNotFound, "event not found")
		case errors.Is(err, domain.ErrForbidden):
			writeError(w, http.StatusForbidden, "you can only update your own events")
		default:
			writeError(w, http.StatusBadRequest, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, event)
}

func (h *EventHandler) Get(w http.ResponseWriter, r *http.Request) {
	eventID := r.PathValue("id")
	event, err := h.eventService.Get(eventID)
	if err != nil {
		writeError(w, http.StatusNotFound, "event not found")
		return
	}
	writeJSON(w, http.StatusOK, event)
}

func (h *EventHandler) List(w http.ResponseWriter, r *http.Request) {
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 20
	}
	events, total, err := h.eventService.List(offset, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list events")
		return
	}
	writeJSONWithMeta(w, http.StatusOK, events, &Meta{Total: total, Offset: offset, Limit: limit})
}
