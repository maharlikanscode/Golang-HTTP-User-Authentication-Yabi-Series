package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"gowebapp/config"
	"gowebapp/yabi"
	"html/template"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
	"github.com/itrepablik/itrlog"
	"github.com/itrepablik/sulat"
	"github.com/itrepablik/tago"
	"github.com/itrepablik/timaan"
)

// SGC initialize this variable globally sulat.SendGridConfig{}
var SGC = sulat.SGC{}

func init() {
	// Set Yabi SMTP option
	SGC = yabi.SetSendGridAPI(sulat.SGC{
		SendGridAPIKey: config.SendGridAPIKey,
	})
}

// AuthRouters are the collection of all URLs for the Auth App.
func AuthRouters(r *mux.Router) {
	r.NotFoundHandler = http.HandlerFunc(PageNotFound)
	r.HandleFunc("/api/v1/user/login", LoginUserEndpoint).Methods("POST")
	r.HandleFunc("/api/v1/user/register", RegisterUserEndpoint).Methods("POST")
	r.HandleFunc("/login", Login).Methods("GET")
	r.HandleFunc("/register", Register).Methods("GET")
	r.HandleFunc("/account_activation_sent", AccountActivationSent).Methods("GET")
	r.HandleFunc("/activate/{token}", ActivateAccount).Methods("GET")
	r.HandleFunc("/logout", Logout).Methods("GET")
}

// Logout function is to render the logout script from yabi
func Logout(w http.ResponseWriter, r *http.Request) {
	yabi.LogOut(w, r, config.MyEncryptDecryptSK)
}

// Register function is to render the user's registration page.
func Register(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles(config.SiteRootTemplate+"front/registration.html", config.SiteHeaderTemplate, config.SiteFooterTemplate))

	data := contextData{
		"PageTitle":    "Register - " + config.SiteShortName,
		"PageMetaDesc": config.SiteShortName + " create new user.",
		"CanonicalURL": r.RequestURI,
		"CsrfToken":    csrf.Token(r),
		"Settings":     config.SiteSettings,
	}
	tmpl.Execute(w, data)
}

// Login function is to render the homepage page.
func Login(w http.ResponseWriter, r *http.Request) {
	// If the user has been authenticated and wanted to re-access this login page, don't allow it
	if ok := yabi.IsUserAuthenticated(w, r, config.MyEncryptDecryptSK); ok {
		http.Redirect(w, r, "/dashboard", 302)
	}

	tmpl := template.Must(template.ParseFiles(config.SiteRootTemplate+"front/login.html", config.SiteHeaderTemplate, config.SiteFooterTemplate))
	data := contextData{
		"PageTitle":    "Login - " + config.SiteShortName,
		"PageMetaDesc": config.SiteShortName + " account, sign in to access your account.",
		"CanonicalURL": r.RequestURI,
		"CsrfToken":    csrf.Token(r),
		"Settings":     config.SiteSettings,
	}
	tmpl.Execute(w, data)
}

