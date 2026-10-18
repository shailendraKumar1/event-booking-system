package service

import (
	"strings"
	"time"

	"event-booking-system/internal/domain"
	"event-booking-system/internal/repository"
	"event-booking-system/internal/worker"
)

// BookingService handles ticket reservation and triggers async confirmation.
// SRP: booking logic only; email dispatch delegated to worker pool.
type BookingService struct {
	eventRepo   repository.EventRepository
	bookingRepo repository.BookingRepository
	workerPool  *worker.Pool
}

func NewBookingService(
	eventRepo repository.EventRepository,
	bookingRepo repository.BookingRepository,
	pool *worker.Pool,
) *BookingService {
	return &BookingService{
		eventRepo:   eventRepo,
		bookingRepo: bookingRepo,
		workerPool:  pool,
	}
}

type BookTicketInput struct {
	Quantity int `json:"quantity"`
}

func (s *BookingService) Book(eventID, customerID, customerEmail, customerName string, input BookTicketInput) (domain.Booking, error) {
	if input.Quantity < 1 {
		return domain.Booking{}, domain.ErrInvalidInput
	}
	if strings.TrimSpace(customerEmail) == "" {
		return domain.Booking{}, domain.ErrInvalidInput
	}

	event, err := s.eventRepo.Reserve(eventID, input.Quantity)
	if err != nil {
		return domain.Booking{}, err
	}

	booking := domain.Booking{
		EventID:       eventID,
		CustomerID:    customerID,
		CustomerEmail: customerEmail,
		CustomerName:  customerName,
		EventTitle:    event.Title,
		Quantity:      input.Quantity,
	}
	booking, err = s.bookingRepo.Create(booking)
	if err != nil {
		return domain.Booking{}, err
	}

	// Background Task 1: send booking confirmation email asynchronously
	s.workerPool.Submit(worker.Job{
		Type: worker.JobBookingConfirmation,
		Payload: worker.BookingPayload{
			CustomerEmail: customerEmail,
			CustomerName:  customerName,
			EventTitle:    event.Title,
			EventVenue:    event.Venue,
			EventDate:     event.StartsAt.Format(time.RFC3339),
			Quantity:      input.Quantity,
			BookingID:     booking.ID,
		},
	})

	return booking, nil
}

func (s *BookingService) MyBookings(customerID string) ([]domain.Booking, error) {
	return s.bookingRepo.FindByCustomerID(customerID)
}

func (s *BookingService) GetBooking(id string) (domain.Booking, error) {
	return s.bookingRepo.FindByID(id)
}
