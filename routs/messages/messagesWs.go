package messages

import (
	"Bmessage_backend/database"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gocql/gocql"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func MessageRouterWs(router *gin.Engine) {
	routeBase := "messages/"
	router.GET(routeBase+"/events-messages", func(c *gin.Context) {
		database.WithDatabaseScylla(serveWs)
	})
}

func serveWs(session *gocql.Session, c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("Failed to set websocket upgrade:", err)
		return
	}
	defer conn.Close()

	for {
		var msg Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Println("Error reading json.", err)
			break
		}
		log.Printf("Received message: %+v", msg)

		if err = conn.WriteJSON(msg); err != nil {
			log.Println("Error writing json.", err)
			break
		}
	}
}
