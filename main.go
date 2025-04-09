package main

import (
	"shortener/configs"
	"shortener/models"
	shortener "shortener/urls"
)

func main() {
	configs.InitConfig()
	models.InitDb()
	shortener.Init()
}
