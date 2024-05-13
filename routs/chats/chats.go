package chats

import (
	"Bmessage_backend/database"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gocql/gocql"
	"github.com/google/uuid"
)

func registerUserTemplate(userID string) bool {

	session, err := database.GetSession()

	if err != nil {
		return false
	}

	keyspace := fmt.Sprintf("keyspace_%s", userID)

	database.CreateKeyspaceScylla(session, keyspace)

	createTableQuery := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.chats (
		chat_id uuid,
		companion_id text,
		chat_type text,
		secured boolean,
		muted boolean,
		last_msg_time timestamp,
		new_msg_count int,
		PRIMARY KEY ((secured), last_msg_time, chat_id)
	) WITH CLUSTERING ORDER BY (last_msg_time DESC)`, keyspace)

	if err := session.Query(createTableQuery).Exec(); err != nil {
		log.Println(err)
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
// @Param page_state query string false "Стейт следющей страницы"
// @Success 200 {object} map[string]interface{} "successful response"
// @Failure 400 {object} map[string]interface{} "bad request"
// @Router /chats/get-chats [get]
func GetChats(session *gocql.Session, c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing user_id parameter"})
		return
	}
	pageStateQuery := c.Query("page_state")
	var pageState []byte // Теперь это слайс байт, а не указатель на строку

	if pageStateQuery != "" {
		var err error
		pageState, err = base64.StdEncoding.DecodeString(pageStateQuery)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid page state"})
			return
		}
	}

	keyspace := fmt.Sprintf("keyspace_%s", userID)

	registerUserTemplate(userID)

	tableName := fmt.Sprintf("%s.%s", keyspace, "chats")

	query := fmt.Sprintf("SELECT chat_id,companion_id,chat_type,secured,last_msg_time,new_msg_count FROM %s", tableName)
	q := session.Query(query).PageSize(10).PageState(pageState)

	iter := q.Iter()
	defer iter.Close()

	var chats []map[string]interface{}
	var chatID gocql.UUID
	var companionID, chatType string
	var secured bool
	var lastMsgTime time.Time
	var newMsgCount int

	for iter.Scan(&chatID, &companionID, &chatType, &secured, &lastMsgTime, &newMsgCount) {
		chat := map[string]interface{}{
			"chat_id":       chatID,
			"companion_id":  companionID,
			"chat_type":     chatType,
			"secured":       secured,
			"last_msg_time": lastMsgTime,
			"new_msg_count": newMsgCount,
		}
		chats = append(chats, chat)
	}

	if err := iter.Close(); err != nil {

		log.Println(err)
		return
	}

	nextPageState := iter.PageState()
	c.JSON(http.StatusOK, gin.H{"data": chats, "nextPageState": nextPageState})

}

// CreateChatStruct represents the JSON structure for a user registration request
// @Description Данные для создания чатов
type CreateChatStruct struct {
	Uuid         string `json:"uuid"`
	User_id      string `json:"user_id"`
	Companion_id string `json:"companion_id"`
}

func createChatInKeyspace(session *gocql.Session, chatID string, userID, companionID string) error {
	keyspace := fmt.Sprintf("keyspace_%s", userID)
	tableName := keyspace + ".chats"

	insertQuery := `INSERT INTO ` + tableName + ` (chat_id, companion_id, chat_type, secured, muted, last_msg_time, new_msg_count) VALUES (?, ?, ?, ?, ?, ?, ?)`
	return session.Query(insertQuery, chatID, companionID, "normal", false, false, time.Now(), 0).Exec()
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
	var chatData CreateChatStruct
	if err := c.BindJSON(&chatData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	chatID := uuid.New().String()

	if err := createChatInKeyspace(session, chatID, chatData.User_id, chatData.Companion_id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create chat: %v", err)})
		return
	}

	if err := createChatInKeyspace(session, chatID, chatData.Companion_id, chatData.User_id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create chat for companion: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Chat setup successfully for both users"})
}
