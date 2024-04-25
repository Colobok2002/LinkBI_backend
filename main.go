package main

import (
	"Bmessage_backend/routs/swagger"
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func main() {
	router := swagger.SetupRouter()

	// Swwagger

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/swagger-docs")))

	router.GET("/swagger-docs", func(c *gin.Context) {
		c.File("./docs/swagger.json")
	})

	router.GET("/docs", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/swagger/index.html")
	})

	router.Run(":8080")
}
