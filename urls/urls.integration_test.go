package urls

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"shortener/cache"
	"shortener/configs"
	"shortener/db"
	"shortener/models"
	"strconv"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

// setupDB connects to the Postgres database named by SHORTENER_TEST_DB (on
// localhost, as $USER) and Redis DB SHORTENER_TEST_REDIS_DB (default 13).
// Tests are skipped when SHORTENER_TEST_DB is unset. Tables are emptied first.
func setupDB(t *testing.T) {
	t.Helper()
	name := os.Getenv("SHORTENER_TEST_DB")
	if name == "" {
		t.Skip("SHORTENER_TEST_DB not set")
	}
	redisDB, _ := strconv.Atoi(os.Getenv("SHORTENER_TEST_REDIS_DB"))
	if redisDB == 0 {
		redisDB = 13
	}
	configs.AppConfig.ApiUrl = "http://localhost:18085"
	configs.PgConfig = configs.POSTGRES_CONFIG{
		Host: "localhost", Port: 5432, Username: os.Getenv("USER"),
		Password: "unused", Database: name, SSLMode: "disable",
	}
	configs.RedisConfig = configs.REDIS_CONFIG{Host: "localhost", Port: 6379, Database: redisDB, TTL: 60}
	db.InitDb()
	cache.InitCache()
	if err := db.DBObj.Exec("TRUNCATE shortened_urls RESTART IDENTITY").Error; err != nil {
		t.Fatal(err)
	}
}

func TestCreateRetriesOnCodeCollision(t *testing.T) {
	setupDB(t)
	defer func() { newCode = generateCode }()

	codes := []string{"SAME001", "SAME001", "SAME001", "FRESH01"}
	calls := 0
	newCode = func() (string, error) {
		c := codes[calls]
		calls++
		return c, nil
	}

	first, err := CreateShortCode("https://example.com/1", 1)
	if err != nil || first.ShortCode != "SAME001" {
		t.Fatalf("first create: %+v, %v", first, err)
	}
	second, err := CreateShortCode("https://example.com/2", 2)
	if err != nil {
		t.Fatalf("second create should retry past collisions: %v", err)
	}
	if second.ShortCode != "FRESH01" || calls != 4 {
		t.Fatalf("got code %q after %d generations, want FRESH01 after 4", second.ShortCode, calls)
	}
	got, err := Expand("SAME001")
	if err != nil || got.URL != "https://example.com/1" {
		t.Fatalf("original link was changed: %+v, %v", got, err)
	}
}

func TestCreateGivesUpAfterMaxAttempts(t *testing.T) {
	setupDB(t)
	defer func() { newCode = generateCode }()
	newCode = func() (string, error) { return "STUCK01", nil }

	if _, err := CreateShortCode("https://example.com/a", 1); err != nil {
		t.Fatal(err)
	}
	_, err := CreateShortCode("https://example.com/b", 1)
	if !errors.Is(err, ErrCodeSpaceExhausted) {
		t.Fatalf("got %v, want ErrCodeSpaceExhausted", err)
	}
}

