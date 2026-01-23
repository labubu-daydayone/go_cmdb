package middleware

import (
	"errors"
	"strings"

	"go_cmdb/internal/auth"
	"go_cmdb/internal/httpx"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// AuthRequired is a middleware that validates JWT token
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			httpx.FailErr(c, httpx.ErrUnauthorized("missing authorization header"))
			c.Abort()
			return
		}

		// Check Bearer prefix
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			httpx.FailErr(c, httpx.ErrUnauthorized("invalid authorization header format"))
			c.Abort()
			return
		}

		tokenString := parts[1]

		// Parse and validate token
		claims, err := auth.ParseToken(tokenString)
		if err != nil {
			// Determine error type
			if errors.Is(err, jwt.ErrTokenExpired) {
				httpx.FailErr(c, httpx.ErrTokenExpired("token expired"))
			} else {
				httpx.FailErr(c, httpx.ErrInvalidToken("invalid token"))
			}
			c.Abort()
			return
		}

		// Set user info in context
		c.Set("uid", claims.UID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)

		c.Next()
	}
}
