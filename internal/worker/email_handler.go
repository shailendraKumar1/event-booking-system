package worker

import (
	"fmt"

	"event-booking-system/internal/email"
)

// EmailJobHandler processes email-related background jobs.
// Implements the Handler interface (LSP: substitutable with any Handler).
type EmailJobHandler struct {
	mailer email.Mailer
}

func NewEmailJobHandler(mailer email.Mailer) *EmailJobHandler {
	return &EmailJobHandler{mailer: mailer}
}

func (h *EmailJobHandler) Handle(job Job) error {
	switch job.Type {
	case JobBookingConfirmation:
		return h.handleBookingConfirmation(job)
	case JobEventNotification:
		return h.handleEventNotification(job)
	default:
		return fmt.Errorf("unknown job type: %s", job.Type)
	}
}

func (h *EmailJobHandler) handleBookingConfirmation(job Job) error {
	p, ok := job.Payload.(BookingPayload)
	if !ok {
		return fmt.Errorf("invalid payload for booking confirmation")
	}
	subject := "Booking Confirmed - " + p.EventTitle
	body := fmt.Sprintf(`<html><body>
<h2>Booking Confirmation</h2>
<p>Dear %s,</p>
<p>Your booking has been confirmed!</p>
<table style="border-collapse:collapse;margin:16px 0">
<tr><td style="padding:4px 12px;font-weight:bold">Booking ID</td><td style="padding:4px 12px">%s</td></tr>
<tr><td style="padding:4px 12px;font-weight:bold">Event</td><td style="padding:4px 12px">%s</td></tr>
<tr><td style="padding:4px 12px;font-weight:bold">Venue</td><td style="padding:4px 12px">%s</td></tr>
<tr><td style="padding:4px 12px;font-weight:bold">Date</td><td style="padding:4px 12px">%s</td></tr>
<tr><td style="padding:4px 12px;font-weight:bold">Tickets</td><td style="padding:4px 12px">%d</td></tr>
</table>
<p>Thank you for your booking!</p>
</body></html>`, p.CustomerName, p.BookingID, p.EventTitle, p.EventVenue, p.EventDate, p.Quantity)
	return h.mailer.Send(p.CustomerEmail, subject, body)
}

func (h *EmailJobHandler) handleEventNotification(job Job) error {
	p, ok := job.Payload.(EventUpdatePayload)
	if !ok {
		return fmt.Errorf("invalid payload for event notification")
	}
	subject := "Event Updated - " + p.EventTitle
	body := fmt.Sprintf(`<html><body>
<h2>Event Update Notification</h2>
<p>Dear %s,</p>
<p>An event you have booked tickets for has been updated:</p>
<p><strong>%s</strong></p>
<p>Changes: %s</p>
<p>Please check the event details for the latest information.</p>
</body></html>`, p.CustomerName, p.EventTitle, p.Changes)
	return h.mailer.Send(p.CustomerEmail, subject, body)
}
