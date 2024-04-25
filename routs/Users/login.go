package Users

import (
	"github.com/gin-gonic/gin"
)

func UsersLoginRouter(router *gin.Engine) {
	roustBase := "user/"
	router.GET(roustBase+"log-in", login)
}

// getExample godoc
// @Summary Функция для авторизации пользователя
// @Description 2
// @Produce  json
// @Success 200 {object} map[string]interface{} "successful response"
// @Router /user/log-in [get]
func login(c *gin.Context) {
	c.JSON(200, gin.H{"message": "example"})
}
