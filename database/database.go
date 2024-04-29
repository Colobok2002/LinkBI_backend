package database

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func GetDb() (*gorm.DB, error) {
	dbHost := os.Getenv("POSTGRES_HOST")
	dbPort := os.Getenv("POSTGRES_PORT")
	dbUser := os.Getenv("POSTGRES_USER")
	dbName := os.Getenv("POSTGRES_DB")
	dbPassword := os.Getenv("POSTGRES_PASSWORD")
	dbSSLMode := "disable"
	dbTimeZone := "Asia/Shanghai"

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=%s",
		dbHost, dbUser, dbPassword, dbName, dbPort, dbSSLMode, dbTimeZone)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return db, nil
}

type HandlerFunc func(db *gorm.DB, c *gin.Context)

func WithDatabase(handler HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		db, err := GetDb()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось подключиться к базе данных"})
			return
		}

		defer func() {
			sqlDB, err := db.DB()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Внутренняя ошибка сервера"})
				return
			}
			sqlDB.Close()
		}()

		handler(db, c)
	}
}
