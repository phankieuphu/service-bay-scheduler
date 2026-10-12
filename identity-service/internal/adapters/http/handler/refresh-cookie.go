package handler

import (
	"identity-service/internal/domain/ports"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	// RefreshCookieName holds the refresh token. It's HttpOnly, so page
	// scripts (and any XSS) can't read it, and it's scoped to the auth
	// routes, so it isn't sent anywhere else.
	RefreshCookieName = "refresh_token"
	refreshCookiePath = "/api/v1/auth"

	// CSRFHeader must be present on the routes that read the cookie. A
	// cross-site form or link can't set a custom header, and a cross-origin
	// fetch that does needs a CORS preflight, which only listed origins pass.
	// Together with SameSite=Strict that's enough CSRF protection here.
	CSRFHeader = "X-Requested-With"
)

// RefreshCookies writes and reads the refresh-token cookie.
type RefreshCookies struct {
	Secure bool
}

func (rc RefreshCookies) set(c *gin.Context, token string, expiresAt time.Time) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     RefreshCookieName,
		Value:    token,
		Path:     refreshCookiePath,
		Expires:  expiresAt,
		MaxAge:   max(int(time.Until(expiresAt).Seconds()), 1),
		HttpOnly: true,
		Secure:   rc.Secure,
		SameSite: http.SameSiteStrictMode,
	})
}

func (rc RefreshCookies) clear(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     RefreshCookieName,
		Value:    "",
		Path:     refreshCookiePath,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   rc.Secure,
		SameSite: http.SameSiteStrictMode,
	})
}

// read returns the cookie's token. It rejects the request (and returns
// false) if the CSRF header is missing, or, when required, the cookie is.
func (rc RefreshCookies) read(c *gin.Context, required bool) (string, bool) {
	if c.GetHeader(CSRFHeader) == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "missing " + CSRFHeader + " header"})
		return "", false
	}
	token, err := c.Cookie(RefreshCookieName)
	if (err != nil || token == "") && required {
		c.JSON(http.StatusUnauthorized, gin.H{"error": ports.ErrInvalidToken.Error()})
		return "", false
	}
	return token, true
}
