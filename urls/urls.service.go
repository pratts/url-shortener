package urls

import (
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"net/url"
	"shortener/cache"
	"shortener/configs"
	"shortener/db"
	"shortener/models"
	"strings"
)

const (
	base62     = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	codeLength = 7
	maxUrlLen  = 2048
)

var ErrInvalidTargetURL = errors.New("url must be an absolute http(s) URL of at most 2048 characters, without credentials, and not pointing to this service")

// generateCode returns a random base62 code from crypto/rand. Bytes >= 248
// are rejected so every character is equally likely (248 = 4 * 62).
func generateCode() (string, error) {
	code := make([]byte, 0, codeLength)
	buf := make([]byte, codeLength*2)
	for len(code) < codeLength {
		if _, err := rand.Read(buf); err != nil {
			return "", err
		}
		for _, b := range buf {
			if b >= 248 {
				continue
			}
			code = append(code, base62[int(b)%len(base62)])
			if len(code) == codeLength {
				break
			}
		}
	}
	return string(code), nil
}

// ValidateTargetURL checks that a URL is safe to redirect to and returns it
// with surrounding whitespace removed.
func ValidateTargetURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || len(raw) > maxUrlLen {
		return "", ErrInvalidTargetURL
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", ErrInvalidTargetURL
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", ErrInvalidTargetURL
	}
	if u.Hostname() == "" || u.User != nil {
		return "", ErrInvalidTargetURL
	}
	if own, err := url.Parse(configs.AppConfig.ApiUrl); err == nil && strings.EqualFold(own.Hostname(), u.Hostname()) {
		return "", ErrInvalidTargetURL
	}
	return raw, nil
}

func invalidateCache(code string) {
	if err := cache.DeleteFromCache(cache.UrlKey(code)); err != nil {
		log.Printf("failed to invalidate cache for %s, it will expire after the TTL: %v", code, err)
	}
}

func CreateShortCode(url string, userId uint64) (models.UrlDto, error) {
	code, err := generateCode()
	if err != nil {
		return models.UrlDto{}, err
	}
	shortenedUrlDetails := models.ShortenedURL{
		ShortCode: code,
		LongURL:   url,
		CreatedBy: userId,
	}

	db := db.DBObj.Create(&shortenedUrlDetails)
	if db.Error != nil {
		return models.UrlDto{}, db.Error
	}

	// Create a URL DTO to return
	shortenedUrlDetailsDto := models.UrlDto{
		Id:        shortenedUrlDetails.Id,
		URL:       shortenedUrlDetails.LongURL,
		ShortCode: shortenedUrlDetails.ShortCode,
		ShortUrl:  fmt.Sprintf("%s/%s", configs.AppConfig.ApiUrl, shortenedUrlDetails.ShortCode),
	}
	return shortenedUrlDetailsDto, nil
}

func GetAllShortCodes(userId uint64) ([]models.UrlDto, error) {
	var urls []models.ShortenedURL
	db := db.DBObj.Where("created_by = ?", userId).Find(&urls)
	if db.Error != nil {
		return nil, db.Error
	}

	var urlDtos []models.UrlDto
	for _, url := range urls {
		urlDtos = append(urlDtos, models.UrlDto{
			Id:        url.Id,
			URL:       url.LongURL,
			ShortCode: url.ShortCode,
			ShortUrl:  fmt.Sprintf("%s/%s", configs.AppConfig.ApiUrl, url.ShortCode),
			CreatedAt: url.CreatedAt.String(),
		})
	}
	return urlDtos, nil
}

func GetUrlDetails(id uint64, userId uint64) (models.UrlDto, error) {
	var url models.ShortenedURL
	db := db.DBObj.Where("id = ? AND created_by = ?", id, userId).First(&url)
	if db.Error != nil {
		return models.UrlDto{}, db.Error
	}

	urlDto := models.UrlDto{
		Id:        url.Id,
		URL:       url.LongURL,
		ShortCode: url.ShortCode,
		ShortUrl:  fmt.Sprintf("%s/%s", configs.AppConfig.ApiUrl, url.ShortCode),
	}
	return urlDto, nil
}

func UpdateUrl(id uint64, urlInput models.UrlInput, userId uint64) (models.UrlDto, error) {
	var url models.ShortenedURL
	result := db.DBObj.Where("id = ? AND created_by = ?", id, userId).First(&url)
	if result.Error != nil {
		return models.UrlDto{}, result.Error
	}

	url.LongURL = urlInput.URL
	result = db.DBObj.Save(&url)
	if result.Error != nil {
		return models.UrlDto{}, result.Error
	}
	invalidateCache(url.ShortCode)

	urlDto := models.UrlDto{
		Id:        url.Id,
		URL:       url.LongURL,
		ShortCode: url.ShortCode,
		ShortUrl:  fmt.Sprintf("%s/%s", configs.AppConfig.ApiUrl, url.ShortCode),
	}
	return urlDto, nil
}

func DeleteUrl(id uint64, userId uint64) error {
	var url models.ShortenedURL
	result := db.DBObj.Where("id = ? AND created_by = ?", id, userId).First(&url)
	if result.Error != nil {
		return result.Error
	}

	result = db.DBObj.Delete(&url)
	if result.Error != nil {
		return result.Error
	}
	invalidateCache(url.ShortCode)
	return nil
}

func Expand(shortened string) (models.UrlDto, error) {
	data := models.ShortenedURL{}
	url := db.DBObj.First(&data, "short_code = ?", shortened)
	if url.Error == nil {
		// URL found, return the original URL
		urlDto := models.UrlDto{
			Id:        data.Id,
			URL:       data.LongURL,
			ShortCode: data.ShortCode,
			ShortUrl:  fmt.Sprintf("%s/%s", configs.AppConfig.ApiUrl, data.ShortCode),
		}
		return urlDto, nil
	}
	// Check if the shortened URL exists in the map
	return models.UrlDto{}, fmt.Errorf("URL not found")
}

func GetDetailsForCode(code string) (models.UrlDto, error) {
	data := models.ShortenedURL{}
	url := db.DBObj.First(&data, "short_code = ?", code)
	if url.Error == nil {
		// URL found, return the original URL
		urlDto := models.UrlDto{
			Id:        data.Id,
			URL:       data.LongURL,
			ShortCode: data.ShortCode,
			ShortUrl:  fmt.Sprintf("%s/%s", configs.AppConfig.ApiUrl, data.ShortCode),
			CreatedBy: data.CreatedBy,
		}
		return urlDto, nil
	}
	return models.UrlDto{}, fmt.Errorf("URL not found")
}

func SaveUrlEvent(urlEvent models.UrlRedirect) error {
	db := db.DBObj.Create(&urlEvent)
	if db.Error != nil {
		return db.Error
	}
	return nil
}
