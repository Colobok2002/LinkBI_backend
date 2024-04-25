package swagger

import (
	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	router := gin.Default()

	router.GET("/example", getExample)

	return router
}

// getExample godoc
// @Summary Show example data
// @Description get string data by id
// @Produce  json
// @Success 200 {object} map[string]interface{} "successful response"
// @Router /example [get]
func getExample(c *gin.Context) {
	c.JSON(200, gin.H{"message": "example"})
}
