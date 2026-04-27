package response

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/lifei6671/quantsage/apps/server/internal/pkg/apperror"
)

// Body is the standard HTTP response envelope for QuantSage APIs.
type Body struct {
	Code   int    `json:"code"`
	Errmsg string `json:"errmsg"`
	Toast  string `json:"toast"`
	Data   any    `json:"data"`
}

// OK writes a successful response envelope.
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Body{
		Code:   apperror.CodeOK,
		Errmsg: "",
		Toast:  "",
		Data:   data,
	})
}

// Fail writes a business error response envelope.
func Fail(c *gin.Context, err error) {
	code := apperror.CodeOf(err)
	errmsg, toast := apperror.MessageOf(code)
	c.JSON(http.StatusOK, Body{
		Code:   code,
		Errmsg: errmsg,
		Toast:  toast,
		Data:   struct{}{},
	})
}
