package models

type ShortenedURL struct {
	Id        uint64 `gorm:"primaryKey autoIncrement"`
	CreatedBy uint64
	LongURL   string
	ShortCode string `gorm:"index"`
	CreatedAt int64
	UpdatedAt int64
}

type UrlInput struct {
	URL string `json:"url"`
}

type UrlDto struct {
	Id        string `json:"id"`
	URL       string `json:"url"`
	ShortUrl  string `json:"short_url"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}