// LoginUserEndpoint is to validate the user's login credential
func LoginUserEndpoint(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusOK)

	body, errBody := ioutil.ReadAll(r.Body)
	if errBody != nil {
		itrlog.Error(errBody)
		panic(errBody.Error())
	}

	keyVal := make(map[string]string)
	json.Unmarshal(body, &keyVal)

	userName := strings.TrimSpace(keyVal["username"])
	password := keyVal["password"]
	isSiteKeepMe, _ := strconv.ParseBool(keyVal["isSiteKeepMe"])

	// Open the MySQL DSB Connection
	dbYabi, err := sql.Open("mysql", DBConStr(""))
	if err != nil {
		itrlog.Error(err)
	}
	defer dbYabi.Close()

	// New yabi user information
	userAccount := yabi.User{
		UserName: userName,
		Password: password,
	}

	// Check the user's login credentials
	isAccountValid, err := yabi.LoginUser(dbYabi, userAccount, isSiteKeepMe, config.UserCookieExp)
	if err != nil {
		itrlog.Error(err)
		w.Write([]byte(`{ "IsSuccess": "false", "AlertTitle": "User's Authentication Failed!", 
		"AlertMsg": "` + err.Error() + `", "AlertType": "error", "RedirectURL": "" }`))
		return
	}

	if isAccountValid {
		// Set the cookie expiry in days.
		expDays := "1" // default to expire in 1 day, otherwise if 0, browser will automatically delete the cookie
		if isSiteKeepMe {
			expDays = fmt.Sprint(config.UserCookieExp)
		}

		// Encrypt the username value to store it from the user's cookie.
		encryptedUserName, err := tago.Encrypt(userName, config.MyEncryptDecryptSK)

		if err != nil {
			itrlog.Error("ERROR FROM encryptedUserName: ", err)
			// Failed encrypting the username
			w.Write([]byte(`{ "IsSuccess": "true", "AlertTitle": "Authentication Failed", 
			"AlertMsg": "Oops!, encryption failed, please try again",
			"AlertType": "error", "RedirectURL": "", "EncUserName": "", "UserCookieExpDays": "" }`))
			return
		}
		// Response back to the user about the succcessful user's authentication process
		w.Write([]byte(`{ "IsSuccess": "true", "AlertTitle": "Login is Successful", 
		"AlertMsg": "You've successfully validated your ` + config.SiteShortName + `'s account",
		"AlertType": "success", "RedirectURL": "` + yabi.YB.BaseURL + `dashboard",
		"EncUserName": "` + encryptedUserName + `", "UserCookieExpDays": "` + expDays + `" }`))
	} else {
		// Failed User's Credentials
		w.Write([]byte(`{ "IsSuccess": "true", "AlertTitle": "Authentication Failed", 
		"AlertMsg": "Login failed for user ` + userName + `, please try again",
		"AlertType": "error", "RedirectURL": "", "EncUserName": "", "UserCookieExpDays": "" }`))
	}
}

// RegisterUserEndpoint is to register a new user
func RegisterUserEndpoint(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusOK)

	body, errBody := ioutil.ReadAll(r.Body)
	if errBody != nil {
		itrlog.Error(errBody)
		panic(errBody.Error())
	}

	keyVal := make(map[string]string)
	json.Unmarshal(body, &keyVal)

	userName := strings.TrimSpace(keyVal["username"])
	email := keyVal["email"]
	password := keyVal["password"]
	confirmPassword := keyVal["confirmPassword"]
	tos, _ := strconv.ParseBool(strings.TrimSpace(keyVal["tos"]))
	isActive, _ := strconv.ParseBool(strings.TrimSpace(keyVal["isActive"]))

	// Open the MySQL DSB Connection
	dbYabi, err := sql.Open("mysql", DBConStr(""))
	if err != nil {
		itrlog.Error(err)
	}
	defer dbYabi.Close()

	// New yabi user information
	newUser := yabi.User{
		UserName:    userName,
		Password:    password,
		Email:       email,
		IsSuperUser: false,
		IsAdmin:     false,
		IsActive:    isActive,
	}

	// Email confirmation token
	rt := timaan.RandomToken()
	emailConfirmPayload := timaan.TP{
		"USERNAME": userName,
		"EMAIL":    email,
	}
	tok := timaan.TK{
		TokenKey: rt,
		Payload:  emailConfirmPayload,
		ExpireOn: time.Now().Add(time.Minute * 30).Unix(),
	}
	newToken, err := timaan.GenerateToken(rt, tok)
	if err != nil {
		itrlog.Error(err)
	}
	confirmURL := yabi.YB.BaseURL + "activate/" + fmt.Sprint(newToken)

	// Email confirmation
	newUserEmailConfirmation := yabi.EmailConfig{
		From:                 config.SiteEmail,
		FromAlias:            "Support Team",
		To:                   email,
		Subject:              "Activate your " + config.SiteShortName + " account",
		DefaultTemplate:      yabi.EmailFormatNewUser,
		EmailConfirmationURL: confirmURL,
	}

	// Check if "isActive" is true, then we don't send an email confirmation to activate the new user's account.
	if isActive {
		_, err := yabi.CreateUser(dbYabi, newUser, newUserEmailConfirmation, confirmPassword, tos)
		if err != nil {
			itrlog.Error(err)
			w.Write([]byte(`{ "IsSuccess": "false", "AlertTitle": "New User Creation Failed!", 
			"AlertMsg": "` + err.Error() + `", "AlertType": "error", "RedirectURL": "" }`))
			return
		}

		// Response back to the user about the succcessful user's registration
		w.Write([]byte(`{ "IsSuccess": "true", "AlertTitle": "New User", 
		"AlertMsg": "You've successfully created a new ` + config.SiteShortName + `'s account.",
		"AlertType": "success", "RedirectURL": "` + yabi.YB.BaseURL + `account_activation_sent" }`))
	} else {
		// Insert the new user's registration here
		_, err := yabi.CreateUser(dbYabi, newUser, newUserEmailConfirmation, confirmPassword, tos)
		if err != nil {
			itrlog.Error(err)
			w.Write([]byte(`{ "IsSuccess": "false", "AlertTitle": "New User Creation Failed!", 
			"AlertMsg": "` + err.Error() + `", "AlertType": "error", "RedirectURL": "" }`))
			return
		}

		// Response back to the user about the succcessful user's registration with auto-redirect to a successful page
		w.Write([]byte(`{ "IsSuccess": "true", "AlertTitle": "Registration is Successful", 
		"AlertMsg": "You've successfully created your new ` + config.SiteShortName + `'s account ",
		"AlertType": "success", "RedirectURL": "` + yabi.YB.BaseURL + `account_activation_sent" }`))
	}
}

