package service

import (
	"strings"

	"event-booking-system/internal/cache"
	"event-booking-system/internal/domain"
	"event-booking-system/internal/repository"
	"event-booking-system/internal/worker"
)

// EventService handles event business logic.
// SRP: only event-related operations; notifications delegated to worker pool.
type EventService struct {
	eventRepo   repository.EventRepository
	bookingRepo repository.BookingRepository
	cache       *cache.EventCache
	workerPool  *worker.Pool
}

func NewEventService(
	eventRepo repository.EventRepository,
	bookingRepo repository.BookingRepository,
	cache *cache.EventCache,
	pool *worker.Pool,
) *EventService {
	return &EventService{
		eventRepo:   eventRepo,
		bookingRepo: bookingRepo,
		cache:       cache,
		workerPool:  pool,
	}
}

type CreateEventInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Venue       string `json:"venue"`
	StartsAt    string `json:"starts_at"`
	Capacity    int    `json:"capacity"`
}

type UpdateEventInput struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Venue       string `json:"venue,omitempty"`
	StartsAt    string `json:"starts_at,omitempty"`
	Capacity    int    `json:"capacity,omitempty"`
}

func (s *EventService) Create(organizerID string, input CreateEventInput) (domain.Event, error) {
	if strings.TrimSpace(input.Title) == "" || input.Capacity < 1 {
		return domain.Event{}, domain.ErrInvalidInput
	}
	startsAt, err := parseTime(input.StartsAt)
	if err != nil {
		return domain.Event{}, domain.ErrInvalidInput
	}
	event := domain.Event{
		OrganizerID: organizerID,
		Title:       strings.TrimSpace(input.Title),
		Description: strings.TrimSpace(input.Description),
		Venue:       strings.TrimSpace(input.Venue),
		StartsAt:    startsAt,
		Capacity:    input.Capacity,
	}
	return s.eventRepo.Create(event)
}

func (s *EventService) Update(eventID, organizerID string, input UpdateEventInput) (domain.Event, error) {
	change := domain.Event{
		Title:       strings.TrimSpace(input.Title),
		Description: strings.TrimSpace(input.Description),
		Venue:       strings.TrimSpace(input.Venue),
		Capacity:    input.Capacity,
	}
	if input.StartsAt != "" {
		t, err := parseTime(input.StartsAt)
		if err != nil {
			return domain.Event{}, domain.ErrInvalidInput
		}
		change.StartsAt = t
	}
	updated, err := s.eventRepo.Update(eventID, organizerID, change)
	if err != nil {
		return domain.Event{}, err
	}
	s.cache.Invalidate(eventID)

	// Background Task 2: notify all booked customers about the update
	go s.notifyBookedCustomers(updated, input)
	return updated, nil
}

func (s *EventService) notifyBookedCustomers(event domain.Event, input UpdateEventInput) {
	bookings, err := s.bookingRepo.FindByEventID(event.ID)
	if err != nil || len(bookings) == 0 {
		return
	}
	changes := buildChangesSummary(input)
	for _, b := range bookings {
		s.workerPool.Submit(worker.Job{
			Type: worker.JobEventNotification,
			Payload: worker.EventUpdatePayload{
				CustomerEmail: b.CustomerEmail,
				CustomerName:  b.CustomerName,
				EventTitle:    event.Title,
				Changes:       changes,
			},
		})
	}
}

func buildChangesSummary(input UpdateEventInput) string {
	var parts []string
	if input.Title != "" {
		parts = append(parts, "title updated")
	}
	if input.Description != "" {
		parts = append(parts, "description updated")
	}
	if input.Venue != "" {
		parts = append(parts, "venue changed to "+input.Venue)
	}
	if input.StartsAt != "" {
		parts = append(parts, "date changed to "+input.StartsAt)
	}
	if input.Capacity > 0 {
		parts = append(parts, "capacity updated")
	}
	if len(parts) == 0 {
		return "event details updated"
	}
	return strings.Join(parts, "; ")
}

func (s *EventService) Get(id string) (domain.Event, error) {
	if e, ok := s.cache.Get(id); ok {
		return e, nil
	}
	e, err := s.eventRepo.FindByID(id)
	if err != nil {
		return domain.Event{}, err
	}
	s.cache.Put(e)
	return e, nil
}

func (s *EventService) List(offset, limit int) ([]domain.Event, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return s.eventRepo.FindAll(offset, limit)
}
