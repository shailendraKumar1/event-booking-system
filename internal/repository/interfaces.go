package repository

import "event-booking-system/internal/domain"

// UserRepository defines the persistence contract for user aggregates.
// ISP: segregated from event/booking concerns.
type UserRepository interface {
	Create(user domain.User) (domain.User, error)
	FindByEmail(email string) (domain.User, error)
	FindByID(id string) (domain.User, error)
}

// EventRepository defines the persistence contract for event aggregates.
type EventRepository interface {
	Create(event domain.Event) (domain.Event, error)
	Update(id, organizerID string, event domain.Event) (domain.Event, error)
	FindByID(id string) (domain.Event, error)
	FindAll(offset, limit int) ([]domain.Event, int, error)
	Reserve(eventID string, quantity int) (domain.Event, error)
}

// BookingRepository defines the persistence contract for booking aggregates.
type BookingRepository interface {
	Create(booking domain.Booking) (domain.Booking, error)
	FindByEventID(eventID string) ([]domain.Booking, error)
	FindByCustomerID(customerID string) ([]domain.Booking, error)
	FindByID(id string) (domain.Booking, error)
}
