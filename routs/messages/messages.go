package messages

import (
	"Bmessage_backend/database"
	"Bmessage_backend/helpers"
	"Bmessage_backend/routs/chats"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gocql/gocql"
)

func MessageRouter(router *gin.Engine) {
	routeBase := "messages/"
	router.POST(routeBase+"add-message", database.WithDatabaseScylla(AddMessage))
	router.GET(routeBase+"get-messages", database.WithDatabaseScylla(GetMessages))
	router.POST(routeBase+"read-message", database.WithDatabaseScylla(ReadMessage))
}

// AddMessageStruct represents the JSON
// @Description Данные для создания сообщения
type AddMessageStruct struct {
	TemporaryMessageId     string  `json:"temporary_message_id"`
	ChatID                 string  `json:"chat_id"`
	UserToken              string  `json:"user_token"`
	MessageText            string  `json:"message_text"`
	ReplyToMessageID       *string `json:"reply_to_message_id"`
	ForwardedFromChatID    *string `json:"forwarded_from_chat_id"`
	ForwardedFromMessageID *string `json:"forwarded_from_message_id"`
}

type Message struct {
	TemporaryMessageId     string      `json:"temporary_message_id"`
	ChatID                 gocql.UUID  `json:"chat_id"`
	MessageID              gocql.UUID  `json:"message_id"`
	SenderID               uint        `json:"sender_id"`
	MessageText            string      `json:"message_text"`
	CreatedAt              time.Time   `json:"created_at"`
	ReplyToMessageID       *gocql.UUID `json:"reply_to_message_id"`
	ForwardedFromChatID    *gocql.UUID `json:"forwarded_from_chat_id"`
	ForwardedFromMessageID *gocql.UUID `json:"forwarded_from_message_id"`
	IsMyMessage            bool        `json:"is_my_message"`
	Read                   bool        `json:"read"`
	Type                   string      `json:"type"`
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
	queryCompanionDetID := fmt.Sprintf(`SELECT companion_id, chat_type, secured, muted, last_updated ,private_key FROM %s.chats WHERE chat_id = ? ALLOW FILTERING;`, keyspaceUser)

	var companionID uint
	var chatType string
	var secured bool
	var muted bool
	var newMsgCount int
	var lastUpdated *time.Time
	var private_keyUser string
	var private_keyCompanion string

	if err := session.Query(queryCompanionDetID, chatID).Scan(&companionID, &chatType, &secured, &muted, &lastUpdated, &private_keyUser); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get chat details"})
		return
	}

	keyspaceCompanion := fmt.Sprintf("user_%d", companionID)

	queryCompanionNewMsgCount := fmt.Sprintf(`SELECT new_msg_count,private_key FROM %s.chats WHERE user_id = ? AND companion_id = ? AND chat_id = ? AND last_updated = ? ALLOW FILTERING;`, keyspaceCompanion)
	if err := session.Query(queryCompanionNewMsgCount, companionID, userID, chatID, lastUpdated).Scan(&newMsgCount, &private_keyCompanion); err != nil {
		if err == gocql.ErrNotFound {
			newMsgCount = 0
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get new message count for companion"})
			return
		}
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

	publicKeyUser, err := helpers.ExtractPublicKey(private_keyUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert message"})
		return
	}

	publicKeyCompanion, err := helpers.ExtractPublicKey(private_keyCompanion)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert message"})
		return
	}

	encryptedDataUser, err := helpers.EncryptWithPublicKey(messageData.MessageText, publicKeyUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert message"})
		return
	}

	encryptedDataCompanion, err := helpers.EncryptWithPublicKey(messageData.MessageText, publicKeyCompanion)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert message"})
		return
	}

	queryUserAddMessage := fmt.Sprintf(`INSERT INTO %s.messages (chat_id, message_id, sender_id, message_text, created_at, reply_to_message_id, forwarded_from_chat_id, forwarded_from_message_id,read) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, keyspaceUser)
	if err := session.Query(queryUserAddMessage, chatID, messageID, userID, encryptedDataUser, createdAt, replyToMessageID, forwardedFromChatID, forwardedFromMessageID, false).Exec(); err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert message"})
		return
	}

	queryCompanionAddMessage := fmt.Sprintf(`INSERT INTO %s.messages (chat_id, message_id, sender_id, message_text, created_at, reply_to_message_id, forwarded_from_chat_id, forwarded_from_message_id,read) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, keyspaceCompanion)
	if err := session.Query(queryCompanionAddMessage, chatID, messageID, userID, encryptedDataCompanion, createdAt, replyToMessageID, forwardedFromChatID, forwardedFromMessageID, false).Exec(); err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert message"})
		return
	}

	deleteChatUser := fmt.Sprintf(`DELETE FROM %s.chats WHERE user_id = ? AND companion_id = ? AND chat_id = ? AND last_updated = ?`, keyspaceUser)
	if err := session.Query(deleteChatUser, userID, companionID, chatID, lastUpdated).Exec(); err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete existing chat for user"})
		return
	}

	insertChatUser := fmt.Sprintf(`INSERT INTO %s.chats (user_id, chat_id, companion_id, chat_type, secured, muted, new_msg_count, last_msg_time, last_updated,private_key) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, keyspaceUser)
	if err := session.Query(insertChatUser, userID, chatID, companionID, chatType, secured, muted, 0, createdAt, createdAt, private_keyUser).Exec(); err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert new chat for user"})
		return
	}

	deleteChatCompanion := fmt.Sprintf(`DELETE FROM %s.chats WHERE user_id = ? AND companion_id = ? AND chat_id = ? AND last_updated = ?`, keyspaceCompanion)
	if err := session.Query(deleteChatCompanion, companionID, userID, chatID, lastUpdated).Exec(); err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete existing chat for companion"})
		return
	}

	insertChatCompanion := fmt.Sprintf(`INSERT INTO %s.chats (user_id, chat_id, companion_id, chat_type, secured, muted, new_msg_count, last_msg_time, last_updated,private_key) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, keyspaceCompanion)
	if err := session.Query(insertChatCompanion, companionID, chatID, userID, chatType, secured, muted, newMsgCount+1, createdAt, createdAt, private_keyCompanion).Exec(); err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert new chat for companion"})
		return
	}

	newMessage := Message{
		TemporaryMessageId:     messageData.TemporaryMessageId,
		ChatID:                 chatID,
		SenderID:               userID,
		MessageID:              messageID,
		MessageText:            messageData.MessageText,
		CreatedAt:              createdAt,
		ReplyToMessageID:       replyToMessageID,
		ForwardedFromChatID:    forwardedFromChatID,
		ForwardedFromMessageID: forwardedFromMessageID,
		IsMyMessage:            true,
		Read:                   false,
		Type:                   "new",
	}

	chats.UpdeteDataChat(userID, chatID)
	chats.UpdeteDataChat(companionID, chatID)
	SendWsMessageToChat(chatID.String(), newMessage)
	c.JSON(http.StatusOK, gin.H{"status": "Message added successfully"})
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

	var privateKey string

	checkQuery := fmt.Sprintf(`SELECT
		private_key
	FROM
		%s.chats
	WHERE
	chat_id = ?
	LIMIT 1
	ALLOW FILTERING;
	`, keyspaceUser)

	if err := session.Query(checkQuery, chatID).Scan(&privateKey); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch messages"})
		return
	}

	query := fmt.Sprintf(`SELECT chat_id, message_id, sender_id, message_text, created_at, reply_to_message_id, forwarded_from_chat_id, forwarded_from_message_id , read
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
		&msg.ForwardedFromMessageID,
		&msg.Read) {

		decryptedText, err := helpers.DecryptWithPrivateKey(msg.MessageText, privateKey)
		if err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decrypt message"})
			return
		}
		msg.MessageText = decryptedText
		msg.IsMyMessage = (msg.SenderID == userID)
		messages = append(messages, msg)
	}

	if err := iter.Close(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch messages"})
		return
	}

	c.JSON(http.StatusOK, messages)
}

// ReadMessageStruct represents the JSON
// @Description Данные для прочтения сообщения
type ReadMessageStruct struct {
	ChatID    string    `json:"chat_id"`
	MessageID string    `json:"message_id"`
	UserToken string    `json:"user_token"`
	CreatedAt time.Time `json:"created_at"`
}

// @Tags Message
// ReadMessage godoc
// @Tags Message
// @Summary Сообщение было прочитано
// @Accept json
// @Produce  json
// @Param data body ReadMessageStruct true "Данные для прочтения сообщения"
// @Success 200 {object} map[string]interface{} "successful response"
// @Failure 400 {object} map[string]interface{} "bad request"
// @Router /messages/read-message [post]
func ReadMessage(session *gocql.Session, c *gin.Context) {
	var messageData ReadMessageStruct
	if err := c.BindJSON(&messageData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	chatID, err := gocql.ParseUUID(messageData.ChatID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chat_id"})
		return
	}

	messageID, err := gocql.ParseUUID(messageData.MessageID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid message_id"})
		return
	}

	userDataToToken, err := helpers.DecryptAES(messageData.UserToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	userID := userDataToToken.User_id
	keyspaceUser := fmt.Sprintf("user_%d", userID)
	queryCompanionDetID := fmt.Sprintf(`SELECT companion_id,new_msg_count,last_updated FROM %s.chats WHERE chat_id = ? ALLOW FILTERING;`, keyspaceUser)

	var companionID uint
	var newMsgCount int
	var last_updated time.Time

	if err := session.Query(queryCompanionDetID, chatID).Scan(&companionID, &newMsgCount, &last_updated); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get chat details"})
		return
	}

	log.Println(newMsgCount)
	keyspaceCompanion := fmt.Sprintf("user_%d", companionID)

	queryUserReadMessage := fmt.Sprintf(`UPDATE %s.messages SET read = true WHERE chat_id = ? AND created_at = ? AND message_id = ?`, keyspaceUser)
	if err := session.Query(queryUserReadMessage, chatID, messageData.CreatedAt, messageID).Exec(); err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update read status for user"})
		return
	}

	queryCompanionReadMessage := fmt.Sprintf(`UPDATE %s.messages SET read = true WHERE chat_id = ? AND created_at = ? AND message_id = ?`, keyspaceCompanion)
	if err := session.Query(queryCompanionReadMessage, chatID, messageData.CreatedAt, messageID).Exec(); err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update read status for companion"})
		return
	}

	if newMsgCount > 0 {
		newMsgCount--
		updateUserChatQuery := fmt.Sprintf(`UPDATE %s.chats SET new_msg_count = ? WHERE user_id = ? and last_updated = ? and companion_id = ? and chat_id = ?`, keyspaceUser)
		if err := session.Query(updateUserChatQuery, newMsgCount, userDataToToken.User_id, last_updated, companionID, chatID).Exec(); err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update new message count for user"})
			return
		}

	}

	newMessage := Message{
		MessageID: messageID,
		Read:      true,
		Type:      "read",
	}

	chats.UpdeteDataChat(userID, chatID)
	chats.UpdeteDataChat(companionID, chatID)

	SendWsMessageToChat(chatID.String(), newMessage)

	c.JSON(http.StatusOK, gin.H{"status": "Message read successfully"})
}
