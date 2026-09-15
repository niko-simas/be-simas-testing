package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"simas-backend/internal/auth"
	"simas-backend/internal/model"
	"simas-backend/internal/repository"
)

type AuthMiddleware struct {
	jwtManager *auth.JWTManager
	userRepo   *repository.UserRepository
}

func NewAuthMiddleware(jwtManager *auth.JWTManager, userRepo *repository.UserRepository) *AuthMiddleware {
	return &AuthMiddleware{
		jwtManager: jwtManager,
		userRepo:   userRepo,
	}
}

// RequireAuth validates JWT and injects user into context
func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || len(authHeader) < 8 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing_token"})
			c.Abort()
			return
		}

		tokenString := authHeader[7:] // strip "Bearer "
		claims, err := m.jwtManager.Validate(tokenString)
		if err != nil {
			status := http.StatusUnauthorized
			msg := "invalid_token"
			if err == auth.ErrTokenExpired {
				msg = "token_expired"
			} else if err == auth.ErrTokenRevoked {
				msg = "token_revoked"
			}
			c.JSON(status, gin.H{"error": msg})
			c.Abort()
			return
		}

		if claims.TokenType == "temp" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "temp_token_not_allowed"})
			c.Abort()
			return
		}

		userID, err := uuid.Parse(claims.UserID.String())
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_user_id"})
			c.Abort()
			return
		}

		user, err := m.userRepo.FindByID(c.Request.Context(), userID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "user_not_found"})
			c.Abort()
			return
		}

		c.Set("user", user)
		c.Set("token", tokenString)
		c.Next()
	}
}

// OptionalAuth validates JWT if present but doesn't block
func (m *AuthMiddleware) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || len(authHeader) < 8 {
			c.Next()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := m.jwtManager.Validate(tokenString)
		if err != nil {
			c.Next()
			return
		}

		userID, _ := uuid.Parse(claims.UserID.String())
		user, _ := m.userRepo.FindByID(c.Request.Context(), userID)
		if user != nil {
			c.Set("user", user)
			c.Set("token", tokenString)
		}
		c.Next()
	}
}

// RequireRole blocks requests where the authenticated user does not have the required role
func (m *AuthMiddleware) RequireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userVal, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}
		user := userVal.(*model.User)
		if user.Role != role {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// InjectUser is a helper to get user from context
func InjectUser(c *gin.Context) *model.User {
	userVal, exists := c.Get("user")
	if !exists {
		return nil
	}
	return userVal.(*model.User)
}
