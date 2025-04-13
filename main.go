package main

import (
	"shortener/configs"
	"shortener/controllers"
	"shortener/models"
)

func main() {
	configs.InitConfig()
	models.InitDb()
	controllers.Init()
	controllers.InitUrls()
}
