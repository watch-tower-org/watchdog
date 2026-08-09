package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	// AccessTokenCookie carries the short-lived JWT. SameSite=Lax keeps it off
	// cross-site requests (CSRF), httpOnly keeps it out of JS (XSS).
	AccessTokenCookie = "wt_access_token"
	// RefreshTokenCookie carries the long-lived JWT used to mint new access
	// tokens.
	RefreshTokenCookie = "wt_refresh_token"
)

func sameSiteLax(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
}

// SetAuthCookies writes the session tokens as httpOnly cookies. maxAge values
// are in seconds and should match the JWT durations.
func SetAuthCookies(c *gin.Context, access, refresh string, accessMaxAge, refreshMaxAge int, secure bool) {
	sameSiteLax(c)
	c.SetCookie(AccessTokenCookie, access, accessMaxAge, "/", "", secure, true)
	c.SetCookie(RefreshTokenCookie, refresh, refreshMaxAge, "/", "", secure, true)
}

// RefreshAccessCookie updates just the access token cookie after a refresh.
func RefreshAccessCookie(c *gin.Context, access string, maxAge int, secure bool) {
	sameSiteLax(c)
	c.SetCookie(AccessTokenCookie, access, maxAge, "/", "", secure, true)
}

// ClearAuthCookies expires both session cookies (logout).
func ClearAuthCookies(c *gin.Context) {
	sameSiteLax(c)
	c.SetCookie(AccessTokenCookie, "", -1, "/", "", false, true)
	c.SetCookie(RefreshTokenCookie, "", -1, "/", "", false, true)
}
