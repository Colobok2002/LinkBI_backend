package main

import (
	"Bmessage_backend/database"
	Models "Bmessage_backend/models"
	"Bmessage_backend/routs/chats"
	"Bmessage_backend/routs/tokens"
	"Bmessage_backend/routs/users"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func main() {
	// Init .env
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	// Migrations
	Models.MigrationUsertabel()
	database.InitScylla()

	// Routs
	router := gin.Default()
	users.UsersRouter(router)
	tokens.TokensRouter(router)
	chats.ChatRouter(router)

	// Docs
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/swagger-docs")))
	router.GET("/swagger-docs", func(c *gin.Context) {
		c.File("./docs/swagger.json")
	})
	router.GET("/docs", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/swagger/index.html")
	})

	// Start server
	serverPort := os.Getenv("SERVER_PORT")
	log.Println("Server will start on port:", serverPort)
	router.Run(":" + serverPort)
}
