package yabi

import (
	"database/sql"
	"errors"
	"fmt"
	"gowebapp/config"
	"strings"
	"time"

	"github.com/itrepablik/itrlog"
	"github.com/itrepablik/sakto"
	"github.com/itrepablik/sulat"
	"github.com/itrepablik/tago"
	"github.com/itrepablik/timaan"
)

// LoginUser validate the user's account from the use
func LoginUser(dbCon *sql.DB, u User, isSiteKeepMe bool, expireInDays int) (bool, error) {
	// Check if userName is empty
	if len(strings.TrimSpace(u.UserName)) == 0 {
		return false, errors.New("Username is Required")
	}

	// Check if password is empty
	if len(strings.TrimSpace(u.Password)) == 0 {
		return false, errors.New("Password is Required")
	}

	// Get the user's stored hash password.
	pwHash, err := GetUserPassword(dbCon, u.UserName)
	if err != nil {
		return false, errors.New("Oops!, error getting user's credential, please try again")
	}

	// Now, match the two passwords, check if it's verified or not.
	isPassHashMatch, err := sakto.CheckPasswordHash(u.Password, pwHash)
	if err != nil {
		return false, errors.New("Oops!, either of your username or password is wrong, please try again, thank you")
	}

	if isPassHashMatch {
		// Get the user's information from the "yabi_user" table
		mUser := GetUserInfo(dbCon, u.UserName)

		// Generate new timaan token for the new user's session token
		// Must convert all different type of values to a string value
		tokenPayload := timaan.TP{
			"USER_ID":       fmt.Sprint(mUser.ID),
			"USERNAME":      fmt.Sprint(mUser.UserName),
			"EMAIL":         fmt.Sprint(mUser.Email),
			"FIRST_NAME":    fmt.Sprint(mUser.FirstName),
			"MIDDLE_NAME":   fmt.Sprint(mUser.MiddleName),
			"LAST_NAME":     fmt.Sprint(mUser.LastName),
			"SUFFIX":        fmt.Sprint(mUser.Suffix),
			"IS_SUPER_USER": fmt.Sprint(mUser.IsSuperUser),
			"IS_ADMIN":      fmt.Sprint(mUser.IsAdmin),
			"LAST_LOGIN":    fmt.Sprint(mUser.LastLogin),
			"DATE_JOINED":   fmt.Sprint(mUser.DateJoined),
		}

		// Set the user's cookie expiry in days, if not provided, yabi use its default value to 30 days
		if expireInDays < 1 {
			expireInDays = ExpireCookieInDays // expire in 30 days
		}

		// Check if the isSiteKeepMe = true or not
		var tokenExpiry int64 = time.Now().Add(time.Minute * 30).Unix() // default to 30 minutes
		if isSiteKeepMe {
			tokenExpiry = time.Now().Add(time.Hour * time.Duration(24*expireInDays)).Unix()
		}

		tok := timaan.TK{
			TokenKey: mUser.UserName,
			Payload:  tokenPayload,
			ExpireOn: tokenExpiry,
		}

		encTokenBytes, err := timaan.GenerateToken(mUser.UserName, tok)
		if err != nil {
			itrlog.Error("error generating token during login: ", err)
			return false, errors.New("Oops!, there was an error during encoding process, please try again, thank you")
		}

		// Encrypt the username value to store it from the user's cookie
		encUserName, err := tago.Encrypt(mUser.UserName, config.MyEncryptDecryptSK)
		if err != nil {
			itrlog.Error("ERROR FROM encUserName: ", err)
		}

		// Store the authentication token
		err = KeepToken(dbCon, encUserName, "auth", encTokenBytes, tokenExpiry)
		if err != nil {
			itrlog.Error("ERROR FROM KeepToken: ", err)
			return false, errors.New("Oops!, keeping your session failed, please try again")
		}

		// Update the user's last login to a current timestamp
		LastLogin(dbCon, mUser.UserName)

		// Get the user's stored hash password from the yabi_user table
		return true, nil
	}
	return false, errors.New("Invalid Credentials, either of your username or password is wrong, please try again, thank you")
}

