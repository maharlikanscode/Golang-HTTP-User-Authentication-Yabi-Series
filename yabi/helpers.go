package yabi

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/itrepablik/itrlog"
	"github.com/itrepablik/tago"
	"github.com/itrepablik/timaan"
)

// LoginRequired is to make any pages protected with the yabi auth system
func LoginRequired(endpoint func(http.ResponseWriter, *http.Request), secretKey string) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read cookie and get the cookie here and decrypt it.
		cookie, err := r.Cookie(YabiCookieName)
		if err != nil {
			itrlog.Error(err)
			LogOut(w, r, secretKey) // Trigger the log-out event
			return
		}

		userName, err := tago.Decrypt(cookie.Value, secretKey) // Decrypt the cookie encrypted username.
		fmt.Println("userName LoginRequired: ", userName)
		if err != nil {
			itrlog.Error(err)
			LogOut(w, r, secretKey) // Trigger the log-out event
			return
		}

		// Extract the Timaan token payload
		tok, err := timaan.DecodePayload(userName)
		if err != nil {
			itrlog.Error(err)
			LogOut(w, r, secretKey) // Trigger the log-out event
			return
		}
		fmt.Println("TokenKey: ", tok.TokenKey)

		// Just for testing, if the payload can be extracted or not
		payLoad := tok.Payload
		for field, val := range payLoad {
			fmt.Println(field, " : ", val)
		}

		// Now, match both decoded username from the gob encoded payload vs the decoded cookie content username
		if strings.TrimSpace(tok.TokenKey) == userName {
			endpoint(w, r) // load whatever the requested protected pages
		} else {
			// otherwise, logout asap for unauthorized user
			LogOut(w, r, secretKey) // Trigger the log-out event
		}
	})
}

// LogOut will be called when the user has been properly logout from the system.
func LogOut(w http.ResponseWriter, r *http.Request, secretKey string) {
	// Read cookie and get the cookie here and decrypt it.
	cookie, err := r.Cookie(YabiCookieName)
	if err != nil {
		itrlog.Error(err)
		ReAuth(w, r) // Back to login page
		return
	}

	// Decrypt the cookie encrypted username.
	userName, err := tago.Decrypt(cookie.Value, secretKey)
	if err != nil {
		itrlog.Error(err)
		ReAuth(w, r) // Back to login page
		return
	}

	// Delete the specified username once logout
	isTokenRemove, err := timaan.UT.Remove(userName)
	if err != nil {
		itrlog.Error(err)
		ReAuth(w, r) // Back to login page
		return
	}
	fmt.Println("isTokenRemove: ", isTokenRemove)
	itrlog.Info("isTokenRemove: ", isTokenRemove)

	// Delete from the "yabi_user_token" table as well
	// *************to follow****************

	// Expire the cookie immediately.
	cookie = &http.Cookie{
		Name:   YabiCookieName,
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	}
	http.SetCookie(w, cookie)

	itrlog.Warn("User has been log-out: ", userName)
	ReAuth(w, r) // Back to the login page
}

// ReAuth will redirect the user's to the login page to re-authenticate if not authenticated.
func ReAuth(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/login", 302)
}

// IsUserAuthenticated is the user's auth indicator wether the user is logged in or not
func IsUserAuthenticated(w http.ResponseWriter, r *http.Request, secretKey string) bool {
	// Read cookie and get the cookie here and decrypt it.
	cookie, err := r.Cookie(YabiCookieName)
	if err != nil {
		itrlog.Error(err)
		return false
	}

	// Decrypt the cookie encrypted username.
	userName, err := tago.Decrypt(cookie.Value, secretKey)
	if err != nil {
		itrlog.Error(err)
		return false
	}

	// Extract the Timaan token payload
	tok, err := timaan.DecodePayload(userName)
	if err != nil {
		itrlog.Error(err)
		return false
	}

	// Now, match both decoded username from the gob encoded payload vs the decoded cookie content username
	if strings.TrimSpace(tok.TokenKey) == userName {
		return true
	}
	return false
}
