package auth

import (
	"errors"
	"time"

	"go_cmdb/internal/auth"
	"go_cmdb/internal/config"
	"go_cmdb/internal/httpx"
	"go_cmdb/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// LoginRequest represents login request body
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse represents login response data
type LoginResponse struct {
	Token    string    `json:"token"`
	ExpireAt string    `json:"expireAt"`
	User     UserInfo  `json:"user"`
}

// UserInfo represents user information in response
type UserInfo struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

// LoginHandler handles user login
func LoginHandler(db *gorm.DB, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			httpx.FailErr(c, httpx.ErrParamInvalid("invalid request body"))
			return
		}

		// Query user by username
		var user model.User
		if err := db.Where("username = ?", req.Username).First(&user).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// User not found or wrong password - return same error for security
				httpx.FailErr(c, httpx.ErrInvalidToken("invalid credentials"))
				return
			}
			// Database error
			httpx.FailErr(c, httpx.ErrDatabaseError("database error", err))
			return
		}

		// Check user status
		if user.Status == model.UserStatusInactive {
			httpx.FailErr(c, httpx.ErrForbidden("user is inactive"))
			return
		}

		// Verify password
		if err := auth.ComparePassword(user.PasswordHash, req.Password); err != nil {
			// Wrong password
			httpx.FailErr(c, httpx.ErrInvalidToken("invalid credentials"))
			return
		}

		// Generate JWT token
		expireAt := time.Now().Add(time.Duration(cfg.JWT.ExpireMinutes) * time.Minute)
		token, err := auth.GenerateToken(user.ID, user.Username, user.Role, expireAt, cfg.JWT.Issuer)
		if err != nil {
			httpx.FailErr(c, httpx.ErrInternalError("failed to generate token", err))
			return
		}

		// Return success response
		httpx.OK(c, LoginResponse{
			Token:    token,
			ExpireAt: expireAt.Format(time.RFC3339),
			User: UserInfo{
				ID:       user.ID,
				Username: user.Username,
				Role:     user.Role,
			},
		})
	}
}
