package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type APIResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   interface{} `json:"error,omitempty"`
}

func Success(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, APIResponse{Status: "success", Message: message, Data: data})
}

func Error(c *gin.Context, code int, message string, err interface{}, data ...interface{}) {
	var d interface{}
	if len(data) > 0 {
		d = data[0]
	}
	c.JSON(code, APIResponse{Status: "failed", Message: message, Error: err, Data: d})
}
