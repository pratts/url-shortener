package services

import (
	"fmt"
	"math/rand"
	"shortener/models"
	"sync"
	"time"
)

var urlMap map[string]models.ShortenedURL
var mu sync.Mutex

func InitMap() {
	urlMap = make(map[string]models.ShortenedURL)
}

func generateCode() string {
	const base62 = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	const length = 6
	maxAttempts := 1000
	for {
		if maxAttempts <= 0 {
			return ""
		}
		maxAttempts--
		code := ""
		for i := 0; i < length; i++ {
			code += string(base62[rand.Intn(len(base62))])
		}
		_, exists := urlMap[code]
		if !exists {
			return code
		}
	}
}

func Shorten(url string) models.ShortenedURL {
	code := generateCode()
	if code == "" {
		panic("Failed to generate a unique URL code")
	}
	shortenedUrlDetails := models.ShortenedURL{
		Id:        code,
		ShortCode: code,
		LongURL:   url,
		CreatedAt: time.Now().UnixMilli(),
		UpdatedAt: time.Now().UnixMilli(),
	}

	mu.Lock()
	defer mu.Unlock()
	urlMap[code] = shortenedUrlDetails
	return shortenedUrlDetails
}

func Expand(shortened string) (string, error) {
	if details, exists := urlMap[shortened]; exists {
		return details.LongURL, nil
	}
	return "", fmt.Errorf("URL not found")
}
