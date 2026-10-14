package middleware

import (
	"net/http"
	"scheduler-service/internal/domain/entity"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"
)

// PrincipalKey is the gin context key RequireAuth stores the caller under.
const PrincipalKey = "auth.principal"

// TokenVerifier checks an access token and says who it belongs to.
type TokenVerifier interface {
	ParseAccessToken(token string) (entity.Principal, error)
}

// RequireAuth rejects requests without a valid identity-service access
// token ("Authorization: Bearer <token>") with 401.
func RequireAuth(tokens TokenVerifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		scheme, token, ok := strings.Cut(c.GetHeader("Authorization"), " ")
		if !ok || !strings.EqualFold(scheme, "Bearer") || token == "" {
			c.Header("WWW-Authenticate", "Bearer")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}

		principal, err := tokens.ParseAccessToken(token)
		if err != nil {
			c.Header("WWW-Authenticate", `Bearer error="invalid_token"`)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		// Kept so calls to other services can act as this caller.
		principal.Token = token
		c.Set(PrincipalKey, principal)
		c.Next()
	}
}

// RequireRole rejects with 403 unless the caller's role is one of roles.
// It must run after RequireAuth.
func RequireRole(roles ...entity.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal, ok := c.Value(PrincipalKey).(entity.Principal)
		if !ok || !slices.Contains(roles, principal.Role) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "your role isn't allowed to do this"})
			return
		}
		c.Next()
	}
}
