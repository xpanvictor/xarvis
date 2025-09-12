package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type HTTPUserInfo struct {
	UserID uuid.UUID
	Email  string
}

func ExtractUserInfo(c *gin.Context) (HTTPUserInfo, bool) {
	userID := c.GetString("userID") // From JWT middleware
	if userID == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "User not authenticated"})
		return HTTPUserInfo{}, false
	}
	// email := c.GetString("email")
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unable to parse user id"})
		return HTTPUserInfo{}, false
	}

	return HTTPUserInfo{
		UserID: userUUID,
	}, true
}