// DBConStr is the connection string for your database
func DBConStr(dbName string) string {
	db := dbName
	if len(strings.TrimSpace(dbName)) == 0 {
		db = config.DBName
	}
	return fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true&charset=utf8mb4,utf8", config.DBUserName,
		config.DBPassword, config.DBHostName, db)
}

// PageNotFound function is to render the 404 not found page.
func PageNotFound(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles(config.SiteRootTemplate+"front/404.html",
		config.SiteHeaderTemplateCommon, config.SiteFooterAccountTemplateCommon))

	data := contextData{
		"PageTitle":    "Page Not Found (404) | " + config.SiteShortName,
		"PageMetaDesc": "Page Not Found error 404 " + config.SiteShortName,
		"CanonicalURL": r.RequestURI,
		"CsrfToken":    csrf.Token(r),
		"Settings":     config.SiteSettings,
	}
	tmpl.Execute(w, data)
}

// AccountActivationSent function is to render the account activation sent page.
func AccountActivationSent(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles(config.SiteRootTemplate+"front/account_activation_sent.html",
		config.SiteHeaderTemplateCommon, config.SiteFooterAccountTemplateCommon))

	data := contextData{
		"PageTitle":    "Email Activation Sent | " + config.SiteShortName,
		"PageMetaDesc": config.SiteShortName + " new user account activation sent",
		"CanonicalURL": r.RequestURI,
		"CsrfToken":    csrf.Token(r),
		"Settings":     config.SiteSettings,
	}
	tmpl.Execute(w, data)
}

// ActivateAccount is to activate the new user's registration directly from the user's email address
func ActivateAccount(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles(config.SiteRootTemplate+"front/account_activation_complete.html",
		config.SiteHeaderTemplateCommon, config.SiteFooterAccountTemplateCommon))

	params := mux.Vars(r)
	token := params["token"]

	// Extract the timaan payload
	tok, err := timaan.DecodePayload(token)
	if err != nil {
		itrlog.Error(err)
	}

	userName := ""
	payLoad := tok.Payload
	for field, val := range payLoad {
		itrlog.Info("timaan token payload ", field, ": ", val)
		if strings.TrimSpace(field) == "USERNAME" {
			userName = fmt.Sprintf("%v", val) // Get the username value
		}
	}
	fmt.Println("userName: ", userName)
	if len(strings.TrimSpace(userName)) == 0 {
		PageNotFound(w, r)
	}

	// Check the timaan token expiry
	if time.Now().Unix() > tok.ExpireOn {
		itrlog.Warn("timaan token has been expired for username: ", userName, " : ", +time.Now().Unix(), " > ", tok.ExpireOn)
		PageNotFound(w, r)
	}

	// Open the MySQL DB Connection
	dbYabi, err := sql.Open("mysql", DBConStr(""))
	if err != nil {
		itrlog.Error(err)
	}
	defer dbYabi.Close()

	// Now, activate the user's new account which is the "is_active=true" status.
	isActive := yabi.ActivateUser(dbYabi, userName)
	if isActive {
		data := contextData{
			"PageTitle":    "Your new " + config.SiteShortName + " account has been activated",
			"PageMetaDesc": "Your newly created " + config.SiteShortName + " account has been successfully activated",
			"CanonicalURL": r.RequestURI,
			"CsrfToken":    csrf.Token(r),
			"Settings":     config.SiteSettings,
		}
		tmpl.Execute(w, data)
	} else {
		PageNotFound(w, r)
	}
}
