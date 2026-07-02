// Package event defines event types and the event bus for real-time notifications (Phase 6).
package event

import "encoding/json"

// EventType represents a type of notification event.
type EventType string

const (
	TaskAssigned        EventType = "task_assigned"
	WorkSubmitted       EventType = "work_submitted"
	TaskApproved        EventType = "task_approved"
	TaskRejected        EventType = "task_rejected"
	TaskPublished       EventType = "task_published"
	PaymentSent         EventType = "payment_sent"
	RatingCreated       EventType = "rating_created"
	FreelancerApproved  EventType = "freelancer_approved"
	FreelancerRejected  EventType = "freelancer_rejected"
)

// NotificationPayload represents the data sent via WebSocket.
type NotificationPayload struct {
	Type      EventType       `json:"type"`
	Message   string          `json:"message"`
	Data      json.RawMessage `json:"data,omitempty"`
	Timestamp string          `json:"timestamp"`
}
