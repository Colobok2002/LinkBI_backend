package chats

import (
	"Bmessage_backend/database"
	"Bmessage_backend/helpers"
	"Bmessage_backend/models"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gocql/gocql"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func RegisterUserTemplate(userID uint) bool {
	session, err := database.GetSession()

	if err != nil {
		return false
	}

	keyspace := fmt.Sprintf("user_%d", userID)

	database.CreateKeyspaceScylla(session, keyspace)

	createTableQuery := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.chats (
		user_id bigint,
		chat_id uuid,
		companion_id bigint,
		chat_type text,
		secured boolean,
		muted boolean,
		new_msg_count int,
		last_msg_time timestamp,
		last_updated timestamp,
		PRIMARY KEY (user_id,last_updated, companion_id, chat_id)
	) WITH CLUSTERING ORDER BY (last_updated DESC);
	`, keyspace)

	if err := session.Query(createTableQuery).Exec(); err != nil {
		log.Println("Failed to create table:", err)
		return false
	}

	createMessageTabelleQuery := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.messages (
		chat_id uuid,
		message_id uuid,
		sender_id bigint,
		message_text TEXT,
		created_at TIMESTAMP,
		attachments LIST<TEXT>,
		reactions MAP<TEXT, INT>,
		reply_to_message_id UUID,
		forwarded_from_chat_id UUID,
		forwarded_from_message_id UUID,
		PRIMARY KEY (chat_id, created_at, message_id)
	) WITH CLUSTERING ORDER BY (created_at DESC);
	`, keyspace)

	if err := session.Query(createMessageTabelleQuery).Exec(); err != nil {
		log.Println("Failed to create table:", err)
		return false
	}

	return true

}

func ChatRouter(router *gin.Engine) {
	routeBase := "chats/"
	router.GET(routeBase+"get-chats", database.WithDatabaseScylla(GetChats))
	// router.GET(routeBase+"get-chats-secured", database.WithDatabaseScylla(GetChatsSecured))
	router.POST(routeBase+"create-chat", database.WithDatabaseScylla(CreateChat))
	router.GET(routeBase+"find-chats", database.WithDatabase(FindChats))
}

type Chat struct {
	ChatID          gocql.UUID  `json:"chat_id"`
	CompanionID     string      `json:"companion_id"`
	CompanionName   string      `json:"companion_name,omitempty"`
	CompanionSoName string      `json:"companion_so_name,omitempty"`
	CompanionNik    string      `json:"companion_nik,omitempty"`
	ChatType        string      `json:"chat_type"`
	Secured         bool        `json:"secured"`
	LastMsgTime     interface{} `json:"last_msg_time"`
	NewMsgCount     int         `json:"new_msg_count"`
	LastMsg         *string     `json:"last_msg,omitempty"`
	LastUpdated     interface{} `json:"last_updated,omitempty"`
	IsMyMessage     bool        `json:"is_my_message"`
}

