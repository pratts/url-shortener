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
	Id string `json:"id"`
}
