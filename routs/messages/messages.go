package messages

import (
	"Bmessage_backend/database"
	"Bmessage_backend/helpers"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gocql/gocql"
)

func MessageRouter(router *gin.Engine) {
	routeBase := "messages/"
	router.POST(routeBase+"add-message", database.WithDatabaseScylla(AddMessage))
	router.GET(routeBase+"get-messages", database.WithDatabaseScylla(GetMessages))
}

// AddMessageStruct represents the JSON
// @Description Данные для создания сообщения
type AddMessageStruct struct {
	ChatID                 string  `json:"chat_id"`
	UserToken              string  `json:"user_token"`
	MessageText            string  `json:"message_text"`
	ReplyToMessageID       *string `json:"reply_to_message_id"`
	ForwardedFromChatID    *string `json:"forwarded_from_chat_id"`
	ForwardedFromMessageID *string `json:"forwarded_from_message_id"`
}

// @Tags Message
// AddMessage godoc
// @Summary Запись сообщения
// @Accept json
// @Produce  json
// @Param data body AddMessageStruct true "Данные для создания сообщения"
// @Success 200 {object} map[string]interface{} "successful response"
// @Failure 400 {object} map[string]interface{} "bad request"
// @Router /messages/add-message [post]
func AddMessage(session *gocql.Session, c *gin.Context) {
	var messageData AddMessageStruct
	if err := c.BindJSON(&messageData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	chatID, err := gocql.ParseUUID(messageData.ChatID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chat_id"})
		return
	}

	userDataToToken, err := helpers.DecryptAES(messageData.UserToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	userID := userDataToToken.User_id
	keyspaceUser := fmt.Sprintf("user_%d", userID)
	queryCompanionDetID := fmt.Sprintf(`SELECT
		companion_id
	FROM
		%s.chats
	WHERE
		chat_id = ? ALLOW FILTERING;
	`, keyspaceUser)

	qSecured := session.Query(queryCompanionDetID, chatID)
	iterSecured := qSecured.Iter()
	defer iterSecured.Close()

	var companionID, keyspaceCompanion string
	for iterSecured.Scan(&companionID) {
		keyspaceCompanion = fmt.Sprintf("user_%s", companionID)
	}

	messageID := gocql.TimeUUID()
	createdAt := time.Now()

	var replyToMessageID *gocql.UUID
	if messageData.ReplyToMessageID != nil {
		replyID, err := gocql.ParseUUID(*messageData.ReplyToMessageID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid reply_to_message_id"})
			return
		}
		replyToMessageID = &replyID
	}

	var forwardedFromChatID *gocql.UUID
	if messageData.ForwardedFromChatID != nil {
		chatID, err := gocql.ParseUUID(*messageData.ForwardedFromChatID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid forwarded_from_chat_id"})
			return
		}
		forwardedFromChatID = &chatID
	}

	var forwardedFromMessageID *gocql.UUID
	if messageData.ForwardedFromMessageID != nil {
		messageID, err := gocql.ParseUUID(*messageData.ForwardedFromMessageID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid forwarded_from_message_id"})
			return
		}
		forwardedFromMessageID = &messageID
	}

	queryUser := fmt.Sprintf(`INSERT INTO %s.messages (chat_id, message_id, sender_id, message_text, created_at, reply_to_message_id, forwarded_from_chat_id, forwarded_from_message_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, keyspaceUser)
	if err := session.Query(queryUser, chatID, messageID, userID, messageData.MessageText, createdAt, replyToMessageID, forwardedFromChatID, forwardedFromMessageID).Exec(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert message"})
		return
	}

	queryCompanion := fmt.Sprintf(`INSERT INTO %s.messages (chat_id, message_id, sender_id, message_text, created_at, reply_to_message_id, forwarded_from_chat_id, forwarded_from_message_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, keyspaceCompanion)
	if err := session.Query(queryCompanion, chatID, messageID, userID, messageData.MessageText, createdAt, replyToMessageID, forwardedFromChatID, forwardedFromMessageID).Exec(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert message"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "Message added successfully"})
}

type Message struct {
	ChatID                 gocql.UUID  `json:"chat_id"`
	MessageID              gocql.UUID  `json:"message_id"`
	SenderID               uint        `json:"sender_id"`
	MessageText            string      `json:"message_text"`
	CreatedAt              time.Time   `json:"created_at"`
	ReplyToMessageID       *gocql.UUID `json:"reply_to_message_id"`
	ForwardedFromChatID    *gocql.UUID `json:"forwarded_from_chat_id"`
	ForwardedFromMessageID *gocql.UUID `json:"forwarded_from_message_id"`
	IsMyMessage            bool        `json:"is_my_message"` // New field
}

// @Tags Message
// GetMessages godoc
// @Summary Получение сообщений
// @Accept json
// @Produce json
// @Param chat_id query string true "Chat ID"
// @Param user_token query string true "User Token"
// @Success 200 {object} []Message "successful response"
// @Failure 400 {object} map[string]interface{} "bad request"
// @Router /messages/get-messages [get]
func GetMessages(session *gocql.Session, c *gin.Context) {
	chatID := c.Query("chat_id")
	userToken := c.Query("user_token")

	if chatID == "" || userToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "chat_id and user_token are required"})
		return
	}

	chatUUID, err := gocql.ParseUUID(chatID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chat_id"})
		return
	}

	userDataToToken, err := helpers.DecryptAES(userToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	userID := userDataToToken.User_id
	keyspaceUser := fmt.Sprintf("user_%d", userID)

	query := fmt.Sprintf(`SELECT chat_id, message_id, sender_id, message_text, created_at, reply_to_message_id, forwarded_from_chat_id, forwarded_from_message_id
	FROM %s.messages
	WHERE chat_id = ?;`, keyspaceUser)

	iter := session.Query(query, chatUUID).Iter()
	defer iter.Close()

	var messages []Message
	var msg Message
	for iter.Scan(
		&msg.ChatID,
		&msg.MessageID,
		&msg.SenderID,
		&msg.MessageText,
		&msg.CreatedAt,
		&msg.ReplyToMessageID,
		&msg.ForwardedFromChatID,
		&msg.ForwardedFromMessageID) {
		msg.IsMyMessage = (msg.SenderID == userID)
		messages = append(messages, msg)
	}

	if err := iter.Close(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch messages"})
		return
	}

	c.JSON(http.StatusOK, messages)
}
