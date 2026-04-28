package middleware

import "github.com/gin-gonic/gin"

// CORS 为本地前端联调与静态预览提供最小跨域支持。
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.Writer.Header()
		header.Set("Access-Control-Allow-Origin", "*")
		header.Set("Access-Control-Allow-Headers", "Content-Type, X-Request-Id")
		header.Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