// CreateUser add a new user to the users collection
func CreateUser(dbCon *sql.DB, u User, e EmailConfig, confirmPassword string, tos bool) (int64, error) {
	// Check if username is empty
	if len(strings.TrimSpace(u.UserName)) == 0 {
		return 0, errors.New("Username is Required")
	}

	// Check if the username is available or not
	if !IsUserNameExist(dbCon, u.UserName) {
		return 0, errors.New("Username is not available, please try again")
	}

	// Check if email is empty
	if len(strings.TrimSpace(u.Email)) == 0 {
		return 0, errors.New("Email is Required")
	}

	// Check if email address is valid or not
	if !sakto.IsEmailValid(u.Email) {
		return 0, errors.New("Invalid Email Address, please try again")
	}

	// Check if the email address is available or not
	if !IsUserEmailExist(dbCon, u.Email) {
		return 0, errors.New("Email is not available, please try again")
	}

	// Check if password is empty
	if len(strings.TrimSpace(u.Password)) == 0 {
		return 0, errors.New("Password is Required")
	}

	// Match both passwords
	if strings.TrimSpace(confirmPassword) != strings.TrimSpace(u.Password) {
		return 0, errors.New("Passwords didn't match, please try again")
	}

	// Check if Terms of service has been checked
	if !tos {
		return 0, errors.New("Terms of Service is Required, By joining " + config.SiteShortName + ", you're agreeing to our terms and conditions.")
	}

	if !u.IsActive {
		// Check if from email address is empty
		if len(strings.TrimSpace(e.From)) == 0 {
			return 0, errors.New("From email address is required")
		}

		// Check if to email address is empty
		if len(strings.TrimSpace(e.To)) == 0 {
			return 0, errors.New("To email address is required")
		}

		// Check if subject is empty
		emailSubject := "Activate your new account"
		if len(strings.TrimSpace(e.Subject)) > 0 {
			emailSubject = e.Subject
		}

		// Check if HTML Header template has been customized
		emailHTMLHeader := YabiHTMLHeader // default to Yabi HTML Header
		if len(strings.TrimSpace(e.CustomizeHeaderTemplate)) > 0 {
			emailHTMLHeader = e.CustomizeHeaderTemplate
		}

		// Check if HTML Body template has been customized
		emailHTMLBody := NewUserActivation(e.EmailConfirmationURL, u.UserName, e.SiteName, e.SiteSupportEmail) // default to Yabi HTML Body
		if len(strings.TrimSpace(e.CustomizeBodyTemplate)) > 0 {
			emailHTMLBody = e.CustomizeBodyTemplate
		}

		// Check if HTML Footer template has been customized
		emailHTMLFooter := YabiHTMLFooter // default to Yabi HTML Footer
		if len(strings.TrimSpace(e.CustomizeFooterTemplate)) > 0 {
			emailHTMLFooter = e.CustomizeFooterTemplate
		}

		// Send an email confirmation now, prepare the HTML email content first
		mailOpt := &sulat.SendMail{
			Subject: emailSubject,
			From:    sulat.NewEmail(e.FromAlias, e.From),
			To:      sulat.NewEmail(e.ToAlias, e.To),
			CC:      sulat.NewEmail(e.CCAlias, e.CC),
			BCC:     sulat.NewEmail(e.BCCAlias, e.BCC),
		}
		htmlContent, err := sulat.SetHTML(&sulat.EmailHTMLFormat{
			IsFullHTML: false,
			HTMLHeader: emailHTMLHeader,
			HTMLBody:   emailHTMLBody,
			HTMLFooter: emailHTMLFooter,
		})
		_, err = sulat.SendEmailSG(mailOpt, htmlContent, &SGC)
		if err != nil {
			itrlog.Error("SendGrid error: ", err)
		}
	}

	// Hash and salt your plain text password
	hsPassword, err := sakto.HashAndSalt([]byte(u.Password))
	if err != nil {
		return 0, err
	}

	// Now, insert the new user's information here
	ins, err := dbCon.Prepare("INSERT INTO " + YabiUser + " (username, password, email, first_name, " +
		"middle_name, last_name, suffix, is_superuser, is_admin, date_joined, is_active) VALUES" +
		"(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")

	if err != nil {
		return 0, err
	}

	// Pass on all the parameter values here
	ins.Exec(u.UserName, hsPassword, u.Email, u.FirstName, u.MiddleName, u.LastName, u.Suffix, u.IsSuperUser,
		u.IsAdmin, time.Now(), u.IsActive)

	// Get the lastest inserted id
	lid, err := GetLastInsertedID(dbCon, "id", YabiUser)
	defer ins.Close()
	return lid, nil
}

