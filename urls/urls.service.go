package urls

import (
	"fmt"
	"math/rand"
	"shortener/models"
	"time"
)

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
		return code
		// models.DBObj.
		// _, exists := urlMap[code]
		// if !exists {
		// 	return code
		// }
	}
}

func Shorten(url string) models.ShortenedURL {
	code := generateCode()
	if code == "" {
		panic("Failed to generate a unique URL code")
	}
	shortenedUrlDetails := models.ShortenedURL{
		ShortCode: code,
		LongURL:   url,
		CreatedAt: time.Now().UnixMilli(),
		UpdatedAt: time.Now().UnixMilli(),
	}

	db := models.DBObj.Create(&shortenedUrlDetails)
	if db.Error != nil {
		fmt.Println("Error saving URL to database:", db.Error)
		panic("Failed to save URL to database")
	}
	return shortenedUrlDetails
}

func Expand(shortened string) (string, error) {
	data := models.ShortenedURL{}
	url := models.DBObj.First(&data, "short_code = ?", shortened)
	if url.Error == nil {
		// URL found, return the original URL
		return data.LongURL, nil
	}
	// Check if the shortened URL exists in the map
	return "", fmt.Errorf("URL not found")
}
