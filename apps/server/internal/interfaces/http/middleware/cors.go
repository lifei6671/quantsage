package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// CORS 为本地前端联调与静态预览提供最小跨域支持。
func CORS(allowedOrigins []string) gin.HandlerFunc {
	allowedOriginSet := buildAllowedOriginSet(allowedOrigins)

	return func(c *gin.Context) {
		header := c.Writer.Header()
		origin := c.GetHeader("Origin")
		if origin != "" && allowedOriginSet[origin] {
			// 只有显式配置过的前端地址才允许读取带 cookie 的私有接口响应，避免任意站点借机读走用户数据。
			header.Set("Access-Control-Allow-Origin", origin)
			header.Set("Access-Control-Allow-Credentials", "true")
			header.Add("Vary", "Origin")
		} else if origin == "" {
			header.Set("Access-Control-Allow-Origin", "*")
		}

		requestHeaders := c.GetHeader("Access-Control-Request-Headers")
		if requestHeaders != "" {
			// 预检请求优先透传浏览器声明的 Header 列表，避免后续新增字段时还要同步硬编码白名单。
			header.Set("Access-Control-Allow-Headers", requestHeaders)
			header.Add("Vary", "Access-Control-Request-Headers")
		} else {
			header.Set("Access-Control-Allow-Headers", "Content-Type, X-Request-Id")
		}
		header.Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func buildAllowedOriginSet(origins []string) map[string]bool {
	result := make(map[string]bool, len(origins))
	for _, origin := range origins {
		trimmed := strings.TrimSpace(origin)
		if trimmed == "" {
			continue
		}
		result[trimmed] = true
	}

	return result
}