// GetLastInsertedID gets the latest inserted id for any specified table and it's auto_increment field
func GetLastInsertedID(dbCon *sql.DB, autoIDFieldName, tableName string) (int64, error) {
	var id int64 = 0
	err := dbCon.QueryRow("SELECT " + autoIDFieldName + " FROM " + tableName + " ORDER BY " + autoIDFieldName + " DESC LIMIT 1").Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

// IsUserNameExist check from the user's collection if it's existed or not, we don't allow to have a
// duplicate username, it must be a unique value
func IsUserNameExist(dbCon *sql.DB, userName string) bool {
	var id int64 = 0
	err := dbCon.QueryRow("SELECT id FROM "+YabiUser+" WHERE username = ?", userName).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return true // returned no rows, the username is not found from the yabi table
		}
		return false
	}
	return false
}

// IsUserEmailExist check from the user's collection if it's existed or not, we don't allow to have a
// duplicate email, it must be a unique value
func IsUserEmailExist(dbCon *sql.DB, email string) bool {
	var id int64 = 0
	err := dbCon.QueryRow("SELECT id FROM "+YabiUser+" WHERE email = ?", email).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return true // returned no rows, the email is not found from the yabi table
		}
		return false
	}
	return false
}

// ActivateUser will update the specific user's status to true/active
func ActivateUser(dbCon *sql.DB, userName string) bool {
	upd, err := dbCon.Prepare("UPDATE " + YabiUser + " SET is_active = ? WHERE username = ?")
	if err != nil {
		itrlog.Error("ERROR FROM ActivateUser: ", err)
		return false
	}
	// Pass on all the parameter values here
	upd.Exec(true, userName) // activate the user's status now
	defer upd.Close()
	return true
}

// GetUserPassword gets the hash password stored in the yabi_user table
func GetUserPassword(dbCon *sql.DB, userName string) (string, error) {
	encPassword := ""
	err := dbCon.QueryRow("SELECT password FROM "+YabiUser+" WHERE username = ? AND is_active = ?", userName, "1").Scan(&encPassword)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", err // returned no rows, the email is not found from the yabi table
		}
		return "", err
	}
	return encPassword, nil
}

// GetUserInfo gets the user's information from the "yabi_user" table
func GetUserInfo(dbCon *sql.DB, userName string) *User {
	var u User
	err := dbCon.QueryRow("SELECT id, username, email, first_name, middle_name, last_name, suffix, is_superuser, "+
		"is_admin, IFNULL(last_login, NOW()), date_joined FROM "+YabiUser+" WHERE username = ? ORDER BY ID DESC LIMIT 1", userName).Scan(&u.ID,
		&u.UserName, &u.Email, &u.FirstName, &u.MiddleName, &u.LastName, &u.Suffix, &u.IsSuperUser,
		&u.IsAdmin, &u.LastLogin, &u.DateJoined)
	if err != nil {
		itrlog.Error("ERROR FROM GetUserInfO: ", userName, ":", err)
		return &User{}
	}
	return &u
}

// LastLogin will update the specific user's last login with the current timestamp
func LastLogin(dbCon *sql.DB, userName string) {
	upd, err := dbCon.Prepare("UPDATE " + YabiUser + " SET last_login = ? WHERE username = ? AND is_active = ?")
	if err != nil {
		itrlog.Error("ERROR FROM LastLogin: ", err)
	}
	// Pass on all the parameter values here
	upd.Exec(time.Now(), userName, true)
	defer upd.Close()
}

// DeleteUserToken will physically delete the specific user's token during logout process
// This process will delete all of the user's specified token key and its token src
func DeleteUserToken(dbCon *sql.DB, encUserName, tokenSrc string) {
	upd, err := dbCon.Prepare("DELETE FROM " + YabiUserToken + " WHERE token_key = ? " +
		"AND token_src = ? AND expire_on >= ?")
	if err != nil {
		itrlog.Error("ERROR FROM DeleteUserToken: ", err)
	}
	// Pass on all the parameter values here
	upd.Exec(encUserName, tokenSrc, time.Now().Unix())
	defer upd.Close()
}

