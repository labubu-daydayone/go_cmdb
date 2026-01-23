package models

import (
	"time"
)

// WSEvent represents a websocket event stored in the database
type WSEvent struct {
	ID        int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Topic     string    `gorm:"column:topic;type:varchar(64);not null;index:idx_topic_id" json:"topic"`
	EventType string    `gorm:"column:event_type;type:enum('add','update','delete');not null" json:"event_type"`
	Payload   string    `gorm:"column:payload;type:json;not null" json:"payload"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

// TableName specifies the table name for WSEvent
func (WSEvent) TableName() string {
	return "ws_events"
}
