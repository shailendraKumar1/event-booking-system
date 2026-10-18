package memory

import (
	"fmt"
	"sync"
	"time"

	"event-booking-system/internal/domain"
)

// UserRepo is a thread-safe in-memory implementation of repository.UserRepository.
type UserRepo struct {
	mu    sync.RWMutex
	users map[string]domain.User
	byEmail map[string]string
	seq   int
}

func NewUserRepo() *UserRepo {
	return &UserRepo{
		users:   make(map[string]domain.User),
		byEmail: make(map[string]string),
	}
}

func (r *UserRepo) Create(u domain.User) (domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.byEmail[u.Email]; exists {
		return domain.User{}, domain.ErrDuplicateEmail
	}
	r.seq++
	u.ID = fmt.Sprintf("user_%d", r.seq)
	u.CreatedAt = time.Now().UTC()
	r.users[u.ID] = u
	r.byEmail[u.Email] = u.ID
	return u, nil
}

func (r *UserRepo) FindByEmail(email string) (domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	id, ok := r.byEmail[email]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}
	return r.users[id], nil
}

func (r *UserRepo) FindByID(id string) (domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	u, ok := r.users[id]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}
	return u, nil
}

// EventRepo is a thread-safe in-memory implementation of repository.EventRepository.
type EventRepo struct {
	mu     sync.RWMutex
	events map[string]domain.Event
	order  []string
	seq    int
}

func NewEventRepo() *EventRepo {
	return &EventRepo{
		events: make(map[string]domain.Event),
		order:  make([]string, 0),
	}
}

func (r *EventRepo) Create(e domain.Event) (domain.Event, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	e.ID = fmt.Sprintf("evt_%d", r.seq)
	now := time.Now().UTC()
	e.CreatedAt = now
	e.UpdatedAt = now
	r.events[e.ID] = e
	r.order = append(r.order, e.ID)
	return e, nil
}

func (r *EventRepo) Update(id, organizerID string, change domain.Event) (domain.Event, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	e, ok := r.events[id]
	if !ok {
		return domain.Event{}, domain.ErrNotFound
	}
	if e.OrganizerID != organizerID {
		return domain.Event{}, domain.ErrForbidden
	}
	if change.Title != "" {
		e.Title = change.Title
	}
	if change.Description != "" {
		e.Description = change.Description
	}
	if change.Venue != "" {
		e.Venue = change.Venue
	}
	if !change.StartsAt.IsZero() {
		e.StartsAt = change.StartsAt
	}
	if change.Capacity > 0 && change.Capacity >= e.TicketsBooked {
		e.Capacity = change.Capacity
	}
	e.UpdatedAt = time.Now().UTC()
	r.events[id] = e
	return e, nil
}

func (r *EventRepo) FindByID(id string) (domain.Event, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	e, ok := r.events[id]
	if !ok {
		return domain.Event{}, domain.ErrNotFound
	}
	return e, nil
}

func (r *EventRepo) FindAll(offset, limit int) ([]domain.Event, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	total := len(r.order)
	if offset >= total {
		return []domain.Event{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	out := make([]domain.Event, 0, end-offset)
	for i := offset; i < end; i++ {
		out = append(out, r.events[r.order[i]])
	}
	return out, total, nil
}

// Reserve atomically checks capacity and decrements available tickets.
func (r *EventRepo) Reserve(eventID string, quantity int) (domain.Event, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	e, ok := r.events[eventID]
	if !ok {
		return domain.Event{}, domain.ErrNotFound
	}
	if e.TicketsBooked+quantity > e.Capacity {
		return domain.Event{}, domain.ErrCapacity
	}
	e.TicketsBooked += quantity
	r.events[eventID] = e
	return e, nil
}

// BookingRepo is a thread-safe in-memory implementation of repository.BookingRepository.
type BookingRepo struct {
	mu       sync.RWMutex
	bookings map[string]domain.Booking
	byEvent  map[string][]string
	byCust   map[string][]string
	seq      int
}

func NewBookingRepo() *BookingRepo {
	return &BookingRepo{
		bookings: make(map[string]domain.Booking),
		byEvent:  make(map[string][]string),
		byCust:   make(map[string][]string),
	}
}

func (r *BookingRepo) Create(b domain.Booking) (domain.Booking, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	b.ID = fmt.Sprintf("bkg_%d", r.seq)
	b.CreatedAt = time.Now().UTC()
	b.Status = "CONFIRMED"
	r.bookings[b.ID] = b
	r.byEvent[b.EventID] = append(r.byEvent[b.EventID], b.ID)
	r.byCust[b.CustomerID] = append(r.byCust[b.CustomerID], b.ID)
	return b, nil
}

func (r *BookingRepo) FindByEventID(eventID string) ([]domain.Booking, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := r.byEvent[eventID]
	out := make([]domain.Booking, 0, len(ids))
	for _, id := range ids {
		out = append(out, r.bookings[id])
	}
	return out, nil
}

func (r *BookingRepo) FindByCustomerID(customerID string) ([]domain.Booking, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := r.byCust[customerID]
	out := make([]domain.Booking, 0, len(ids))
	for _, id := range ids {
		out = append(out, r.bookings[id])
	}
	return out, nil
}

func (r *BookingRepo) FindByID(id string) (domain.Booking, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	b, ok := r.bookings[id]
	if !ok {
		return domain.Booking{}, domain.ErrNotFound
	}
	return b, nil
}
