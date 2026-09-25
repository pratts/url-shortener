package models

import "time"

type ShortenedURL struct {
	Id uint64 `gorm:"primaryKey;autoIncrement;index:idx_shortened_urls_owner_id,priority:2"`
	// (created_by, id) serves the per-user listing ordered by id.
	CreatedBy uint64 `gorm:"not null;index:idx_shortened_urls_owner_id,priority:1"`
	LongURL   string `gorm:"not null"`
	ShortCode string `gorm:"not null;uniqueIndex:uidx_shortened_urls_short_code"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type UrlInput struct {
	URL string `json:"url"`
}

type UrlDto struct {
	Id        uint64 `json:"id"`
	URL       string `json:"url"`
	ShortCode string `json:"short_code"`
	ShortUrl  string `json:"short_url"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	CreatedBy uint64 `json:"created_by"`
}
