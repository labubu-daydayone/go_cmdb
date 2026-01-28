package ws

import (
	"encoding/json"
	"log"

	socketio "github.com/googollee/go-socket.io"
	"go_cmdb/internal/db"
	"go_cmdb/internal/model"
)

// RequestWebsitesData represents the data sent by client in request:websites event
type RequestWebsitesData struct {
	LastEventId int64 `json:"lastEventId"`
}

// WebsiteListItem represents a website item in the list
type WebsiteListItem struct {
	ID                 int      `json:"id"`
	LineGroupID        int      `json:"line_group_id"`
	LineGroupName      string   `json:"line_group_name"`
	CacheRuleID        int      `json:"cache_rule_id"`
	OriginMode         string   `json:"origin_mode"`
	OriginGroupID      int      `json:"origin_group_id"`
	OriginGroupName    string   `json:"origin_group_name"`
	OriginSetID        int      `json:"origin_set_id"`
	RedirectURL        string   `json:"redirect_url"`
	RedirectStatusCode int      `json:"redirect_status_code"`
	Status             string   `json:"status"`
	Domains            []string `json:"domains"`
	PrimaryDomain      string   `json:"primary_domain"`
	HTTPSEnabled       bool     `json:"https_enabled"`
	CreatedAt          string   `json:"created_at"`
	UpdatedAt          string   `json:"updated_at"`
}

// handleRequestWebsites handles the request:websites event
func handleRequestWebsites(s socketio.Conn, data interface{}) {
	log.Printf("[WebSocket] request:websites from client %s, data: %v", s.ID(), data)

	// Parse lastEventId from data
	var lastEventId int64 = 0
	if dataMap, ok := data.(map[string]interface{}); ok {
		if lastEventIdFloat, ok := dataMap["lastEventId"].(float64); ok {
			lastEventId = int64(lastEventIdFloat)
		}
	}

	log.Printf("[WebSocket] Parsed lastEventId: %d", lastEventId)

	// If lastEventId is provided, try to send incremental updates
	if lastEventId > 0 {
		if sendIncrementalUpdates(s, lastEventId) {
			// Incremental updates sent successfully
			return
		}
		// If incremental updates failed, fall through to send full list
		log.Printf("[WebSocket] Incremental updates failed, falling back to full list")
	}

	// Send full list
	sendFullWebsitesList(s)
}

// sendIncrementalUpdates sends incremental updates to the client
// Returns true if successful, false if should fall back to full list
func sendIncrementalUpdates(s socketio.Conn, lastEventId int64) bool {
	// Query incremental events (limit to 500)
	maxCount := 500
	events, err := GetIncrementalEvents(lastEventId, maxCount)
	if err != nil {
		log.Printf("[WebSocket] Failed to query incremental events: %v", err)
		return false
	}

	// If too many events (>= maxCount), fall back to full list
	if len(events) >= maxCount {
		log.Printf("[WebSocket] Too many incremental events (%d), falling back to full list", len(events))
		return false
	}

	// If no events, send empty response
	if len(events) == 0 {
		log.Printf("[WebSocket] No incremental events found")
		// Get latest event ID
		latestEventId, _ := GetLatestEventId()
		s.Emit("websites:initial", map[string]interface{}{
			"items":       []interface{}{},
			"total":       0,
			"version":     0,
			"lastEventId": latestEventId,
		})
		return true
	}

	// Send incremental updates
	log.Printf("[WebSocket] Sending %d incremental events", len(events))
	for _, event := range events {
		// Parse payload
		var payload interface{}
		if err := json.Unmarshal([]byte(event.Payload), &payload); err != nil {
			log.Printf("[WebSocket] Failed to unmarshal event payload: %v", err)
			continue
		}

		// Emit websites:update event
		s.Emit("websites:update", map[string]interface{}{
			"eventId": event.ID,
			"type":    event.EventType,
			"data":    payload,
		})
	}

	return true
}

// sendFullWebsitesList sends the full websites list to the client
func sendFullWebsitesList(s socketio.Conn) {
	// Query full websites list
	var websitesList []WebsiteListItem
	var total int64

	// Use the same logic as List handler
	query := db.GetDB().Model(&model.Website{})

	// Count total
	if err := query.Count(&total).Error; err != nil {
		log.Printf("[WebSocket] Failed to count websites: %v", err)
		s.Emit("error", map[string]interface{}{
			"message": "Failed to query websites",
		})
		return
	}

	// Query all websites (limit to 10000 for safety)
	var websitesModels []model.Website
	if err := query.Limit(10000).Find(&websitesModels).Error; err != nil {
		log.Printf("[WebSocket] Failed to query websites: %v", err)
		s.Emit("error", map[string]interface{}{
			"message": "Failed to query websites",
		})
		return
	}

	// Convert to WebsiteListItem (simplified version)
	// Note: In production, you should use the same conversion logic as the List handler
	for _, website := range websitesModels {
		item := WebsiteListItem{
			ID:                 website.ID,
			LineGroupID:        website.LineGroupID,
			CacheRuleID:        website.CacheRuleID,
			OriginMode:         website.OriginMode,
			OriginGroupID:      website.OriginGroupID,
			OriginSetID:        website.OriginSetID,
			RedirectURL:        website.RedirectURL,
			RedirectStatusCode: website.RedirectStatusCode,
			Status:             website.Status,
			HTTPSEnabled:       false, // TODO: Query from website_https table
			CreatedAt:          website.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:          website.UpdatedAt.Format("2006-01-02 15:04:05"),
		}

		// TODO: Query domains, line group name, origin group name, etc.
		// For now, just add the basic fields

		websitesList = append(websitesList, item)
	}

	// Get latest event ID
	latestEventId, _ := GetLatestEventId()

	// Send websites:initial event
	s.Emit("websites:initial", map[string]interface{}{
		"items":       websitesList,
		"total":       total,
		"version":     0, // TODO: Add version tracking
		"lastEventId": latestEventId,
	})

	log.Printf("[WebSocket] Sent full websites list: total=%d, lastEventId=%d", total, latestEventId)
}
