package models

import "time"

// UrlRedirect is one click on a short URL.
type UrlRedirect struct {
	ID        uint64 `gorm:"primaryKey;autoIncrement"`
	ShortCode string `gorm:"index"`
	// (short_url_id, created_at) serves per-link, time-ranged click queries.
	ShortUrlID uint64 `gorm:"not null;index:idx_url_redirects_url_time,priority:1"`
	Agent      string
	IPAddress  string
	CreatedBy  uint64                 `gorm:"not null;index"`
	Metadata   map[string]interface{} `gorm:"type:jsonb;serializer:json"`
	CreatedAt  time.Time              `gorm:"index:idx_url_redirects_url_time,priority:2"`
}
