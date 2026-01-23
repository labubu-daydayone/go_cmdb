package ws

import (
	"encoding/json"
	"fmt"
	"log"

	"go_cmdb/internal/db"
	"go_cmdb/models"
)

// PublishWebsiteEvent publishes a website event to the database and broadcasts it
// eventType: "add", "update", "delete"
// payload: the website data to be sent to clients
func PublishWebsiteEvent(eventType string, payload interface{}) error {
	// 1. Serialize payload to JSON
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[WebSocket] Failed to marshal payload: %v", err)
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// 2. Write event to database
	event := models.WSEvent{
		Topic:     "websites",
		EventType: eventType,
		Payload:   string(payloadJSON),
	}

	if err := db.GetDB().Create(&event).Error; err != nil {
		log.Printf("[WebSocket] Failed to write event to database: %v", err)
		return fmt.Errorf("failed to write event to database: %w", err)
	}

	log.Printf("[WebSocket] Event written to database: id=%d, type=%s, topic=%s", event.ID, eventType, event.Topic)

	// 3. Broadcast event to all connected clients
	// Note: Broadcast failure should not affect the main flow
	broadcastData := map[string]interface{}{
		"eventId": event.ID,
		"type":    eventType,
		"data":    payload,
	}

	// Broadcast to all clients (no room filtering for now)
	BroadcastToAll("websites:update", broadcastData)

	log.Printf("[WebSocket] Event broadcasted: eventId=%d, type=%s", event.ID, eventType)

	return nil
}

// GetIncrementalEvents retrieves incremental events from the database
// Returns events with id > lastEventId, limited to maxCount
func GetIncrementalEvents(lastEventId int64, maxCount int) ([]models.WSEvent, error) {
	var events []models.WSEvent

	err := db.GetDB().
		Where("topic = ? AND id > ?", "websites", lastEventId).
		Order("id ASC").
		Limit(maxCount).
		Find(&events).Error

	if err != nil {
		return nil, fmt.Errorf("failed to query incremental events: %w", err)
	}

	return events, nil
}

// GetLatestEventId retrieves the latest event ID from the database
func GetLatestEventId() (int64, error) {
	var event models.WSEvent

	err := db.GetDB().
		Where("topic = ?", "websites").
		Order("id DESC").
		Limit(1).
		First(&event).Error

	if err != nil {
		// If no events found, return 0
		if err.Error() == "record not found" {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to query latest event: %w", err)
	}

	return event.ID, nil
}
