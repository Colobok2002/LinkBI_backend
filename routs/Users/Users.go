package Users

import (
	Helpers "Bmessage_backend/helpers"
	"log"
	"net/http"
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
	dName, err := Helpers.DecryptDataWithPrivateKey(userData.Name, privateKeyPEM)
	if err != nil {
		log.Println("Error decrypting name:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decrypting name"})
		return
	}
	dSoName, err := Helpers.DecryptDataWithPrivateKey(userData.SoName, privateKeyPEM)
	if err != nil {
		log.Println("Error decrypting soName:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decrypting soName"})
		return
	}
	dNik, err := Helpers.DecryptDataWithPrivateKey(userData.Nik, privateKeyPEM)
	if err != nil {
		log.Println("Error decrypting nik:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decrypting nik"})
		return
	}
	dLogin, err := Helpers.DecryptDataWithPrivateKey(userData.Login, privateKeyPEM)
	if err != nil {
		log.Println("Error decrypting login:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decrypting login"})
		return
	}
	dPassword, err := Helpers.DecryptDataWithPrivateKey(userData.Password, privateKeyPEM)
	if err != nil {
		log.Println("Error decrypting password:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decrypting password"})
		return
	}

	clusterIP := os.Getenv("CLUSTER_IP")
	cluster := gocql.NewCluster(clusterIP)
	cluster.Keyspace = "bmessage_system"
	session, err := cluster.CreateSession()
	if err != nil {
		log.Printf("Failed to connect to database: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect to database"})
		return
	}
	defer session.Close()

	log.Printf("213 %v", dNik)
	log.Printf("213 %v", dLogin)

	// Check if nik or login already exists in the database
	var count int
	query := `SELECT COUNT(*) FROM bmessage_system.users WHERE Nik = ? OR login = ?`
	if err := session.Query(query, dNik, dLogin).Consistency(gocql.Quorum).Scan(&count); err != nil {
		log.Printf("Failed to query existing user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check user uniqueness"})
		return
	}
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "User with given nik or login already exists"})
		return
	}

	// Insert new user
	id := uuid.New()
	if err := session.Query(`INSERT INTO users (id, Name, SoName, Nik, login, password, PrivateKey) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, dName, dSoName, dNik, dLogin, dPassword, privateKeyPEM).Exec(); err != nil {
		log.Printf("Failed to create user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User registered successfully"})
}
