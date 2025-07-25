package middleware

import (
        "fmt"
        "net/http"

        "github.com/gin-gonic/gin"
)

func CORS() gin.HandlerFunc {
        return func(c *gin.Context) {
                c.Header("Access-Control-Allow-Origin", "*")
                c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
                c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
                c.Header("Access-Control-Allow-Credentials", "true")

                if c.Request.Method == "OPTIONS" {
                        c.AbortWithStatus(http.StatusNoContent)
                        return
                }

                c.Next()
        }
}

func Logger() gin.HandlerFunc {
        return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
                return fmt.Sprintf("[%s] %s %s %s %d %s %s\n",
                        param.TimeStamp.Format("2006-01-02 15:04:05"),
                        param.ClientIP,
                        param.Method,
                        param.Path,
                        param.StatusCode,
                        param.Latency,
                        param.ErrorMessage,
                )
        })
}