// KeepToken stores the session token to the "yabi_user_token" table to make it persistent
func KeepToken(dbCon *sql.DB, tokenKey, tokenSrc string, tokenData []byte, expireOn int64) error {
	// Now, insert the new user's auth token here
	ins, err := dbCon.Prepare("INSERT INTO " + YabiUserToken + " (token_key, token_data, token_src, expire_on) VALUES" +
		"(?, ?, ?, ?)")

	if err != nil {
		return err
	}

	// Pass on all the parameter values here
	ins.Exec(tokenKey, tokenData, tokenSrc, expireOn)

	defer ins.Close()
	return nil
}

// RestoreToken restores all the valid, unexpired tokens back to the map collections
func RestoreToken(dbCon *sql.DB, secretKey string) {
	tokens, err := dbCon.Query("SELECT token_key, token_data, token_src, expire_on "+
		"FROM "+YabiUserToken+" WHERE expire_on >= ?", time.Now().Unix())

	if err != nil {
		itrlog.Error("ERROR FROM RestoreToken:", err)
	}

	for tokens.Next() {
		var t UserToken
		err = tokens.Scan(&t.TokenKey, &t.TokenData, &t.TokenSrc, &t.ExpireOn)

		if err != nil {
			itrlog.Error("ERROR FROM RestoreToken at tokens.Next():", err)
		}

		// Decrypt the username from the "yabi_user_token" table
		userName, err := tago.Decrypt(t.TokenKey, secretKey)
		if err != nil {
			itrlog.Error(err)
		} else {
			timaan.UT.Add(userName, []byte(t.TokenData))
		}
	}
}

// ValidatePasswordReset validate the user's registered email address to reset the password
func ValidatePasswordReset(dbCon *sql.DB, e EmailConfig) (bool, error) {
	// Check if email is empty
	if len(strings.TrimSpace(e.To)) == 0 {
		return false, errors.New("Email is Required")
	}

	// Check if email address is valid or not
	if !sakto.IsEmailValid(e.To) {
		return false, errors.New("Invalid Email Address, please try again")
	}

	// Check if the email address exist or not
	if IsUserEmailExist(dbCon, e.To) {
		return false, errors.New("Email is not Found, please try again")
	}

	// Check if subject is empty
	emailSubject := "Password reset"
	if len(strings.TrimSpace(e.Subject)) > 0 {
		emailSubject = e.Subject
	}

	// Check if HTML Header template has been customized
	emailHTMLHeader := YabiHTMLHeader // default to Yabi HTML Header
	if len(strings.TrimSpace(e.CustomizeHeaderTemplate)) > 0 {
		emailHTMLHeader = e.CustomizeHeaderTemplate
	}

	// Check if HTML Body template has been customized
	emailHTMLBody := PasswordResetEmail(e.EmailConfirmationURL, e.To, e.SiteName, e.SiteSupportEmail) // default to Yabi HTML Body
	if len(strings.TrimSpace(e.CustomizeBodyTemplate)) > 0 {
		emailHTMLBody = e.CustomizeBodyTemplate
	}

	// Check if HTML Footer template has been customized
	emailHTMLFooter := YabiHTMLFooter // default to Yabi HTML Footer
	if len(strings.TrimSpace(e.CustomizeFooterTemplate)) > 0 {
		emailHTMLFooter = e.CustomizeFooterTemplate
	}

	// Send an email confirmation now, prepare the HTML email content first
	mailOpt := &sulat.SendMail{
		Subject: emailSubject,
		From:    sulat.NewEmail(e.FromAlias, e.From),
		To:      sulat.NewEmail(e.ToAlias, e.To),
		CC:      sulat.NewEmail(e.CCAlias, e.CC),
		BCC:     sulat.NewEmail(e.BCCAlias, e.BCC),
	}
	htmlContent, err := sulat.SetHTML(&sulat.EmailHTMLFormat{
		IsFullHTML: false,
		HTMLHeader: emailHTMLHeader,
		HTMLBody:   emailHTMLBody,
		HTMLFooter: emailHTMLFooter,
	})
	_, err = sulat.SendEmailSG(mailOpt, htmlContent, &SGC)
	if err != nil {
		itrlog.Error("SendGrid error: ", err)
	}
	return true, nil
}
