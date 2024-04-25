package main

import (
	"Bmessage_backend/routs/Tokens"
	"Bmessage_backend/routs/Users"
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func main() {
	router := gin.Default()

	Users.UsersLoginRouter(router)
	Tokens.TokensRouter(router)

	// swagger

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/swagger-docs")))

	router.GET("/swagger-docs", func(c *gin.Context) {
		c.File("./docs/swagger.json")
	})

	router.GET("/docs", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/swagger/index.html")
	})

	router.Run(":8080")
}
