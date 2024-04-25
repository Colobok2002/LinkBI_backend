package Tokens

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"golang.org/x/crypto/ssh"
)

func TokensRouter(router *gin.Engine) {
	roustBase := "token/"
	router.GET(roustBase+"generateToken", generateToken)
	router.GET(roustBase+"get-public-key", getPublicKey)
}

// @Tags Tokens
// generateToken godoc
// @Summary Получение токена и uuid
// @Produce  json
// @Success 200 {object} map[string]interface{} "successful response"
// @Router /token/generateToken [get]
func generateToken(c *gin.Context) {
	uuid := uuid.New().String()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		fmt.Println("Error generating private key:", err)
		c.JSON(500, gin.H{"error": "Internal server error"})
		return
	}

	publicKey, err := sshPublicKey(privateKey)
	if err != nil {
		fmt.Println("Error generating public key:", err)
		c.JSON(500, gin.H{"error": "Internal server error"})
		return
	}

	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
	})

	privateKeyStr := string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)}))
	publicKeyStr := string(ssh.MarshalAuthorizedKey(publicKey))

	ctx := context.Background()

	err = client.Set(ctx, uuid+"_private_key", privateKeyStr, 0).Err()
	if err != nil {
		fmt.Println("Error setting private key in Redis:", err)
		c.JSON(500, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(200, gin.H{"uuid": uuid, "public_key": publicKeyStr})
}

// @Tags Tokens
// getPublicKey godoc
// @Summary Получение токена и публичного ключа по UUID
// @Produce  json
// @Param uuid query string true "UUID пользователя"
// @Success 200 {object} map[string]interface{} "successful response"
// @Router /token/get-public-key [get]
func getPublicKey(c *gin.Context) {
	uuid := c.Query("uuid")
	if uuid == "" {
		c.JSON(400, gin.H{"error": "UUID is required"})
		return
	}

	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
	})

	ctx := context.Background()
	privateKeyStr, err := client.Get(ctx, uuid+"_private_key").Result()
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

	publicKey, err := sshPublicKey(privateKey)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to generate public key"})
		return
	}

	c.JSON(200, gin.H{"public_key": string(ssh.MarshalAuthorizedKey(publicKey))})
}

func sshPublicKey(privateKey *rsa.PrivateKey) (ssh.PublicKey, error) {
	publicRsaKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, err
	}
	return publicRsaKey, nil
}
