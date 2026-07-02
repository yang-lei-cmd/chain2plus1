package event

import (
	"encoding/json"
	"time"
)

// SendNotification is a convenience function to emit a notification via the hub.
// This is called from handlers after state-changing operations.
func (h *Hub) SendNotification(targetUserID uint, eventType EventType, message string, data interface{}) {
	var jsonData []byte
	if data != nil {
		jsonData, _ = json.Marshal(data)
	}

	payload := NotificationPayload{
		Type:      eventType,
		Message:   message,
		Data:      json.RawMessage(jsonData),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	if targetUserID == 0 {
		h.BroadcastToAll(payload)
	} else {
		h.BroadcastToUser(targetUserID, payload)
	}
}
