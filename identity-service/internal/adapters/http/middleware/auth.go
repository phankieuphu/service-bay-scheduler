package middleware

import (
	"identity-service/internal/domain/ports"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	// UserIDKey and RoleKey are the gin context keys RequireAuth sets for
	// handlers behind it.
	UserIDKey = "auth.userID"
	RoleKey   = "auth.role"
)

// RequireAuth rejects requests without a valid "Authorization: Bearer
// <access token>" header with 401.
func RequireAuth(tokens ports.TokenIssuer) gin.HandlerFunc {
	return func(c *gin.Context) {
		scheme, token, ok := strings.Cut(c.GetHeader("Authorization"), " ")
		if !ok || !strings.EqualFold(scheme, "Bearer") || token == "" {
			c.Header("WWW-Authenticate", "Bearer")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}

		claims, err := tokens.ParseAccessToken(token)
		if err != nil {
			c.Header("WWW-Authenticate", `Bearer error="invalid_token"`)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		c.Set(UserIDKey, claims.UserID)
		c.Set(RoleKey, claims.Role)
		c.Next()
	}
}
