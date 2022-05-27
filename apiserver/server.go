package apiserver

import (
	"github.com/gin-gonic/gin"
)

func RunServer() {
	r := gin.Default()
	NewApiController(r).Router()

	r.Run(":8080")
}

