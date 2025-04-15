package urls

import (
	"fmt"
	"math/rand"
	"shortener/configs"
	"shortener/models"
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
	}
}

func CreateShortCode(url string, userId uint64) models.UrlDto {
	code := generateCode()
	if code == "" {
		panic("Failed to generate a unique URL code")
	}
	shortenedUrlDetails := models.ShortenedURL{
		ShortCode: code,
		LongURL:   url,
		CreatedBy: userId,
	}

	db := models.DBObj.Create(&shortenedUrlDetails)
	if db.Error != nil {
		panic("Failed to save URL to database")
	}

	// Create a URL DTO to return
	shortenedUrlDetailsDto := models.UrlDto{
		Id:       shortenedUrlDetails.Id,
		URL:      shortenedUrlDetails.LongURL,
		ShortUrl: fmt.Sprintf("%s/%s", configs.AppConfig.ApiUrl, shortenedUrlDetails.ShortCode),
	}
	return shortenedUrlDetailsDto
}

func GetAllShortCodes(userId uint64) ([]models.UrlDto, error) {
	var urls []models.ShortenedURL
	db := models.DBObj.Where("created_by = ?", userId).Find(&urls)
	if db.Error != nil {
		return nil, db.Error
	}

	var urlDtos []models.UrlDto
	for _, url := range urls {
		urlDtos = append(urlDtos, models.UrlDto{
			Id:       url.Id,
			URL:      url.LongURL,
			ShortUrl: fmt.Sprintf("%s/%s", configs.AppConfig.ApiUrl, url.ShortCode),
		})
	}
	return urlDtos, nil
}

func GetUrlDetails(code string, userId uint64) (models.UrlDto, error) {
	var url models.ShortenedURL
	db := models.DBObj.Where("short_code = ? AND created_by = ?", code, userId).First(&url)
	if db.Error != nil {
		return models.UrlDto{}, db.Error
	}

	urlDto := models.UrlDto{
		Id:       url.Id,
		URL:      url.LongURL,
		ShortUrl: fmt.Sprintf("%s/%s", configs.AppConfig.ApiUrl, url.ShortCode),
	}
	return urlDto, nil
}

func UpdateUrl(code string, urlInput models.UrlInput, userId uint64) (models.UrlDto, error) {
	var url models.ShortenedURL
	db := models.DBObj.Where("short_code = ? AND created_by = ?", code, userId).First(&url)
	if db.Error != nil {
		return models.UrlDto{}, db.Error
	}

	url.LongURL = urlInput.URL
	db = models.DBObj.Save(&url)
	if db.Error != nil {
		return models.UrlDto{}, db.Error
	}

	urlDto := models.UrlDto{
		Id:       url.Id,
		URL:      url.LongURL,
		ShortUrl: fmt.Sprintf("%s/%s", configs.AppConfig.ApiUrl, url.ShortCode),
	}
	return urlDto, nil
}

func DeleteUrl(code string, userId uint64) error {
	var url models.ShortenedURL
	db := models.DBObj.Where("short_code = ? AND created_by = ?", code, userId).First(&url)
	if db.Error != nil {
		return db.Error
	}

	db = models.DBObj.Delete(&url)
	if db.Error != nil {
		return db.Error
	}
	return nil
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
