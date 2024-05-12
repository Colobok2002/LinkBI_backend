package database

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/gocql/gocql"
)

func CreateKeyspaceScylla(session *gocql.Session, keyspaceName string) error {
	createKeyspaceQuery := fmt.Sprintf("CREATE KEYSPACE IF NOT EXISTS %s WITH replication = {'class': 'SimpleStrategy', 'replication_factor' : 1};", keyspaceName)
	return session.Query(createKeyspaceQuery).Exec()
}

func GetSession() (*gocql.Session, error) {
	cluster := gocql.NewCluster(os.Getenv("SCYLLA_HOST"))
	cluster.Port = 9042
	cluster.Consistency = gocql.Quorum

	// Создание сессии
	session, err := cluster.CreateSession()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to ScyllaDB: %w", err)
	}
	return session, nil
}

type HandlerFuncScylla func(session *gocql.Session, c *gin.Context)

func WithDatabaseScylla(handler HandlerFuncScylla) gin.HandlerFunc {
	return func(c *gin.Context) {
		session, err := GetSession()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось подключиться к базе данных"})
			return
		}
		defer session.Close()
		handler(session, c)
	}
}
