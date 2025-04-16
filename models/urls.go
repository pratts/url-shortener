package models

import "time"

type ShortenedURL struct {
	Id        uint64 `gorm:"primaryKey autoIncrement"`
	CreatedBy uint64 `gorm:"index"`
	LongURL   string
	ShortCode string `gorm:"index"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type UrlInput struct {
	URL string `json:"url"`
}

type UrlDto struct {
	Id        uint64 `json:"id"`
	URL       string `json:"url"`
	ShortUrl  string `json:"short_url"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}
