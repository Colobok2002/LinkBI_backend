package main

import (
	"Bmessage_backend/routs/Tokens"
	"Bmessage_backend/routs/Users"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gocql/gocql"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func initialScylla() {
	cluster := gocql.NewCluster("127.0.0.1")
	session, err := cluster.CreateSession()

	if err != nil {
		log.Fatalf("Failed to create session: %v", err)
	}
	defer session.Close()

	var count int

	err = session.Query("SELECT COUNT(*) FROM system_schema.keyspaces WHERE keyspace_name = 'bmessage_system'").Consistency(gocql.One).Scan(&count)
	if err != nil {
		log.Fatalf("Failed to check keyspace existence: %v", err)
	}

	if count > 0 {
		log.Println("Keyspace 'bmessage_system' already exists")
	} else {
		if err := session.Query("CREATE KEYSPACE bmessage_system WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1}").Exec(); err != nil {
			log.Fatalf("Failed to create keyspace: %v", err)
		}
		log.Println("Keyspace 'bmessage_system' created successfully")
	}

	err = session.Query("SELECT COUNT(*) FROM system_schema.tables WHERE keyspace_name = 'bmessage_system' AND table_name = 'users'").Consistency(gocql.One).Scan(&count)
	if err != nil {
		log.Fatalf("Failed to check table existence: %v", err)
	}

	if count > 0 {
		log.Println("Table 'users' in keyspace 'bmessage_system' already exists")
	} else {
		if err := session.Query("CREATE TABLE bmessage_system.users (id UUID PRIMARY KEY, Name TEXT, SoName TEXT, Nik TEXT, login TEXT, password TEXT, PrivateKey TEXT)").Exec(); err != nil {
			log.Fatalf("Failed to create table: %v", err)
		}
		log.Println("Table 'users' in keyspace 'bmessage_system' created successfully")
	}
}

func main() {
	router := gin.Default()

	Users.UsersLoginRouter(router)
	Tokens.TokensRouter(router)

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/swagger-docs")))

	router.GET("/swagger-docs", func(c *gin.Context) {
		c.File("./docs/swagger.json")
	})

	router.GET("/docs", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/swagger/index.html")
	})

	initialScylla()
	log.Println("Server started port 8080")
	router.Run(":8080")
}
