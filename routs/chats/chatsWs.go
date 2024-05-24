package chats

import (
	"Bmessage_backend/database"
	"Bmessage_backend/helpers"
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
	connections   = make(map[uint][]ConnectionData)
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
	routeBase := "chatsWS/"
	router.GET(routeBase+"events-chats", database.WithDatabaseScylla(cahtsConnectorWs))
}

func cahtsConnectorWs(session *gocql.Session, c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("Failed to set websocket upgrade:", err)
		return
	}
	defer conn.Close()

	userToken := c.Query("userToken")
	userDataToToken, err := helpers.DecryptAES(userToken)
	if err != nil {
		log.Println("Error writing json to connection:", err)
		return
	}

	userID := userDataToToken.User_id

	wsData := ConnectionData{
		WS: conn,
	}

	addConnection(userID, wsData)
	defer removeConnection(userID, conn)

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

func addConnection(userID uint, data ConnectionData) {
	connectionsMu.Lock()
	defer connectionsMu.Unlock()
	connections[userID] = append(connections[userID], data)
}

func removeConnection(userID uint, conn *websocket.Conn) {
	connectionsMu.Lock()
	defer connectionsMu.Unlock()
	conns := connections[userID]
	for i, c := range conns {
		if c.WS == conn {
			connections[userID] = append(conns[:i], conns[i+1:]...)
			break
		}
	}
}

func UpdeteDataChat(userID uint, chatID gocql.UUID) {
	connectionsMu.Lock()
	defer connectionsMu.Unlock()

	chatData, _ := GetChatDetails(userID, chatID)
	conns, ok := connections[userID]
	if !ok {
		return
	}

	for _, conn := range conns {
		if err := conn.WS.WriteJSON(gin.H{"chatUpdate": chatData}); err != nil {
			log.Println("Error writing json to connection:", err)
		}
	}
}
