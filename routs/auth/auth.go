package auth

import (
	"Bmessage_backend/database"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func AuthRouter(router *gin.Engine) {
	roustBase := "auth/"
	router.POST(roustBase+"get-tokens", database.WithDatabase(GetTokens))
	// router.POST(roustBase+"registration", database.WithDatabase(registerUser))
	// router.POST(roustBase+"chek-token", database.WithDatabase(chekTokenUser))
	// router.POST(roustBase+"check-uniqueness-registration-data", database.WithDatabase(check_uniqueness_registration_data))
}

// GuidTokens represents the JSON structure for a user registration request
// @Description структура для получения токена.
type GUIDTokens struct {
	GUID string `json:"GUID"`
}

// @Tags Auth
// GetTokens godoc
// @Summary эндпойт для получения пары токенов
// @Accept json
// @Produce  json
// @Param data body GUIDTokens true "GUID ID"
// @Success 200 {object} map[string]interface{} "successful response"
// @Failure 400 {object} map[string]interface{} "bad request"
// @Router /user/log-in-with-credentials [post]
func GetTokens(db *gorm.DB, c *gin.Context) {
	var reguestData GUIDTokens

	if err := c.BindJSON(&reguestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Пользователь успешно аутентифицирован"})
}
