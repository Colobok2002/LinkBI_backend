package Users

import (
	"Bmessage_backend/helpers"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"log"
	"os"
	
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
	"github.com/gocql/gocql"
	"github.com/google/uuid"
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

	decryptedLogin, err := Helpers.DecryptDataWithPrivateKey(userData.Login, privateKeyPEM)
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
		c.JSON(400, gin.H{"error": "Invalid request data"})
		return
	}

	clusterIP := os.Getenv("CLUSTER_IP")
	cluster := gocql.NewCluster(clusterIP)
	cluster.Keyspace = "bmessage_system"
	session, err := cluster.CreateSession()
	if err != nil {
		log.Printf("Failed to create session: %v", err)
		c.JSON(500, gin.H{"error": "Failed to connect to database"})
		return
	}
	defer session.Close()

	id := uuid.New()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Printf("Failed to generate private key: %v", err)
		c.JSON(500, gin.H{"error": "Failed to generate private key"})
		return
	}

	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	})

	if err := session.Query(`INSERT INTO users (id, Name, SoName, Nik, login, password, PrivateKey) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, userData.Name, userData.SoName, userData.Nik, userData.Login, userData.Password, string(privateKeyPEM)).Exec(); err != nil {
		log.Printf("Failed to create user: %v", err)
		c.JSON(500, gin.H{"error": "Failed to register user"})
		return
	}

	c.JSON(200, gin.H{"message": "User registered successfully", "id": id.String()})
}
