package api

import (
	"gowebapp/config"
	"gowebapp/yabi"
	"html/template"
	"net/http"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
)

// DashboardRouters are the collection of all URLs for the Dashboard App.
func DashboardRouters(r *mux.Router) {
	r.HandleFunc("/dashboard", yabi.LoginRequired(Dashboard, config.MyEncryptDecryptSK)).Methods("GET")
}

// Dashboard function is to render the user's registration page.
func Dashboard(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles(config.SiteRootTemplate+"dashboard/index.html",
		config.SiteHeaderDashTemplate, config.SiteFooterDashTemplate))

	data := contextData{
		"PageTitle":    "Dashboard - " + config.SiteShortName,
		"PageMetaDesc": config.SiteShortName + " dashboard, managing your user's transactions.",
		"CanonicalURL": r.RequestURI,
		"CsrfToken":    csrf.Token(r),
		"Settings":     config.SiteSettings,
		"Yabi":         yabi.YBUserData,
	}
	tmpl.Execute(w, data)
}
