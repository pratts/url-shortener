package urls

import (
	"fmt"
	"math/rand"
	"shortener/configs"
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

func Shorten(url string) models.UrlDto {
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

	// Create a URL DTO to return
	shortenedUrlDetailsDto := models.UrlDto{
		Id:        fmt.Sprintf("%d", shortenedUrlDetails.Id),
		URL:       shortenedUrlDetails.LongURL,
		ShortUrl:  fmt.Sprintf("%s/%s", configs.AppConfig.ApiUrl, shortenedUrlDetails.ShortCode),
		CreatedAt: shortenedUrlDetails.CreatedAt,
		UpdatedAt: shortenedUrlDetails.UpdatedAt,
	}
	return shortenedUrlDetailsDto
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