func TestListPagination(t *testing.T) {
	setupDB(t)
	for i := 1; i <= 7; i++ {
		if _, err := CreateShortCode(fmt.Sprintf("https://example.com/%d", i), 1); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := CreateShortCode("https://example.com/other-user", 2); err != nil {
		t.Fatal(err)
	}

	var seen []uint64
	cursor := uint64(0)
	pages := 0
	for {
		urls, next, err := ListShortCodes(1, 3, cursor)
		if err != nil {
			t.Fatal(err)
		}
		pages++
		for _, u := range urls {
			if u.CreatedBy != 1 {
				t.Fatalf("listed another user's URL: %+v", u)
			}
			seen = append(seen, u.Id)
		}
		if next == 0 {
			break
		}
		cursor = next
	}
	want := []uint64{7, 6, 5, 4, 3, 2, 1}
	if fmt.Sprint(seen) != fmt.Sprint(want) || pages != 3 {
		t.Fatalf("got ids %v over %d pages, want %v over 3", seen, pages, want)
	}

	urls, next, err := ListShortCodes(3, 50, 0)
	if err != nil || urls == nil || len(urls) != 0 || next != 0 {
		t.Fatalf("user with no URLs: got %#v, %d, %v; want empty non-nil slice", urls, next, err)
	}
}

func TestUpdateAndDeleteAreScopedToOwner(t *testing.T) {
	setupDB(t)
	created, err := CreateShortCode("https://example.com/mine", 1)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := UpdateUrl(created.Id, models.UrlInput{URL: "https://evil.example"}, 2); !errors.Is(err, ErrUrlNotFound) {
		t.Fatalf("update by another user: got %v, want ErrUrlNotFound", err)
	}
	if err := DeleteUrl(created.Id, 2); !errors.Is(err, ErrUrlNotFound) {
		t.Fatalf("delete by another user: got %v, want ErrUrlNotFound", err)
	}
	if _, err := GetUrlDetails(created.Id, 2); !errors.Is(err, ErrUrlNotFound) {
		t.Fatalf("get by another user: got %v, want ErrUrlNotFound", err)
	}

	updated, err := UpdateUrl(created.Id, models.UrlInput{URL: "https://example.com/new"}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if updated.URL != "https://example.com/new" || updated.ShortCode != created.ShortCode ||
		updated.CreatedBy != 1 || updated.CreatedAt != created.CreatedAt || updated.UpdatedAt == "" {
		t.Fatalf("update did not return the full row: %+v (created %+v)", updated, created)
	}

	cache.CacheSetWithExpiration(cache.UrlKey(created.ShortCode), "https://example.com/new", 60)
	if err := DeleteUrl(created.Id, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := cache.GetFromCache(cache.UrlKey(created.ShortCode)); err == nil {
		t.Fatal("delete did not invalidate the cache entry")
	}
	if err := DeleteUrl(created.Id, 1); !errors.Is(err, ErrUrlNotFound) {
		t.Fatalf("second delete: got %v, want ErrUrlNotFound", err)
	}
	if _, err := Expand(created.ShortCode); !errors.Is(err, ErrUrlNotFound) {
		t.Fatalf("expand deleted code: got %v, want ErrUrlNotFound", err)
	}
}

func TestDatabaseErrorsAreNot404(t *testing.T) {
	setupDB(t)
	created, err := CreateShortCode("https://example.com/x", 1)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DBObj.DB()
	sqlDB.Close()

	app := testApp()
	paths := []struct{ method, path string }{
		{"GET", fmt.Sprintf("/urls/%d", created.Id)},
		{"DELETE", fmt.Sprintf("/urls/%d", created.Id)},
		{"GET", "/urls"},
	}
	for _, p := range paths {
		resp, err := app.Test(httptest.NewRequest(p.method, p.path, nil))
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != fiber.StatusInternalServerError {
			t.Errorf("%s %s with DB down: got %d, want 500", p.method, p.path, resp.StatusCode)
		}
	}
}

func TestListEndpointPageShape(t *testing.T) {
	setupDB(t)
	for i := 1; i <= 5; i++ {
		if _, err := CreateShortCode(fmt.Sprintf("https://example.com/%d", i), 1); err != nil {
			t.Fatal(err)
		}
	}
	app := testApp()
	get := func(path string) models.UrlPage {
		t.Helper()
		resp, err := app.Test(httptest.NewRequest("GET", path, nil))
		if err != nil || resp.StatusCode != fiber.StatusOK {
			t.Fatalf("GET %s: %v, status %d", path, err, resp.StatusCode)
		}
		var page models.UrlPage
		if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
			t.Fatal(err)
		}
		return page
	}

	var ids []uint64
	path := "/urls?limit=2"
	for pages := 0; ; pages++ {
		if pages > 3 {
			t.Fatal("pagination did not terminate")
		}
		page := get(path)
		for _, u := range page.Items {
			ids = append(ids, u.Id)
		}
		if page.NextCursor == nil {
			break
		}
		path = "/urls?limit=2&cursor=" + *page.NextCursor
	}
	if fmt.Sprint(ids) != "[5 4 3 2 1]" {
		t.Fatalf("got ids %v, want [5 4 3 2 1]", ids)
	}

	resp, _ := app.Test(httptest.NewRequest("GET", "/urls", nil))
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), `"next_cursor":null`) {
		t.Fatalf("last page should have next_cursor null, got %s", body)
	}
}
