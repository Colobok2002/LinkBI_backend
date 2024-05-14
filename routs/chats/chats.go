package chats

import (
	"Bmessage_backend/database"
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
		PRIMARY KEY (companion_id, chat_id)
	) WITH CLUSTERING ORDER BY (chat_id ASC)`, keyspace)

	if err := session.Query(createTableQuery).Exec(); err != nil {
		log.Println("Failed to create table:", err)
		return false
	}

	createViewQuery := fmt.Sprintf(`CREATE MATERIALIZED VIEW IF NOT EXISTS %s.chats_view AS
	SELECT *
	FROM %s.chats
	WHERE chat_id IS NOT NULL AND last_msg_time IS NOT NULL
	PRIMARY KEY ((companion_id), chat_id, last_msg_time)
	WITH CLUSTERING ORDER BY (last_msg_time DESC)
	`, keyspace, keyspace)

	if err := session.Query(createViewQuery).Exec(); err != nil {
		log.Println("Failed to create materialized view:", err)
		return false
	}

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
	var pageState []byte

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

	viewName := fmt.Sprintf("%s.chats_view", keyspace)

	query := fmt.Sprintf("SELECT chat_id,companion_id,chat_type,secured,last_msg_time,new_msg_count FROM %s WHERE secured = ? ALLOW FILTERING", viewName)
	q := session.Query(query, false).PageSize(10).PageState(pageState)

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
// @Param user_id query string true "user_id"
// @Param uuid query string true "UUID пользователя"
// @Success 200 {object} map[string]interface{} "successful response"
// @Failure 400 {object} map[string]interface{} "bad request"
// @Router /chats/get-chats-secured [get]
func GetChatsSecured(session *gocql.Session, c *gin.Context) {
	userID := c.Query("user_id")

	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing user_id parameter"})
		return
	}

	keyspace := fmt.Sprintf("keyspace_%s", userID)

	registerUserTemplate(userID)

	viewName := fmt.Sprintf("%s.chats_view", keyspace)

	querySecured := fmt.Sprintf("SELECT chat_id, companion_id, chat_type, secured, last_msg_time, new_msg_count FROM %s WHERE secured = ? ALLOW FILTERING", viewName)
	qSecured := session.Query(querySecured, true)
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
			"Name":    user.Name,
			"SoName":  user.SoName,
			"Nik":     user.Nik,
		}
		results = append(results, result)
	}

	c.JSON(http.StatusOK, gin.H{"users": results})
}
