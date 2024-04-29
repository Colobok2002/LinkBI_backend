package models

import (
	Database "Bmessage_backend/database"
	"fmt"
	"log"

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

	db, err := Database.GetDb()
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	if err := db.AutoMigrate(&User{}); err != nil {
		log.Fatalf("failed to migrate: %v", err)
	}

	fmt.Println("Migration executed successfully")
	sqlDB, err := db.DB()
	sqlDB.Close()
}
