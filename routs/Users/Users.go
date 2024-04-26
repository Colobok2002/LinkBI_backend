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
)

func UsersRouter(router *gin.Engine) {
	roustBase := "user/"
	router.POST(roustBase+"log-in-with-credentials", loginWithCredentials)
	router.POST(roustBase+"registration", registerUser)
}

// ResponseMessage defines a standard response message structure.
type ResponseMessage struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
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
		c.JSON(400, gin.H{"error": "Invalid request data"})
		return
	}

	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
	})

	privateKeyPEM, err := client.Get(userData.Uuid + "_private_key").Result()
	if err != nil {
		if err == redis.Nil {
			c.JSON(404, gin.H{"error": "Private key not found"})
		} else {
			log.Println("Error getting private key from Redis:", err)
			c.JSON(500, gin.H{"error": "Internal server error"})
		}
		return
	}

	decryptedLogin, err := helpers.DecryptDataWithPrivateKey(userData.Login, privateKeyPEM)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return

	}

	c.JSON(200, gin.H{"message": "User authenticated successfully", "login": decryptedLogin})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
	})

	privateKeyPEM, err := client.Get(userData.Uuid + "_private_key").Result()
	if err != nil {
		if err == redis.Nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Private key not found"})
		} else {
			log.Println("Error getting private key from Redis:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		}
		return
	}

	// Decrypt data
	dName, err := helpers.DecryptDataWithPrivateKey(userData.Name, privateKeyPEM)
	if err != nil {
		log.Println("Error decrypting name:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decrypting name"})
		return
	}
	dSoName, err := helpers.DecryptDataWithPrivateKey(userData.SoName, privateKeyPEM)
	if err != nil {
		log.Println("Error decrypting soName:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decrypting soName"})
		return
	}
	dNik, err := helpers.DecryptDataWithPrivateKey(userData.Nik, privateKeyPEM)
	if err != nil {
		log.Println("Error decrypting nik:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decrypting nik"})
		return
	}
	dLogin, err := helpers.DecryptDataWithPrivateKey(userData.Login, privateKeyPEM)
	if err != nil {
		log.Println("Error decrypting login:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decrypting login"})
		return
	}
	dPassword, err := helpers.DecryptDataWithPrivateKey(userData.Password, privateKeyPEM)
	if err != nil {
		log.Println("Error decrypting password:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decrypting password"})
		return
	}

	db, err := database.GetDb()

	var count int64
	if err := db.Model(&models.User{}).Where("nik = ? OR login = ?", dNik, dLogin).Count(&count).Error; err != nil {
		log.Printf("Failed to query existing user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check user uniqueness"})
		return
	}
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "User with given nik or login already exists"})
		return
	}


	newUser := models.User{
		Name:       dName,
		SoName:     dSoName,
		Nik:        dNik,
		Login:      dLogin,
		Password:   dPassword,
		PrivateKey: tokens.GeneratePrivateKey(),
	}

	if err := db.Create(newUser).Error; err != nil {
		log.Printf("Failed to create user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User registered successfully"})
}