// GetChats retrieves chats for a user.
// @Tags Chats
// @Summary Получение чатов пользователя
// @Description Получает список чатов для указанного пользователя.
// @Accept json
// @Produce json
// @Param user_token query string true "Токен пользователя"
// @Param uuid query string true "UUID пользователя"
// @Param page_state query string false "Стейт следющей страницы"
// @Success 200 {object} map[string]interface{} "successful response"
// @Failure 400 {object} map[string]interface{} "bad request"
// @Router /chats/get-chats [get]
func GetChats(session *gocql.Session, c *gin.Context) {
	userTokenQuery := c.Query("user_token")

	if userTokenQuery == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing user_id parameter"})
		return
	}

	userDataToToken, err := helpers.DecryptAES(userTokenQuery)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	userID := userDataToToken.User_id

	pageStateQuery := c.Query("page_state")
	var pageState []byte

	if pageStateQuery != "" {
		var err error
		pageState, err = base64.StdEncoding.DecodeString(pageStateQuery)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid page state"})
			return
		}
	}

	keyspaceChats := fmt.Sprintf("user_%d.chats", userID)
	keyspaceMessages := fmt.Sprintf("user_%d.messages", userID)

	query := fmt.Sprintf(`SELECT
		chat_id,
		companion_id,
		chat_type,
		secured,
		last_msg_time,
		last_updated,
		new_msg_count
	FROM
		%s
	WHERE
		user_id = ?
	ORDER BY
		last_updated DESC ALLOW FILTERING;
	`, keyspaceChats)
	q := session.Query(query, userID).PageSize(10).PageState(pageState)

	iter := q.Iter()
	defer iter.Close()

	var chats []Chat
	var chatID gocql.UUID
	var companionID, chatType string
	var secured bool
	var lastMsgTime, lastUpdated *time.Time
	var newMsgCount int

	for iter.Scan(&chatID, &companionID, &chatType, &secured, &lastMsgTime, &lastUpdated, &newMsgCount) {
		var lastMsgTimeValue, lastUpdateTimeValue interface{}
		if lastMsgTime != nil {
			lastMsgTimeValue = *lastMsgTime
		} else {
			lastMsgTimeValue = nil
		}

		if lastUpdated != nil {
			lastUpdateTimeValue = *lastUpdated
		} else {
			lastUpdateTimeValue = nil
		}

		chats = append(chats, Chat{
			ChatID:      chatID,
			CompanionID: companionID,
			ChatType:    chatType,
			Secured:     secured,
			LastMsgTime: lastMsgTimeValue,
			NewMsgCount: newMsgCount,
			LastUpdated: lastUpdateTimeValue,
		})
	}

	if err := iter.Close(); err != nil {
		log.Println(err)
		return
	}
	nextPageState := iter.PageState()

	var companionIDs []string
	for _, chat := range chats {
		companionIDs = append(companionIDs, chat.CompanionID)
	}

	db, err := database.GetDb()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось подключиться к базе данных"})
		return
	}

	var users []models.User
	if err := db.Where("id IN ?", companionIDs).Find(&users).Error; err != nil {
		log.Println("Failed to fetch users from PostgreSQL:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}

	userMap := make(map[string]models.User)
	for _, user := range users {
		strID := fmt.Sprintf("%d", user.ID)
		userMap[strID] = user
	}

	for i := range chats {
		if user, ok := userMap[chats[i].CompanionID]; ok {
			chats[i].CompanionName = user.Name
			chats[i].CompanionSoName = user.SoName
			chats[i].CompanionNik = user.Nik
		}
		queryLastMessage := fmt.Sprintf(`SELECT sender_id, message_text FROM %s WHERE chat_id = ? ORDER BY created_at DESC LIMIT 1`, keyspaceMessages)
		var lastMessageSenderID *uint
		var lastMessageText *string

		if err := session.Query(queryLastMessage, chats[i].ChatID).Scan(&lastMessageSenderID, &lastMessageText); err != nil {
			lastMessageSenderID = nil
			lastMessageText = nil
		}

		chats[i].LastMsg = lastMessageText
		chats[i].IsMyMessage = lastMessageSenderID != nil && *lastMessageSenderID == userID
	}

	c.JSON(http.StatusOK, gin.H{"chats": chats, "nextPageState": nextPageState})

}

// CreateChatStruct represents the JSON structure for a user registration request
// @Description Данные для создания чатов
type CreateChatStruct struct {
	Uuid         string `json:"uuid"`
	User_token   string `json:"user_token"`
	Companion_id uint   `json:"companion_id"`
}

func createChatInKeyspace(session *gocql.Session, chatID gocql.UUID, userID, companionID uint) (gocql.UUID, error) {
	keyspace := fmt.Sprintf("user_%d", userID)
	tableName := keyspace + ".chats"

	var existingChatID gocql.UUID
	var chatType string
	var secured bool
	var last_msg_time *time.Time
	var muted bool
	var newMsgCount int
	var last_updated *time.Time

	checkQuery := fmt.Sprintf(`SELECT
		chat_id,
		chat_type,
		secured,
		muted,
		last_msg_time,
		new_msg_count,
		last_updated
	FROM
		%s.chats
	WHERE
		companion_id = ?
	LIMIT 1
	ALLOW FILTERING;
	`, keyspace)

	if err := session.Query(checkQuery, companionID).Scan(&existingChatID, &chatType, &secured, &muted, &last_msg_time, &newMsgCount, &last_updated); err != nil {
		if err != gocql.ErrNotFound {
			return gocql.UUID{}, err
		}
		chatType = "chat"
		secured = false
		muted = false
		newMsgCount = 0
		last_msg_time = nil
		existingChatID = chatID
	} else {
		deleteQuery := `DELETE FROM ` + tableName + ` WHERE user_id = ? AND last_updated = ? AND companion_id = ? AND chat_id = ?`
		if err := session.Query(deleteQuery, userID, last_updated, companionID, existingChatID).Exec(); err != nil {
			return gocql.UUID{}, err
		}
	}

	insertQuery := `INSERT INTO ` + tableName + ` (chat_id,user_id, companion_id, chat_type, secured, muted,last_msg_time, new_msg_count, last_updated) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	if err := session.Query(insertQuery, existingChatID, userID, companionID, chatType, secured, muted, last_msg_time, newMsgCount, time.Now()).Exec(); err != nil {
		return gocql.UUID{}, err
	}

	return existingChatID, nil
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

	userDataToToken, err := helpers.DecryptAES(chatData.User_token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	newUUID := uuid.New()
	newChatID, _ := gocql.UUIDFromBytes(newUUID[:])

	userID := userDataToToken.User_id

	chatID, err := createChatInKeyspace(session, newChatID, userID, chatData.Companion_id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create chat: %v", err)})
		return
	}

	_, err = createChatInKeyspace(session, newChatID, chatData.Companion_id, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create chat for companion: %v", err)})
		return
	}

	UpdeteDataChat(userID, chatID)
	// UpdeteDataChat(chatData.Companion_id, chatID)

	c.JSON(http.StatusOK, gin.H{"message": "Chat setup successfully for both users", "chat_id": chatID})
}

// FindChats retrieves chats for a user.
// @Tags Chats
// @Summary Поиск пользователей
// @Description Получает список пользователей
// @Accept json
// @Produce json
// @Param search_term query string true "search_term"
// @Param uuid query string true "UUID пользователя"
// @Success 200 {object} map[string]interface{} "successful response"
// @Failure 400 {object} map[string]interface{} "bad request"
// @Router /chats/find-chats [get]
func FindChats(db *gorm.DB, c *gin.Context) {
	searchTerm := c.Query("search_term")
	searchTerm = "%" + searchTerm + "%"
	var users []models.User

	if err := db.Model(&models.User{}).Select("ID", "Name", "SoName", "Nik").Where(
		"LOWER(name) LIKE LOWER(?) OR LOWER(so_name) LIKE LOWER(?) OR LOWER(nik) LIKE LOWER(?)",
		searchTerm, searchTerm, searchTerm,
	).Find(&users).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to find users", "details": err.Error()})
		return
	}

	var results []map[string]interface{}
	for _, user := range users {
		result := map[string]interface{}{
			"user_id": user.ID,
			"name":    user.Name,
			"soName":  user.SoName,
			"nik":     user.Nik,
		}
		results = append(results, result)
	}

	c.JSON(http.StatusOK, gin.H{"data": results})
}

func GetChatDetails(userID uint, chatID gocql.UUID) (Chat, error) {
	var chat Chat

	session, err := database.GetSession()
	if err != nil {
		return chat, err
	}
	defer session.Close()

	keyspaceChats := fmt.Sprintf("user_%d.chats", userID)
	keyspaceMessages := fmt.Sprintf("user_%d.messages", userID)

	query := fmt.Sprintf(`SELECT
		chat_id,
		companion_id,
		chat_type,
		secured,
		last_msg_time,
		last_updated,
		new_msg_count
	FROM
		%s
	WHERE
		chat_id = ?
	ALLOW FILTERING;
	`, keyspaceChats)

	var lastMsgTime *time.Time
	var lastUpdateTime *time.Time

	if err := session.Query(query, chatID).Scan(
		&chat.ChatID, &chat.CompanionID, &chat.ChatType, &chat.Secured, &lastMsgTime, &lastUpdateTime, &chat.NewMsgCount,
	); err != nil {
		return chat, fmt.Errorf("failed to fetch chat details: %v", err)
	}

	if lastMsgTime != nil {
		chat.LastMsgTime = *lastMsgTime
	} else {
		chat.LastMsgTime = nil
	}

	if lastUpdateTime != nil {
		chat.LastUpdated = *lastUpdateTime
	} else {
		chat.LastUpdated = nil
	}

	db, err := database.GetDb()
	if err != nil {
		return chat, fmt.Errorf("failed to connect to the database: %v", err)
	}

	var users []models.User
	if err := db.Where("id IN ?", []string{fmt.Sprintf("%d", userID), chat.CompanionID}).Find(&users).Error; err != nil {
		log.Println("Failed to fetch users from PostgreSQL:", err)
		return chat, fmt.Errorf("failed to fetch users: %v", err)
	}

	for _, user := range users {
		if fmt.Sprintf("%d", user.ID) == fmt.Sprintf("%d", userID) {
		} else if fmt.Sprintf("%d", user.ID) == chat.CompanionID {
			chat.CompanionName = user.Name
			chat.CompanionSoName = user.SoName
			chat.CompanionNik = user.Nik
		}
	}

	queryLastMessage := fmt.Sprintf(`SELECT sender_id, message_text FROM %s WHERE chat_id = ? ORDER BY created_at DESC LIMIT 1`, keyspaceMessages)
	var lastMessageSenderID *uint
	var lastMessageText *string

	if err := session.Query(queryLastMessage, chat.ChatID).Scan(&lastMessageSenderID, &lastMessageText); err != nil {
		lastMessageSenderID = nil
		lastMessageText = nil
	}

	chat.LastMsg = lastMessageText
	chat.IsMyMessage = lastMessageSenderID != nil && *lastMessageSenderID == userID

	return chat, nil
}
