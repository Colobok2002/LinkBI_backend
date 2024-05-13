package chats

import (
	"Bmessage_backend/database"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gocql/gocql"
)

func registerUserTemplate(userID string) bool {

	session, err := database.GetSession()

	if err != nil {
		return false
	}

	keyspace := fmt.Sprintf("keyspace_%s", userID)

	database.CreateKeyspaceScylla(session, keyspace)

	createTableQuery := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.chat (
		chat_id uuid PRIMARY KEY,
		companion_id text,
		chat_type text,
	)`, keyspace)

	if err := session.Query(createTableQuery).Exec(); err != nil {
		return false
	}

	return true
}

func ChatRouter(router *gin.Engine) {
	routeBase := "chats/"
	router.GET(routeBase+"get-chats", database.WithDatabaseScylla(GetChats))
	router.POST(routeBase+"create-chat", database.WithDatabaseScylla(CreateChat))
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
func GetChats(session *gocql.Session, c *gin.Context) {
	userID := c.Query("user_id")

	registerUserTemplate(userID)

	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing user_id parameter"})
		return
	}

	tableName := fmt.Sprintf("%s.%s", userID, "chats")

	// if exists, err := checkTableExists(session, userID, "chats"); err != nil {
	// 	c.JSON(http.StatusOK, gin.H{"chats": []string{}})
	// 	return
	// } else if !exists {
	// 	c.JSON(http.StatusOK, gin.H{"chats": []string{}})
	// 	return
	// }

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
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error fetching chats: %v", err)})
		return
	}

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
func CreateChat(session *gocql.Session, c *gin.Context) {
	// var chatData CreateChatStruct
	// if err := c.BindJSON(&chatData); err != nil {
	// 	c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
	// 	return
	// }

	// userID := chatData.User_id
	// companionID := chatData.Companion_id

	// keyspace := "chat"
	// if err := database.CreateKeyspaceScylla(session, keyspace); err != nil {
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to ensure keyspace %s: %v", keyspace, err)})
	// 	return
	// }

	// userTables := []string{userID, companionID}
	// for _, ut := range userTables {
	// 	tableName := fmt.Sprintf("%s.%s", keyspace, ut)
	// 	if exists, _ := checkTableExists(session, keyspace, ut); !exists {
	// 		if err := createChatTable(session, tableName); err != nil {
	// 			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create table %s: %v", tableName, err)})
	// 			return
	// 		}
	// 	}
	// }

	// if err := addChatEntry(session, fmt.Sprintf("%s.%s", keyspace, userID), companionID); err != nil {
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to add entry to table %s: %v", userID, err)})
	// 	return
	// }
	// if err := addChatEntry(session, fmt.Sprintf("%s.%s", keyspace, companionID), userID); err != nil {
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to add entry to table %s: %v", companionID, err)})
	// 	return
	// }

	// c.JSON(http.StatusOK, gin.H{"message": "Chat setup successfully for both users"})
}
