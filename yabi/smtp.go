package yabi

import (
	"errors"
	"strings"

	"github.com/itrepablik/sulat"
)

// EmailConfig common yabi email configurations
type EmailConfig struct {
	From, To, CC, BCC                     string // Standard email format e.g support@itrepablik.com
	FromAlias, ToAlias, CCAlias, BCCAlias string // FromAlias is an alias name e.g "Support Team", optional
	Subject                               string // Email subject
	SiteName                              string // Either your company name or site domain name, etc.
	UserName                              string // Username is required for new user email confirmation
	SiteSupportEmail                      string // Email footer or any contact information within your email content
	EmailConfirmationURL                  string // Use for new user activation or password reset
	CustomizeHeaderTemplate               string // Customize HTML Header for your email content
	CustomizeBodyTemplate                 string // Customize HTML Body for your email content
	CustomizeFooterTemplate               string // Customize HTML Footer for your email content
	DefaultTemplate                       string // either "NEW_USER_EMAIL_CONFIRMATION" or "PASSWORD_RESET"
}

// SGC initialize this variable globally sulat.SendGridConfig{}
var SGC = sulat.SGC{}

// SetSendGridAPI is the SendGridAPI Key initialization
func SetSendGridAPI(s sulat.SGC) sulat.SGC {
	if len(strings.TrimSpace(s.SendGridAPIKey)) == 0 {
		return sulat.SGC{}
	}
	if len(strings.TrimSpace(s.SendGridEndPoint)) == 0 {
		s.SendGridEndPoint = "/v3/mail/send"
	}
	if len(strings.TrimSpace(s.SendGridHost)) == 0 {
		s.SendGridHost = "https://api.sendgrid.com"
	}
	// Set the SendGrid API Key required info
	SGC = sulat.SGC{
		SendGridAPIKey:   s.SendGridAPIKey,
		SendGridEndPoint: s.SendGridEndPoint,
		SendGridHost:     s.SendGridHost,
	}
	return SGC
}

func init() {
	SGC = SetSendGridAPI(SGC)
}

// SendEmailSendGrid will now send an email to each individual recipients from the config.yaml
// using SendGrid API Key
func SendEmailSendGrid(e EmailConfig) (bool, error) {
	if len(strings.TrimSpace(e.From)) == 0 {
		return false, errors.New("email from is missing")
	}
	if len(strings.TrimSpace(e.To)) == 0 {
		return false, errors.New("email to is missing")
	}
	if len(strings.TrimSpace(e.DefaultTemplate)) == 0 {
		return false, errors.New("email template is missing")
	}

	// NewUserActivation(confirmURL, userName, siteName, siteSupportEmail string)
	// Send an auto email notification to all the recipients
	newEmailContent := ""
	switch e.DefaultTemplate {
	case EmailFormatNewUser:
		newEmailContent = YabiHTMLHeader + NewUserActivation(e.EmailConfirmationURL, e.UserName,
			e.SiteName, e.SiteSupportEmail) + YabiHTMLFooter
	}

	htmlContent, err := sulat.SetHTML(&sulat.EmailHTMLFormat{
		IsFullHTML: false,
		HTMLHeader: YabiHTMLHeader,
		HTMLBody:   newEmailContent,
		HTMLFooter: YabiHTMLFooter,
	})

	mailOpt := &sulat.SendMail{
		Subject: e.Subject,
		From:    sulat.NewEmail(e.FromAlias, e.From),
		To:      sulat.NewEmail(e.ToAlias, e.To),
		CC:      sulat.NewEmail(e.CCAlias, e.CC),
		BCC:     sulat.NewEmail(e.BCCAlias, e.BCC),
	}

	_, err = sulat.SendEmailSG(mailOpt, htmlContent, &SGC)
	if err != nil {
		return false, err
	}
	return true, nil
}
