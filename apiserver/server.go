package apiserver

import (
	"github.com/gin-gonic/gin"
)

func RunServer() {
	r := gin.Default()
	NewApiController(r).Router()

	r.Run("0.0.0.0:8080")
}
