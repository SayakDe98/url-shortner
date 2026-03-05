package middleware

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		path := c.Request.URL.Path
		method := c.Request.Method

		c.Next()

		status := c.Writer.Status()
		latency := time.Since(start)

		log.Printf(
			"%s %s | status=%d | latency=%v",
			method,
			path,
			status,
			latency,
		)
	}
}
