package notification

import (
	"encoding/json"

	"github.com/pusher/pusher-http-go/v5"
)

// notificationService handles Pusher notifications.
type NotificationService struct {
    pusherClient *pusher.Client
}

// New creates a new notificationService instance.
func New(pusherClient *pusher.Client) *NotificationService {
    return &NotificationService{
        pusherClient: pusherClient,
    }
}

// SendNotification sends a notification with the specified channel, event, and message.
func (ns *NotificationService) SendNotification(channel, event string, message map[string]interface{} ) error {
    // Serialize the message to JSON
	messageJSON, err := json.Marshal(message)
	if err != nil {
		return err
	}

	// Trigger the notification with the serialized JSON message
	return ns.pusherClient.Trigger(channel, event, string(messageJSON))
}
