package helpers

import (
	"Bmessage_backend/models"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"os"
	"time"

	"github.com/dgrijalva/jwt-go"
)

// Структура для создания JWT Access Token
type TokenPayload struct {
	GUID string `json:"guid"`
	IP   string `json:"ip"`
	jwt.StandardClaims
}

// Генерация Access и Refresh токенов
func GenerateTokens(user *models.User, clientIP string) (string, string, error) {
	// Генерация Access токена
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS512, &TokenPayload{
		GUID: user.GUID,
		IP:   clientIP,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(15 * time.Minute).Unix(), // Access токен живет 15 минут
		},
	})
	accessSecret := os.Getenv("ACCESS_SECRET") // Секретный ключ для Access токена
	accessTokenString, err := accessToken.SignedString([]byte(accessSecret))
	if err != nil {
		return "", "", err
	}

	// Генерация Refresh токена
	refreshToken, _ := generateRandomString(32) // Генерация случайной строки для Refresh токена
	err = user.SetRefreshToken(refreshToken)    // Сохранение хешированного Refresh токена в базе
	if err != nil {
		return "", "", err
	}

	return accessTokenString, refreshToken, nil
}

// Функция генерации случайной строки для Refresh токена
// Генерация случайной строки нужной длины
func generateRandomString(length int) (string, error) {
	// Параметры для генерации случайных байтов
	if length <= 0 {
		return "", errors.New("invalid length")
	}

	// Генерация случайных байтов
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	// Преобразование байтов в строку Base64
	randomString := base64.StdEncoding.EncodeToString(bytes)

	// Обрезаем строку до желаемой длины, если необходимо
	if len(randomString) > length {
		randomString = randomString[:length]
	}

	return randomString, nil
}
