package yabi

import "time"

// YabiUser is the exact table name for the "yabi_user" table
const YabiUser = "yabi_user"

// YabiTokenAuth is the login auth indicator for the yabi auth system
const YabiTokenAuth = "auth"

// User model collections for the user's basic information
type User struct {
	ID          int64     `json:"id"`
	UserName    string    `json:"username"`
	Password    string    `json:"password"`
	Email       string    `json:"email"`
	FirstName   string    `json:"first_name"`
	MiddleName  string    `json:"middle_name"`
	LastName    string    `json:"last_name"`
	Suffix      string    `json:"suffix"`
	IsSuperUser bool      `json:"is_superuser"`
	IsAdmin     bool      `json:"is_admin"`
	LastLogin   time.Time `json:"last_login"`
	DateJoined  time.Time `json:"date_joined"`
	IsActive    bool      `json:"is_active"`
}

// YabiUserToken is the exact table name for the "yabi_user_token" table
const YabiUserToken = "yabi_user_token"

// UserToken model collections for the user's persisted timaan token
type UserToken struct {
	TokenKey  string `json:"token_key"`
	TokenData []byte `json:"token_data"`
	TokenSrc  string `json:"token_src"`
	ExpireOn  int64  `json:"expire_on"`
}
