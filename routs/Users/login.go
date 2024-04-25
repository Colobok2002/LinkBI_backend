package Users

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
)

func UsersLoginRouter(router *gin.Engine) {
	roustBase := "user/"
	router.GET(roustBase+"log-in", loginWithCredentials)
}

// @Tags Users
// loginWithCredentials godoc
// @Summary Аутентификация пользователя по логину и паролю
// @Accept json
// @Produce  json
// @Param uuid query string true "UUID пользователя"
// @Param login query string true "Зашифрованный логин пользователя"
// @Param password query string true "Зашифрованный пароль пользователя"
// @Success 200 {object} map[string]interface{} "successful response"
// @Failure 400 {object} map[string]interface{} "bad request"
// @Router /user/log-in-with-credentials [post]
func loginWithCredentials(c *gin.Context) {
	uuid := c.Query("uuid")
	login := c.Query("login")
	password := c.Query("password")

	if uuid == "" || login == "" || password == "" {
		c.JSON(400, gin.H{"error": "UUID, login and password are required"})
		return
	}

	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
	})

	privateKeyStr, err := client.Get(uuid + "_private_key").Result()
	if err != nil {
		if err == redis.Nil {
			c.JSON(404, gin.H{"error": "Private key not found"})
		} else {
			fmt.Println("Error getting private key from Redis:", err)
			c.JSON(500, gin.H{"error": "Internal server error"})
		}
		return
	}

	block, _ := pem.Decode([]byte(privateKeyStr))
	if block == nil {
		c.JSON(500, gin.H{"error": "Failed to decode private key"})
		return
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to parse private key"})
		return
	}

	decryptedLogin, err := rsa.DecryptPKCS1v15(rand.Reader, privateKey, []byte(login))
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to decrypt login"})
		return
	}

	// decryptedPassword, err := rsa.DecryptPKCS1v15(rand.Reader, privateKey, []byte(password))
	// if err != nil {
	// 	c.JSON(500, gin.H{"error": "Failed to decrypt password"})
	// 	return
	// }

	c.JSON(200, gin.H{"message": "User authenticated successfully", "login": string(decryptedLogin)})
}
