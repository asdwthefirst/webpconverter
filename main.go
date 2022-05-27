package main

import (
	"webpconverter/apiserver"
	"webpconverter/logger"
)

func main() {

	logger.Init()

	apiserver.RunServer()
}
