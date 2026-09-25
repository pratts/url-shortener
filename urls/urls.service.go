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
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	base62          = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	codeLength      = 7
	maxUrlLen       = 2048
	maxCodeAttempts = 5
)

var (
	ErrInvalidTargetURL   = errors.New("url must be an absolute http(s) URL of at most 2048 characters, without credentials, and not pointing to this service")
	ErrUrlNotFound        = errors.New("url not found")
	ErrCodeSpaceExhausted = errors.New("could not generate a unique short code")
)

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

// newCode generates short codes; tests replace it to force collisions.
var newCode = generateCode

func toDto(u models.ShortenedURL) models.UrlDto {
	return models.UrlDto{
		Id:        u.Id,
		URL:       u.LongURL,
		ShortCode: u.ShortCode,
		ShortUrl:  fmt.Sprintf("%s/%s", configs.AppConfig.ApiUrl, u.ShortCode),
		CreatedAt: u.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: u.UpdatedAt.UTC().Format(time.RFC3339),
		CreatedBy: u.CreatedBy,
	}
}

// notFound maps GORM's not-found error to ErrUrlNotFound and passes other
// errors through.
func notFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrUrlNotFound
	}
	return err
}

// CreateShortCode stores url under a new random code. The unique index on
// short_code rejects collisions, in which case a new code is tried.
func CreateShortCode(url string, userId uint64) (models.UrlDto, error) {
	for attempt := 0; attempt < maxCodeAttempts; attempt++ {
		code, err := newCode()
		if err != nil {
			return models.UrlDto{}, err
		}
		shortened := models.ShortenedURL{
			ShortCode: code,
			LongURL:   url,
			CreatedBy: userId,
		}
		err = db.DBObj.Create(&shortened).Error
		if err == nil {
			return toDto(shortened), nil
		}
		if !db.IsUniqueViolation(err, "short_code") {
			return models.UrlDto{}, err
		}
		log.Printf("short code collision on attempt %d, retrying", attempt+1)
	}
	return models.UrlDto{}, ErrCodeSpaceExhausted
}

// ListShortCodes returns up to limit of the user's URLs, newest first,
// starting after cursor (an id; 0 means from the newest). nextCursor is 0 when
// there are no more results.
func ListShortCodes(userId uint64, limit int, cursor uint64) (urls []models.UrlDto, nextCursor uint64, err error) {
	query := db.DBObj.Where("created_by = ?", userId)
	if cursor > 0 {
		query = query.Where("id < ?", cursor)
	}
	var rows []models.ShortenedURL
	// Fetch one extra row to learn whether another page exists.
	if err := query.Order("id DESC").Limit(limit + 1).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	if len(rows) > limit {
		rows = rows[:limit]
		nextCursor = rows[limit-1].Id
	}
	urls = make([]models.UrlDto, 0, len(rows))
	for _, row := range rows {
		urls = append(urls, toDto(row))
	}
	return urls, nextCursor, nil
}

func GetUrlDetails(id uint64, userId uint64) (models.UrlDto, error) {
	var url models.ShortenedURL
	if err := db.DBObj.Where("id = ? AND created_by = ?", id, userId).First(&url).Error; err != nil {
		return models.UrlDto{}, notFound(err)
	}
	return toDto(url), nil
}

// UpdateUrl changes the long URL in one statement, returning the updated row.
func UpdateUrl(id uint64, urlInput models.UrlInput, userId uint64) (models.UrlDto, error) {
	var url models.ShortenedURL
	result := db.DBObj.Model(&url).Clauses(clause.Returning{}).
		Where("id = ? AND created_by = ?", id, userId).
		Update("long_url", urlInput.URL)
	if result.Error != nil {
		return models.UrlDto{}, result.Error
	}
	if result.RowsAffected == 0 {
		return models.UrlDto{}, ErrUrlNotFound
	}
	invalidateCache(url.ShortCode)
	return toDto(url), nil
}

// DeleteUrl deletes in one statement, returning the row so its cache entry
// can be removed.
func DeleteUrl(id uint64, userId uint64) error {
	var url models.ShortenedURL
	result := db.DBObj.Clauses(clause.Returning{}).
		Where("id = ? AND created_by = ?", id, userId).
		Delete(&url)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrUrlNotFound
	}
	invalidateCache(url.ShortCode)
	return nil
}

func Expand(shortened string) (models.UrlDto, error) {
	var data models.ShortenedURL
	if err := db.DBObj.First(&data, "short_code = ?", shortened).Error; err != nil {
		return models.UrlDto{}, notFound(err)
	}
	return toDto(data), nil
}

func GetDetailsForCode(code string) (models.UrlDto, error) {
	return Expand(code)
}

func SaveUrlEvent(urlEvent models.UrlRedirect) error {
	return db.DBObj.Create(&urlEvent).Error
}
