package Models

import (
	"fmt"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Определение модели User
type User struct {
	gorm.Model  
	Name       string `gorm:"column:name"`
	SoName     string `gorm:"column:so_name"`
	Nik        string `gorm:"column:nik;unique"`
	Login      string `gorm:"column:login;unique"`
	Password   string `gorm:"column:password"`
	PrivateKey string `gorm:"column:private_key"`
}

func MigrationUsertabel() {

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
		log.Fatalf("failed to connect database: %v", err)
	}

	if err := db.AutoMigrate(&User{}); err != nil {
		log.Fatalf("failed to migrate: %v", err)
	}

	fmt.Println("Migration executed successfully")
}
