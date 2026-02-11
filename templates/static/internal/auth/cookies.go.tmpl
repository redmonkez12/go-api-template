package auth

import (
	"errors"
	"net/http"
	"time"
)

const (
	accessTokenCookieName  = "access_token"
	refreshTokenCookieName = "refresh_token"
)

// SetAuthCookies sets both access and refresh token cookies
func SetAuthCookies(w http.ResponseWriter, accessToken, refreshToken string, isProduction bool, accessDuration, refreshDuration time.Duration) {
	// Set access token cookie
	http.SetCookie(w, &http.Cookie{
		Name:     accessTokenCookieName,
		Value:    accessToken,
		Path:     "/",
		MaxAge:   int(accessDuration.Seconds()),
		HttpOnly: true,
		Secure:   isProduction, // Only send over HTTPS in production
		SameSite: http.SameSiteLaxMode,
	})

	// Set refresh token cookie
	http.SetCookie(w, &http.Cookie{
		Name:     refreshTokenCookieName,
		Value:    refreshToken,
		Path:     "/",
		MaxAge:   int(refreshDuration.Seconds()),
		HttpOnly: true,
		Secure:   isProduction, // Only send over HTTPS in production
		SameSite: http.SameSiteLaxMode,
	})
}

// ClearAuthCookies expires both auth cookies immediately
func ClearAuthCookies(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     accessTokenCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     refreshTokenCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
}

// ShouldUseCookies determines if the request should receive cookies
// Returns true if Origin header is present (indicates browser CORS request)
func ShouldUseCookies(r *http.Request) bool {
	return r.Header.Get("Origin") != ""
}

// GetAccessTokenFromCookie retrieves the access token from cookies
func GetAccessTokenFromCookie(r *http.Request) (string, error) {
	cookie, err := r.Cookie(accessTokenCookieName)
	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			return "", errors.New("access token cookie not found")
		}
		return "", err
	}
	return cookie.Value, nil
}

// GetRefreshTokenFromCookie retrieves the refresh token from cookies
func GetRefreshTokenFromCookie(r *http.Request) (string, error) {
	cookie, err := r.Cookie(refreshTokenCookieName)
	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			return "", errors.New("refresh token cookie not found")
		}
		return "", err
	}
	return cookie.Value, nil
}
