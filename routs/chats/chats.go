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

	// createViewQuery := fmt.Sprintf(`
	// CREATE MATERIALIZED VIEW IF NOT EXISTS %s.chats_view AS
	// SELECT *
	// FROM %s.chats
	// WHERE companion_id IS NOT NULL AND chat_id IS NOT NULL AND last_msg_time IS NOT NULL
	// PRIMARY KEY (companion_id, chat_id, last_msg_time)
	// WITH CLUSTERING ORDER BY (last_msg_time DESC)
	// `, keyspace, keyspace)

	// if err := session.Query(createViewQuery).Exec(); err != nil {
	// 	log.Println("Failed to create materialized view:", err)
	// 	return false
	// }

	return true

}

func ChatRouter(router *gin.Engine) {
	routeBase := "chats/"
	router.GET(routeBase+"get-chats", database.WithDatabaseScylla(GetChats))
	router.GET(routeBase+"get-chats-secured", database.WithDatabaseScylla(GetChatsSecured))
	router.POST(routeBase+"create-chat", database.WithDatabaseScylla(CreateChat))
	router.GET(routeBase+"find-chats", database.WithDatabase(FindChats))
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

	keyspace := fmt.Sprintf("user_%d.chats", userID)

	// RegisterUserTemplate(uint(21))
	// RegisterUserTemplate(uint(22))
	// RegisterUserTemplate(uint(23))

	query := fmt.Sprintf(` SELECT
		chat_id,
		companion_id,
		chat_type,
		secured,
		last_updated,
		new_msg_count
	FROM
		%s
	WHERE
		secured = ?
		AND user_id = ?
	ORDER BY
		last_updated DESC ALLOW FILTERING;
	`, keyspace)
	q := session.Query(query, false, userID).PageSize(10).PageState(pageState)

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
			"lastMsg":       false,
			"new_msg_count": newMsgCount,
		}
		chats = append(chats, chat)
	}

	if err := iter.Close(); err != nil {
		log.Println(err)
		return
	}
	nextPageState := iter.PageState()

	var companionIDs []string

	for _, chat := range chats {
		companionIDs = append(companionIDs, chat["companion_id"].(string))
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

	for i, chat := range chats {
		if user, ok := userMap[chat["companion_id"].(string)]; ok {
			chats[i]["companion_name"] = user.Name
			chats[i]["companion_so_name"] = user.SoName
			chats[i]["companion_nik"] = user.Nik
		}
	}

	c.JSON(http.StatusOK, gin.H{"chats": chats, "nextPageState": nextPageState})

}

// GetChatsSecured retrieves chats for a user.
// @Tags Chats
// @Summary Получение чатов пользователя
// @Description Получает список чатов для указанного пользователя.
// @Accept json
// @Produce json
// @Param user_token query string true "Token пользователя"
// @Param uuid query string true "UUID пользователя"
// @Success 200 {object} map[string]interface{} "successful response"
// @Failure 400 {object} map[string]interface{} "bad request"
// @Router /chats/get-chats-secured [get]
func GetChatsSecured(session *gocql.Session, c *gin.Context) {
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

	keyspace := fmt.Sprintf("user_%d.chats", userID)

	querySecured := fmt.Sprintf(` SELECT
		chat_id,
		companion_id,
		chat_type,
		secured,
		last_updated,
		new_msg_count
	FROM
		%s
	WHERE
		secured = ?
		AND user_id = ?
	ORDER BY
		last_updated DESC ALLOW FILTERING;
	`, keyspace)

	qSecured := session.Query(querySecured, true, userID)
	iterSecured := qSecured.Iter()
	defer iterSecured.Close()

	var chatsSecured []map[string]interface{}
	var chatID gocql.UUID
	var companionID, chatType string
	var secured bool
	var lastMsgTime time.Time
	var newMsgCount int

	for iterSecured.Scan(&chatID, &companionID, &chatType, &secured, &lastMsgTime, &newMsgCount) {
		chat := map[string]interface{}{
			"chat_id":       chatID,
			"companion_id":  companionID,
			"chat_type":     chatType,
			"secured":       secured,
			"last_msg_time": lastMsgTime,
			"new_msg_count": newMsgCount,
		}
		chatsSecured = append(chatsSecured, chat)
	}

	var companionIDs []string

	for _, chat := range chatsSecured {
		companionIDs = append(companionIDs, chat["companion_id"].(string))
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

	for i, chat := range chatsSecured {
		if user, ok := userMap[chat["companion_id"].(string)]; ok {
			chatsSecured[i]["companion_name"] = user.Name
			chatsSecured[i]["companion_so_name"] = user.SoName
			chatsSecured[i]["companion_nik"] = user.Nik
		}
	}

	c.JSON(http.StatusOK, gin.H{"chatsSecured": chatsSecured})

}

// CreateChatStruct represents the JSON structure for a user registration request
// @Description Данные для создания чатов
type CreateChatStruct struct {
	Uuid         string `json:"uuid"`
	User_token   string `json:"user_token"`
	Companion_id uint   `json:"companion_id"`
}

func createChatInKeyspace(session *gocql.Session, chatID string, userID, companionID uint) (string, error) {
	keyspace := fmt.Sprintf("user_%d", userID)
	tableName := keyspace + ".chats"

	var existingChatID string
	var chatType string
	var secured bool
	var last_msg_time *time.Time
	var muted bool
	var newMsgCount int
	var last_updated *time.Time

	checkQuery := `SELECT
		chat_id,
		chat_type,
		secured,
		muted,
		last_msg_time,
		new_msg_count,
		last_updated
	FROM
		user_21.chats
	WHERE
		companion_id = ?
	LIMIT 1
	ALLOW FILTERING;
	`

	if err := session.Query(checkQuery, companionID).Scan(&existingChatID, &chatType, &secured, &muted, &last_msg_time, &newMsgCount, &last_updated); err != nil {
		if err != gocql.ErrNotFound {
			return "", err
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
			return "", err
		}
	}

	insertQuery := `INSERT INTO ` + tableName + ` (chat_id,user_id, companion_id, chat_type, secured, muted,last_msg_time, new_msg_count, last_updated) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	if err := session.Query(insertQuery, existingChatID, userID, companionID, chatType, secured, muted, last_msg_time, newMsgCount, time.Now()).Exec(); err != nil {
		return "", err
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
	newChatID := uuid.New().String()
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

	log.Println(searchTerm)

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
