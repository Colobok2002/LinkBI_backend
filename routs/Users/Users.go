package users

import (
	database "Bmessage_backend/database"
	helpers "Bmessage_backend/helpers"
	models "Bmessage_backend/models"
	tokens "Bmessage_backend/routs/tokens"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
	"golang.org/x/crypto/bcrypt"
)

func UsersRouter(router *gin.Engine) {
	roustBase := "user/"
	router.POST(roustBase+"log-in-with-credentials", loginWithCredentials)
	router.POST(roustBase+"registration", registerUser)
	router.POST(roustBase+"chek-token", chekTokenUser)
}

// ResponseMessage defines a standard response message structure.
type ResponseMessage struct {
	Status  bool                   `json:"status"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"Data"`
}

// ErrorResponse defines a standard error response structure.
type ErrorResponse struct {
	Status bool   `json:"status"`
	Error  string `json:"error"`
}

// UserLogin represents the JSON structure for a user registration request
// @Description User login data structure.
type UserLogin struct {
	Uuid     string `json:"uuid"`
	PKey     string `json:"pKey"`
	Login    string `json:"login" example:"john_doe"`
	Password string `json:"password" example:"securePassword123"`
}

// @Tags Users
// loginWithCredentials godoc
// @Summary Аутентификация пользователя по логину и паролю
// @Accept json
// @Produce  json
// @Param data body UserLogin true "Данные пользователя"
// @Success 200 {object} map[string]interface{} "successful response"
// @Failure 400 {object} map[string]interface{} "bad request"
// @Router /user/log-in-with-credentials [post]
func loginWithCredentials(c *gin.Context) {
	var userData UserLogin

	if err := c.BindJSON(&userData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
	})

	privateKeyPEM, err := client.Get(userData.Uuid + "_private_key").Result()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": ""})
		return
	}

	dLogin, err := helpers.DecryptWithPrivateKey(userData.Login, privateKeyPEM)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return

	}

	dPassword, err := helpers.DecryptWithPrivateKey(userData.Password, privateKeyPEM)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка дешифрования пароля"})
		return
	}

	db, err := database.GetDb()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка подключения к базе данных, попробуйте позже"})
		return
	}

	var user models.User

	if err := db.Where("login = ?", dLogin).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Пользователь не найден"})
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(dPassword)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Неверный пароль"})
		return
	}

	TokenUserData := map[string]interface{}{
		"userId": user.ID,
	}

	token, err := helpers.EncryptAES(TokenUserData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Внутренняя ошибка сервера"})
		return
	}

	сToken, err := helpers.EncryptWithPublicKey(token, userData.PKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Внутренняя ошибка сервера"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Пользователь успешно аутентифицирован", "token": сToken})
}

// UserRegistration represents the JSON structure for a user registration request
// @Description User registration data structure.
type UserRegistration struct {
	Uuid     string `json:"uuid"`
	Name     string `json:"name" example:"John"`
	SoName   string `json:"soName" example:"Doe"`
	Nik      string `json:"nik" example:"JohnnyD"`
	Login    string `json:"login" example:"john_doe"`
	Password string `json:"password" example:"securePassword123"`
	PKey     string `json:"pKey"`
}

// registerUser godoc
// @Tags Users
// @Summary Регистрация нового пользователя
// @Description Регистрирует пользователя, сохраняя зашифрованные данные и ключ в базу данных.
// @Accept json
// @Produce json
// @Param data body UserRegistration true "Данные пользователя"
// @Success 200 {object} ResponseMessage "User registered successfully"
// @Failure 400 {object} ErrorResponse "Invalid request data"
// @Failure 500 {object} ErrorResponse "Failed to connect to database"
// @Router /user/registration [post]
func registerUser(c *gin.Context) {
	var userData UserRegistration

	if err := c.BindJSON(&userData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные входные данные"})
		return
	}

	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
	})

	privateKeyPEM, err := client.Get(userData.Uuid + "_private_key").Result()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Внутренняя ошибка сервера"})
		return
	}

	dName, err := helpers.DecryptWithPrivateKey(userData.Name, privateKeyPEM)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка дешифрования имени"})
		return
	}
	dSoName, err := helpers.DecryptWithPrivateKey(userData.SoName, privateKeyPEM)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка дешифрования фамилии"})
		return
	}

	dNik, err := helpers.DecryptWithPrivateKey(userData.Nik, privateKeyPEM)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка дешифрования ника"})
		return
	}

	dLogin, err := helpers.DecryptWithPrivateKey(userData.Login, privateKeyPEM)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка дешифрования логина"})
		return
	}
	dPassword, err := helpers.DecryptWithPrivateKey(userData.Password, privateKeyPEM)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка дешифрования пароля"})
		return
	}

	bytesPass, err := bcrypt.GenerateFromPassword([]byte(dPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка хеширования пароля"})
		return
	}

	db, err := database.GetDb()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка подключения к базе данных, попробуйте позже"})
		return
	}

	var count int64
	if err := db.Model(&models.User{}).Where("nik = ? OR login = ?", dNik, dLogin).Count(&count).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка проверки уникальности пользователя"})
		return
	}

	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Пользователь с таким ником или логином уже существует"})
		return
	}

	newUser := &models.User{
		Name:       dName,
		SoName:     dSoName,
		Nik:        dNik,
		Login:      dLogin,
		Password:   string(bytesPass),
		PrivateKey: tokens.GeneratePrivateKey(),
	}

	if err := db.Create(newUser).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка регистрации пользователя"})
		return
	}

	TokenUserData := map[string]interface{}{
		"userId": newUser.ID,
	}

	token, err := helpers.EncryptAES(TokenUserData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Внутренняя ошибка сервера"})
		return
	}

	сToken, err := helpers.EncryptWithPublicKey(token, userData.PKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Внутренняя ошибка сервера"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Пользователь успешно зарегистрирован", "token": сToken})
}

// UserCT represents the JSON structure for a user registration request
// @Description User chekToken data structure.
type UserCT struct {
	Uuid  string `json:"uuid"`
	PKey  string `json:"pKey"`
	Token string `json:"token"`
}

// registerUser godoc
// @Tags Users
// @Summary Проверка токена
// @Description Проверяет валидность пользователя => сессии
// @Accept json
// @Produce json
// @Param data body UserCT true "Данные о токене"
// @Success 200 {object} ResponseMessage "User registered successfully"
// @Failure 400 {object} ErrorResponse "Invalid request data"
// @Failure 500 {object} ErrorResponse "Failed to connect to database"
// @Router /user/chek-token [post]
func chekTokenUser(c *gin.Context) {
	var userData UserCT

	if err := c.BindJSON(&userData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
	})

	privateKeyPEM, err := client.Get(userData.Uuid + "_private_key").Result()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": ""})
		return
	}

	dToken, err := helpers.DecryptWithPrivateKey(userData.Token, privateKeyPEM)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return

	}

	userDataToToken, err := helpers.DecryptAES(dToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return

	}

	log.Println(userDataToToken)

	c.JSON(http.StatusOK, gin.H{"message": "Пользователь успешно зарегистрирован", "toke": userData})
}
