package notification

import (
	"encoding/json"
	"fmt"
	"time"

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
func (ns *NotificationService) SendNotification(channel, event string, message map[string]interface{}) error {
	messageJSON, err := json.Marshal(message)
	if err != nil {
		return err
	}

	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		err = ns.pusherClient.Trigger(channel, event, string(messageJSON))
		if err == nil {
			return nil
		}
		time.Sleep(time.Second * time.Duration(i+1))
	}

	return fmt.Errorf("failed to send notification after %d retries: %v", maxRetries, err)
}
