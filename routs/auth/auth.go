package auth

import (
	"Bmessage_backend/database"
	"Bmessage_backend/helpers"
	"Bmessage_backend/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func AuthRouter(router *gin.Engine) {
	roustBase := "auth/"
	router.POST(roustBase+"get-tokens", database.WithDatabase(GetTokens))
	// router.POST(roustBase+"registration", database.WithDatabase(registerUser))
	// router.POST(roustBase+"chek-token", database.WithDatabase(chekTokenUser))
	// router.POST(roustBase+"check-uniqueness-registration-data", database.WithDatabase(check_uniqueness_registration_data))
}

// GuidTokens represents the JSON structure for a user registration request
// @Description структура для получения токена.
type GUIDTokens struct {
	GUID string `json:"GUID"`
}

// Эндпоинт для получения токенов
// @Tags Auth
// GetTokens godoc
// @Summary Эндпойт для получения пары токенов
// @Accept json
// @Produce json
// @Param data body GUIDTokens true "GUID пользователя"
// @Success 200 {object} map[string]interface{} "successful response"
// @Failure 400 {object} map[string]interface{} "bad request"
// @Router /auth/get-tokens [post]
func GetTokens(db *gorm.DB, c *gin.Context) {
	var requestData GUIDTokens

	// Валидация JSON
	if err := c.BindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
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

	// Генерация токенов
	accessToken, refreshToken, err := helpers.GenerateTokens(&user, clientIP)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate tokens"})
		return
	}

	// Хеширование Refresh токена
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
		"refresh_token": refreshToken, // Возвращаем только base64, не хеш
	})
}

// RefreshTokens обрабатывает запрос на обновление токенов
func RefreshTokens(db *gorm.DB, c *gin.Context) {
	var requestData struct {
		RefreshToken string `json:"refresh_token"`
	}

	// Валидация JSON
	if err := c.BindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// Получение IP-адреса клиента
	clientIP := c.ClientIP()

	// Поиск пользователя по Refresh Token
	var user models.User
	if err := db.First(&user, "refresh_token IS NOT NULL").Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Проверка совпадения IP-адреса
	if user.RefreshIP != clientIP {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "IP address mismatch"})
		return
	}

	// Проверка корректности хеша Refresh Token
	err := bcrypt.CompareHashAndPassword([]byte(user.RefreshToken), []byte(requestData.RefreshToken))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		return
	}

	// Генерация новых токенов
	accessToken, newRefreshToken, err := helpers.GenerateTokens(&user, clientIP)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate tokens"})
		return
	}

	// Хеширование нового Refresh токена
	hashedNewRefreshToken, err := bcrypt.GenerateFromPassword([]byte(newRefreshToken), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not hash new refresh token"})
		return
	}

	// Обновление хешированного Refresh токена и IP
	user.RefreshToken = string(hashedNewRefreshToken)
	user.RefreshIP = clientIP
	if err := db.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not save user"})
		return
	}

	// Возвращаем новые токены
	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": newRefreshToken, // Возвращаем только base64, не хеш
	})
}