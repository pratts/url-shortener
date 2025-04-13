package models

type ShortenedURL struct {
	Id        string `json:"id"`
	CreatedBy string `json:"created_by"`
	LongURL   string `json:"url"`
	ShortCode string `json:"code"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

type UrlInput struct {
	URL string `json:"url"`
}
