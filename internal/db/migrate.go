package db

import (
	"fmt"
	"log"

	"go_cmdb/internal/model"

	"gorm.io/gorm"
)

// Migrate runs database migrations for all models
func Migrate(db *gorm.DB) error {
	log.Println("Starting database migration...")

	// List of all models to migrate
	models := []interface{}{
		&model.User{},
		&model.APIKey{},
		&model.Domain{},
		&model.DomainDNSProvider{},
		&model.DomainDNSRecord{},
		&model.Node{},
		&model.NodeSubIP{},
		&model.NodeGroup{},
		&model.NodeGroupSubIP{},
		&model.LineGroup{},
	}

	// Run AutoMigrate for all models
	if err := db.AutoMigrate(models...); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	log.Printf("✓ Database migration completed successfully (%d tables)", len(models))
	return nil
}
