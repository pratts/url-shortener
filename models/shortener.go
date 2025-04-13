package models

type ShortenedURL struct {
	Id        string `json:"id"`
	URL       string `json:"url"`
	Code      string `json:"code"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

type UrlInput struct {
	URL string `json:"url"`
}
