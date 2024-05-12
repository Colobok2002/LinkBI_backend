package database

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/gocql/gocql"
)

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

func keyspaceExists(session *gocql.Session, keyspaceName string) (bool, error) {
	query := "SELECT keyspace_name FROM system_schema.keyspaces WHERE keyspace_name = ?"
	var name string
	if err := session.Query(query, keyspaceName).Consistency(gocql.One).Scan(&name); err != nil {
		if err == gocql.ErrNotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func CreateKeyspaceScylla(session *gocql.Session, keyspaceName string) error {
	createKeyspaceQuery := fmt.Sprintf("CREATE KEYSPACE IF NOT EXISTS %s WITH replication = {'class': 'SimpleStrategy', 'replication_factor' : 1};", keyspaceName)
	return session.Query(createKeyspaceQuery).Exec()
}

func InitScylla() {
	session, err := GetSession()
	if err != nil {
		fmt.Printf("Ошибка при подключении к базе данных: %v\n", err)
		return
	}
	defer session.Close()

	keyspaceName := "chat"

	exists, err := keyspaceExists(session, keyspaceName)
	if err != nil {
		fmt.Printf("Ошибка при проверке наличия keyspace %s: %v\n", keyspaceName, err)
		return
	}

	if exists {
		fmt.Printf("Keyspace %s уже существует\n", keyspaceName)
	} else {
		// Создание keyspace, если он не существует
		if err := CreateKeyspaceScylla(session, keyspaceName); err != nil {
			fmt.Printf("Ошибка при создании keyspace %s: %v\n", keyspaceName, err)
		} else {
			fmt.Printf("Keyspace %s успешно создан\n", keyspaceName)
		}
	}
}
