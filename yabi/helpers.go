package yabi

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

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

		// Extract the timaan token payload here
		ybUserName, ybEmail, ybFirstName, ybMiddleName, ybLastName, ybSuffix := "", "", "", "", "", ""
		ybLastLogin, ybDateJoin := time.Now(), time.Now()

		ybIsSuperUser, ybIsAdmin := false, false
		var ybUserID int64 = 0

		payLoad := tok.Payload
		for field, val := range payLoad {
			switch field {
			case "USER_ID":
				ybUserID, _ = strconv.ParseInt(fmt.Sprint(val), 10, 64)
			case "USERNAME":
				ybUserName = fmt.Sprint(val)
			case "EMAIL":
				ybEmail = fmt.Sprint(val)
			case "FIRST_NAME":
				ybFirstName = fmt.Sprint(val)
			case "MIDDLE_NAME":
				ybMiddleName = fmt.Sprint(val)
			case "LAST_NAME":
				ybLastName = fmt.Sprint(val)
			case "SUFFIX":
				ybSuffix = fmt.Sprint(val)
			case "IS_SUPER_USER":
				ybIsSuperUser, _ = strconv.ParseBool(fmt.Sprint(val))
			case "IS_ADMIN":
				ybIsAdmin, _ = strconv.ParseBool(fmt.Sprint(val))
			case "LAST_LOGIN":
				ybLastLogin, _ = time.Parse("020106 150405", fmt.Sprint(val)) // Datetime format September 22, 2002 03:04:05PM
			case "DATE_JOINED":
				ybDateJoin, _ = time.Parse("020106 150405", fmt.Sprint(val)) // Datetime format September 22, 2002 03:04:05PM
			}
		}

		// Feed the yabi user data struct with the current user's auth information
		YBUserData.ID = ybUserID
		YBUserData.UserName = ybUserName
		YBUserData.Email = ybEmail
		YBUserData.FirstName = ybFirstName
		YBUserData.MiddleName = ybMiddleName
		YBUserData.LastName = ybLastName
		YBUserData.Suffix = ybSuffix
		YBUserData.IsSuperUser = ybIsSuperUser
		YBUserData.IsAdmin = ybIsAdmin
		YBUserData.LastLogin = ybLastLogin
		YBUserData.DateJoined = ybDateJoin

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
	_, err = timaan.UT.Remove(userName)
	if err != nil {
		itrlog.Error(err)
		ReAuth(w, r) // Back to login page
		return
	}

	// Delete from the "yabi_user_token" table as well
	dbYabi, err := sql.Open("mysql", YB.DBConStr)
	if err != nil {
		itrlog.Error(err)
	}
	defer dbYabi.Close()
	DeleteUserToken(dbYabi, cookie.Value, YabiTokenAuth)

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
