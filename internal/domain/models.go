package domain

import "time"

// Role represents the authorization level of a user (SRP: only identity concern).
type Role string

const (
	RoleOrganizer Role = "ORGANIZER"
	RoleCustomer  Role = "CUSTOMER"
)

// User is the aggregate root for identity and authentication.
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Name         string    `json:"name"`
	Role         Role      `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
}

// Event is the aggregate root for event management.
type Event struct {
	ID            string    `json:"id"`
	OrganizerID   string    `json:"organizer_id"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	Venue         string    `json:"venue"`
	StartsAt      time.Time `json:"starts_at"`
	Capacity      int       `json:"capacity"`
	TicketsBooked int       `json:"tickets_booked"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (e Event) AvailableTickets() int {
	return e.Capacity - e.TicketsBooked
}

// Booking represents a customer's ticket reservation for an event.
type Booking struct {
	ID            string    `json:"id"`
	EventID       string    `json:"event_id"`
	CustomerID    string    `json:"customer_id"`
	CustomerEmail string    `json:"customer_email"`
	CustomerName  string    `json:"customer_name"`
	EventTitle    string    `json:"event_title"`
	Quantity      int       `json:"quantity"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}
