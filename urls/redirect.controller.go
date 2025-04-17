package urls

import (
	"fmt"
	"shortener/cache"
	"shortener/models"

	"github.com/gofiber/fiber/v2"
)

func RedirectUrl(ctx *fiber.Ctx) error {
	code := ctx.Params("code")

	val, err := cache.GetFromCache(code)
	if err == nil && val != "" {
		urlEvent := models.UrlRedirect{
			ShortCode: code,
			Agent:     ctx.Get("User-Agent"),
			IPAddress: ctx.IP(),
			Metadata:  make(map[string]interface{}),
		}
		go prepareUrlEvent(code, urlEvent)
		return ctx.Status(fiber.StatusSeeOther).Redirect(val)
	}
	urlDetails, err := Expand(code)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "URL not found",
		})
	}

	if err := cache.SetToCache(code, urlDetails.URL); err != nil {
		fmt.Println("Error setting cache:", err)
	}
	urlEvent := models.UrlRedirect{
		ShortCode: code,
		Agent:     ctx.Get("User-Agent"),
		IPAddress: ctx.IP(),
		Metadata:  make(map[string]interface{}),
	}
	go prepareUrlEvent(code, urlEvent)
	return ctx.Status(fiber.StatusSeeOther).Redirect(urlDetails.URL)
}

func prepareUrlEvent(code string, urlEvent models.UrlRedirect) {
	urlDetails, err := GetDetailsForCode(code)
	if err != nil {
		fmt.Println("Error getting URL details:", err)
		return
	}
	urlEvent.ShortCode = urlDetails.ShortCode
	urlEvent.ShortUrlID = urlDetails.Id
	urlEvent.CreatedBy = urlDetails.CreatedBy

	err = SaveUrlEvent(urlEvent)
	if err != nil {
		fmt.Println("Error saving URL event:", err)
		return
	}
}
