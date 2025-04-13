package main

import (
	"shortener/configs"
	"shortener/controllers"
)

func main() {
	configs.InitConfig()
	controllers.Init()
	controllers.InitUrls()
}
