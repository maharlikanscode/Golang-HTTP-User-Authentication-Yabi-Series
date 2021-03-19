package yabi

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/itrepablik/itrlog"
	"github.com/itrepablik/timaan"
)

// YabiCookieName is the default cookie name for the yabi auth system
const YabiCookieName = "yabi"

// ExpireCookieInDays is the default user's cookie expiration in 30 days if not provided
var ExpireCookieInDays int = 30 // number of days

// RemoveExpiredTokens is the default scheduled time in seconds to auto remove expired persisted tokens
var RemoveExpiredTokens int = 1800 // number of seconds, default to 1800 seconds which is 30 mins

// InitYabi initializes the common configurations that yabi package use
type InitYabi struct {
	BaseURL                string     // e.g http://127.0.0.1:8081/ or https://maharlikanscode.com/ with the trailing "/" slash
	DBConStr               string     // MySQL database connection string
	AutoRemoveExpiredToken int        // value must be in seconds, it will be permanently delete the rows from the "yabi_user_token" table
	mu                     sync.Mutex // ensures atomic writes; protect the following fields
}

// YB is the pointer for InitYabi configuration
var YB *InitYabi

// InitYabiConfig initialize all the necessary yabi configurations and its default values
func InitYabiConfig(b *InitYabi) *InitYabi {
	// Check all the required configurations are in place or not
	if len(strings.TrimSpace(b.BaseURL)) == 0 {
		b.BaseURL = "http://127.0.0.1:8081/"
	}

	// Set the default time interval to auto remove the expired persisted tokens
	if b.AutoRemoveExpiredToken <= 0 {
		b.AutoRemoveExpiredToken = RemoveExpiredTokens
	}

	// Run task to auto remove expired persisted tokens
	go TaskRemovePersistedTokens()

	// Run the auto remove expired timaan tokens stored in the memory
	go TaskRemoveMapTokens()

	return &InitYabi{
		BaseURL:                b.BaseURL,
		DBConStr:               b.DBConStr,
		AutoRemoveExpiredToken: b.AutoRemoveExpiredToken,
	}
}

func init() {
	// Set initial default yabi config
	YB = InitYabiConfig(&InitYabi{})
}

// SetYabiConfig sets the custom config values from the user
func SetYabiConfig(b *InitYabi) *InitYabi {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Check all the required configurations are in place or not
	if len(strings.TrimSpace(b.BaseURL)) == 0 {
		b.BaseURL = "http://127.0.0.1:8081/"
	}

	// Re-configure the yabi configurations
	b = InitYabiConfig(b)
	YB = b // Must re-assign whatever the new custom config values
	return b
}

// TaskRemovePersistedTokens task to auto remove the persisted token stored in "yabi_user_token" table
func TaskRemovePersistedTokens() {
	for {
		if len(strings.TrimSpace(YB.DBConStr)) > 0 {
			// Delete from the "yabi_user_token" table as well
			dbCon, err := sql.Open("mysql", YB.DBConStr)
			if err != nil {
				itrlog.Error(err)
			}
			defer dbCon.Close()

			upd, err := dbCon.Prepare("DELETE FROM " + YabiUserToken + " WHERE expire_on < ?")
			if err != nil {
				itrlog.Error("ERROR FROM TaskDeleteUserToken: ", err)
			}

			// Pass on all the parameter values here
			upd.Exec(time.Now().Unix())
			defer upd.Close()
		}
		time.Sleep(time.Duration(YB.AutoRemoveExpiredToken) * time.Second)
	}
}

// TaskRemoveMapTokens task to auto remove expired tokens from the timaan token which is stored in the memory
func TaskRemoveMapTokens() {
	for {
		for userName := range timaan.UT.Token {
			// Decode the timaan payload
			tok, err := timaan.DecodePayload(userName)
			if err != nil {
				itrlog.Error(err)
			}

			// Remove any expired tokens
			if tok.ExpireOn < time.Now().Unix() {
				isTokenRemove, err := timaan.UT.Remove(userName)
				if err != nil {
					itrlog.Error(err)
				}
				fmt.Println("isTokenRemove: ", isTokenRemove)
			}
		}
		time.Sleep(5 * time.Second)
	}
}

// YBUserData is the user's log-in successfully with its collected information
var YBUserData = User{
	ID:          0,
	UserName:    "",
	Email:       "",
	FirstName:   "",
	MiddleName:  "",
	LastName:    "",
	Suffix:      "",
	IsSuperUser: false,
	IsAdmin:     false,
	LastLogin:   time.Now(),
	DateJoined:  time.Now(),
	IsActive:    false,
}
