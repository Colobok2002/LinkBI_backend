package auth

import (
	"Bmessage_backend/database"
	"Bmessage_backend/helpers"
	"Bmessage_backend/models"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// Конфигурация для JWT токенов
var jwtSecretKey = []byte(os.Getenv("ACCESS_SECRET"))

// TokenPayload структура для payload в Access токене
type TokenPayload struct {
	GUID string `json:"guid"`
	IP   string `json:"ip"`
	jwt.StandardClaims
}

// AuthRouter регистрирует маршруты аутентификации
func AuthRouter(router *gin.Engine) {
	roustBase := "auth/"
	router.GET(roustBase+"get-tokens", database.WithDatabase(GetTokens))
	router.GET(roustBase+"refresh-tokens", database.WithDatabase(RefreshTokens))
}

// GUIDTokens представляет структуру для получения токенов
type GUIDTokens struct {
	GUID string `json:"GUID"`
}

// GetTokens генерирует токены для пользователя
// @Summary Получение токенов по GUID пользователя
// @Description Эндпойнт для генерации access и refresh токенов на основе GUID пользователя.
// @Tags Auth
// @Accept json
// @Produce json
// @Param GUID query string true "GUID пользователя"
// @Success 200 {object} map[string]interface{} "successful response"
// @Failure 400 {object} map[string]interface{} "bad request"
// @Failure 500 {object} map[string]interface{} "internal server error"
// @Router /auth/get-tokens [get]
func GetTokens(db *gorm.DB, c *gin.Context) {
	var requestData GUIDTokens

	// Получение GUID из запроса
	requestData.GUID = c.Query("GUID")
	if requestData.GUID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "GUID is required"})
		return
	}

	// Получение IP-адреса клиента
	clientIP := c.ClientIP()

	// Поиск пользователя по GUID
	var user models.User
	if err := db.Where("guid = ?", requestData.GUID).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// Если пользователь не найден, создаем нового
			user = models.User{
				GUID:      requestData.GUID,
				Email:     "mock@example.com", // Моковый email для уведомлений
				RefreshIP: clientIP,
			}
			// Сохранение пользователя в базе
			if err := db.Create(&user).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create user"})
				return
			}
		} else {
			// Если произошла другая ошибка
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
			return
		}
	}

	// Генерация Access и Refresh токенов
	accessToken, refreshToken, err := helpers.GenerateTokens(&user, clientIP)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate tokens"})
		return
	}

	// Хеширование Refresh токена с использованием bcrypt
	hashedRefreshToken, err := bcrypt.GenerateFromPassword([]byte(refreshToken), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not hash refresh token"})
		return
	}

	// Сохранение хешированного Refresh токена и IP
	user.RefreshToken = string(hashedRefreshToken)
	user.RefreshIP = clientIP
	if err := db.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not save user"})
		return
	}

	// Возвращаем токены
	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": base64.StdEncoding.EncodeToString([]byte(refreshToken)), // Возвращаем Refresh токен в base64 формате
	})
}

// RefreshTokens обрабатывает запрос на обновление токенов
// @Summary Обновление токенов по Access и Refresh токенам
// @Description Эндпойнт для обновления access и refresh токенов на основе существующего refresh токена.
// @Tags Auth
// @Accept json
// @Produce json
// @Param access_token query string true "Access токен"
// @Param refresh_token query string true "Refresh токен"
// @Success 200 {object} map[string]interface{} "successful response"
// @Failure 400 {object} map[string]interface{} "bad request"
// @Failure 401 {object} map[string]interface{} "unauthorized"
// @Failure 500 {object} map[string]interface{} "internal server error"
// @Router /auth/refresh-tokens [get]
func RefreshTokens(db *gorm.DB, c *gin.Context) {
	// Получение токенов из запроса
	accessToken := c.Query("access_token")
	refreshToken := c.Query("refresh_token")

	if accessToken == "" || refreshToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Both access and refresh tokens are required"})
		return
	}

	// Проверка валидности Access токена
	token, err := jwt.ParseWithClaims(accessToken, &TokenPayload{}, func(token *jwt.Token) (interface{}, error) {
		// Убедись, что метод подписи совпадает с ожиданием (HS512)
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecretKey, nil
	})

	if err != nil || !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid access token", "token": err.Error()})
		return
	}

	claims, ok := token.Claims.(*TokenPayload)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
		return
	}

	// Поиск пользователя по GUID
	var user models.User
	if err := db.Where("guid = ?", claims.GUID).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user"})
		return
	}

	// Проверка соответствия refresh токена с хешем
	decodedRefreshToken, err := base64.StdEncoding.DecodeString(refreshToken)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid refresh token format"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.RefreshToken), decodedRefreshToken); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		return
	}

	// Проверка совпадения IP-адреса
	clientIP := c.ClientIP()
	if user.RefreshIP != clientIP {
		// Отправка email предупреждения
		helpers.SendEmailWarning(user.Email, clientIP)
	}

	// Генерация новой пары токенов
	newAccessToken, newRefreshToken, err := helpers.GenerateTokens(&user, clientIP)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate new tokens"})
		return
	}

	// Хеширование нового Refresh токена
	hashedNewRefreshToken, err := bcrypt.GenerateFromPassword([]byte(newRefreshToken), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not hash new refresh token"})
		return
	}

	// Обновление Refresh токена в базе
	user.RefreshToken = string(hashedNewRefreshToken)
	user.RefreshIP = clientIP
	if err := db.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not save new refresh token"})
		return
	}

	// Возвращаем новую пару токенов
	c.JSON(http.StatusOK, gin.H{
		"access_token":  newAccessToken,
		"refresh_token": base64.StdEncoding.EncodeToString([]byte(newRefreshToken)), // Возвращаем Refresh токен в base64 формате
	})
}
