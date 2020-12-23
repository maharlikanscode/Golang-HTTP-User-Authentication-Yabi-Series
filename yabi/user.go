package yabi

import (
	"database/sql"
	"errors"
	"gowebapp/config"
	"strings"
	"time"

	"github.com/itrepablik/sakto"
)

// CreateUser add a new user to the users collection
func CreateUser(dbCon *sql.DB, u User, confirmPassword string, tos bool) (int64, error) {
	// Check if username is empty
	if len(strings.TrimSpace(u.UserName)) == 0 {
		return 0, errors.New("Username is Required")
	}

	// Check if email is empty
	if len(strings.TrimSpace(u.Email)) == 0 {
		return 0, errors.New("Email is Required")
	}

	// Check if email address is valid or not
	if !sakto.IsEmailValid(u.Email) {
		return 0, errors.New("Invalid Email Address, please try again")
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
