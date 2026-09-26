// Package analytics records clicks on short links.
package analytics

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// Click is one visit to a short link.
type Click struct {
	ID         uint64 `gorm:"column:id;primaryKey"`
	ShortURLID uint64 `gorm:"column:short_url_id"`
	ShortCode  string
	CreatedBy  uint64
	Agent      string
	IPAddress  string
	Metadata   map[string]interface{} `gorm:"serializer:json"`
	CreatedAt  time.Time
}

func (Click) TableName() string { return "url_redirects" }

// Recorder stores clicks.
type Recorder struct {
	db *gorm.DB
}

func NewRecorder(db *gorm.DB) *Recorder {
	return &Recorder{db: db}
}

func (r *Recorder) Record(ctx context.Context, c Click) error {
	if c.Metadata == nil {
		c.Metadata = map[string]interface{}{}
	}
	return r.db.WithContext(ctx).Create(&c).Error
}
