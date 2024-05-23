package chats

import (
	"Bmessage_backend/database"
	"log"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gocql/gocql"
	"github.com/gorilla/websocket"
)

type ConnectionData struct {
	WS        *websocket.Conn
	UserToken string
}

var (
	connections   = make(map[string][]ConnectionData)
	connectionsMu sync.Mutex
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func ChatsRouterWs(router *gin.Engine) {
	routeBase := "messages/"
	router.GET(routeBase+"events-messages", database.WithDatabaseScylla(cahtsConnectorWs))
}

func cahtsConnectorWs(session *gocql.Session, c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("Failed to set websocket upgrade:", err)
		return
	}
	defer conn.Close()

	chatID := c.Query("chatId")
	userToken := c.Query("userToken")

	wsData := ConnectionData{
		WS:        conn,
		UserToken: userToken,
	}

	addConnection(chatID, wsData)
	log.Println(connections)
	defer removeConnection(chatID, conn)

	for {
		var msg interface{}
		err := conn.ReadJSON(&msg)
		if err != nil {
			break
		}

		if err = conn.WriteJSON(msg); err != nil {
			break
		}
	}
}

func addConnection(chatID string, data ConnectionData) {
	connectionsMu.Lock()
	defer connectionsMu.Unlock()
	connections[chatID] = append(connections[chatID], data)
}

func removeConnection(chatID string, conn *websocket.Conn) {
	connectionsMu.Lock()
	defer connectionsMu.Unlock()
	conns := connections[chatID]
	for i, c := range conns {
		if c.WS == conn {
			connections[chatID] = append(conns[:i], conns[i+1:]...)
			break
		}
	}
}

// func UpdeteDataChat(chatID string, chatData Message) {
// 	connectionsMu.Lock()
// 	defer connectionsMu.Unlock()

// 	conns, ok := connections[chatID]
// 	if !ok {
// 		log.Printf("No connections found for chatId: %s", chatID)
// 		return
// 	}

// 	for _, conn := range conns {
// 		userToken := conn.UserToken
// 		userDataToToken, err := helpers.DecryptAES(userToken)
// 		if err != nil {
// 			log.Println("Error writing json to connection:", err)
// 			return
// 		}

// 		userID := userDataToToken.User_id
// 		messageData.IsMyMessage = (userID == messageData.SenderID)
// 		if err := conn.WS.WriteJSON(gin.H{"newMessage": messageData}); err != nil {
// 			log.Println("Error writing json to connection:", err)
// 		}
// 	}
// }
