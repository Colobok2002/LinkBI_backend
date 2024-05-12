package chats

import (
	"Bmessage_backend/database"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gocql/gocql"
)

func ChatRouter(router *gin.Engine) {
	routeBase := "chats/"
	router.GET(routeBase+"get-chats", database.WithDatabaseScylla(GetChatsRouter))
	router.POST(routeBase+"create-chat", database.WithDatabaseScylla(GetChatsRouter))
}

// GetChatsRouter retrieves chats for a user.
// @Tags Chats
// @Summary Получение чатов пользователя
// @Description Получает список чатов для указанного пользователя.
// @Accept json
// @Produce json
// @Param user_id query string true "user_id"
// @Param uuid query string true "UUID пользователя"
// @Success 200 {object} map[string]interface{} "successful response"
// @Failure 400 {object} map[string]interface{} "bad request"
// @Router /chats/get-chats [get]
func GetChatsRouter(session *gocql.Session, c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing user_id parameter"})
		return
	}

	keyspace := "chat"
	tableName := fmt.Sprintf("%s.%s", keyspace, userID)

	query := fmt.Sprintf("SELECT chat_id, companion_id, type FROM %s", tableName)
	var chats []map[string]interface{}
	iter := session.Query(query).Iter()

	var chatID, companionAccountID, chatType string
	for iter.Scan(&chatID, companionAccountID, &chatType) {
		chats = append(chats, map[string]interface{}{
			"chat_id":      chatID,
			"companion_id": companionAccountID,
			"type":         chatType,
		})
	}
	if err := iter.Close(); err != nil {
		log.Printf("Error closing iterator: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching chats"})
		return
	}

	// Decide on response based on the content of the chats slice
	if len(chats) == 0 {
		c.JSON(http.StatusOK, gin.H{"chats": []string{}})
	} else {
		c.JSON(http.StatusOK, gin.H{"chats": chats})
	}
}

// CreateChatStruct represents the JSON structure for a user registration request
// @Description Данные для создания чатов
type CreateChatStruct struct {
	Uuid         string `json:"uuid"`
	User_id      string `json:"user_id"`
	Companion_id string `json:"companion_id"`
}

// @Tags Chats
// CreateChat godoc
// @Summary Создание чата
// @Accept json
// @Produce  json
// @Param data body CreateChatStruct true "Данные для создания чата"
// @Success 200 {object} map[string]interface{} "successful response"
// @Failure 400 {object} map[string]interface{} "bad request"
// @Router /chats/create-chat [post]
func CreateChat(c *gin.Context, session *gocql.Session) {
	var chatData CreateChatStruct
	if err := c.BindJSON(&chatData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	userID := chatData.User_id
	companionID := chatData.Companion_id

	keyspace := "chat"
	if err := database.CreateKeyspaceScylla(session, keyspace); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to ensure keyspace %s: %v", keyspace, err)})
		return
	}

	userTables := []string{userID, companionID}
	for _, ut := range userTables {
		tableName := fmt.Sprintf("%s.%s", keyspace, ut)
		if exists, _ := checkTableExists(session, keyspace, ut); !exists {
			if err := createChatTable(session, tableName); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create table %s: %v", tableName, err)})
				return
			}
		}
	}

	if err := addChatEntry(session, fmt.Sprintf("%s.%s", keyspace, userID), companionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to add entry to table %s: %v", userID, err)})
		return
	}
	if err := addChatEntry(session, fmt.Sprintf("%s.%s", keyspace, companionID), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to add entry to table %s: %v", companionID, err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Chat setup successfully for both users"})
}

func checkTableExists(session *gocql.Session, keyspace, tableName string) (bool, error) {
	query := fmt.Sprintf("SELECT COUNT(*) FROM system_schema.tables WHERE keyspace_name='%s' AND table_name='%s';", keyspace, tableName)
	var count int
	if err := session.Query(query).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func createChatTable(session *gocql.Session, tableName string) error {
	createTableQuery := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		chat_id uuid PRIMARY KEY,
		companion_id text,
		type text DEFAULT 'chat'
	)`, tableName)
	return session.Query(createTableQuery).Exec()
}

func addChatEntry(session *gocql.Session, tableName, companionID string) error {
	insertQuery := fmt.Sprintf("INSERT INTO %s (chat_id, companion_id, type) VALUES (uuid(), ?, 'chat');", tableName)
	return session.Query(insertQuery, companionID).Exec()
}
