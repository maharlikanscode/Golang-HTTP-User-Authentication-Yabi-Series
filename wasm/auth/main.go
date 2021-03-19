package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"syscall/js"
	"time"
)

// Server mode indicator, make it true to switch to production server settings
var isProdServerMode bool = false // true

// default localhost server ip with port number, include http://127.0.0.1:8081 without trailing '/' slash
var serverIP string = "http://127.0.0.1:8081"

func login(this js.Value, args []js.Value) interface{} {
	jsDoc := js.Global().Get("document")
	if !jsDoc.Truthy() {
		return js.Global().Call("eval", `Swal.fire("Oops!, Error", "Unable to get document objects", "error");`)
	}
	username := jsDoc.Call("getElementById", "username")
	if !username.Truthy() {
		return js.Global().Call("eval", `Swal.fire("Username is Required!", "Unable to get username object", "error");`)
	}
	password := jsDoc.Call("getElementById", "password")
	if !password.Truthy() {
		return js.Global().Call("eval", `Swal.fire("Password is Required!", "Unable to get password object", "error");`)
	}
	isSiteKeepMe := jsDoc.Call("getElementById", "isSiteKeepMe")
	if !isSiteKeepMe.Truthy() {
		return js.Global().Call("eval", `Swal.fire("Oops!, Error", "Unable to get remember option", "error");`)
	}

	// Get the CSRF Token value from the client side
	csrfToken := jsDoc.Call("getElementById", "csrfToken")
	if !csrfToken.Truthy() {
		return js.Global().Call("eval", `Swal.fire("Oops!", "Unable to get the CSRF token", "error");`)
	}
	var pCSRFToken string = csrfToken.Get("value").String()

	var pUserName string = username.Get("value").String()
	var pPassword string = password.Get("value").String()
	var pSiteKeepMe bool = isSiteKeepMe.Get("checked").Bool()

	// Compose the JSON post payload to teh API endpoint.
	payLoad := map[string]interface{}{
		"username":     pUserName,
		"password":     pPassword,
		"isSiteKeepMe": fmt.Sprint(pSiteKeepMe),
	}

	bytesRepresentation, err := json.Marshal(payLoad)
	if err != nil {
		return js.Global().Call("eval", `Swal.fire("Oops! Error", "Something went wrong with your user's registration", "error");`)
	}

	// HTTP new request
	siteHost := serverIP + "/api/v1/user/login"
	client := &http.Client{}

	req, err := http.NewRequest("POST", fmt.Sprintf(siteHost), bytes.NewBuffer(bytesRepresentation))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-TOKEN", pCSRFToken)
	if err != nil {
		return js.Global().Call("eval", `Swal.fire("Oops! Error", "Something went wrong with your connection, please try again", "error");`)
	}

	var isSuccess bool = false

	// Get the response from the http.NewRequest post method from a channel
	c1 := make(chan map[string]interface{}, 1)
	var result2 map[string]interface{}
	go func() {
		resp, _ := client.Do(req)
		defer resp.Body.Close()
		json.NewDecoder(resp.Body).Decode(&result2)
		c1 <- result2 // send the response data to our channel name 'c'
	}()

	var result map[string]interface{}
	go func() interface{} {
		for {
			timeout := make(chan bool, 1)
			go func() {
				time.Sleep(time.Second * 1)
				timeout <- true
			}()

			select {
			case result = <-c1:
				i := result["IsSuccess"] // You must return with JSON format value of either 'true' or 'false' only.
				mStatus := fmt.Sprint(i)
				isSuccess, _ = strconv.ParseBool(mStatus)
				alertTitle := fmt.Sprint(result["AlertTitle"])
				alertMsg := fmt.Sprint(result["AlertMsg"])
				alertType := fmt.Sprint(result["AlertType"])
				redirectURL := fmt.Sprint(result["RedirectURL"])
				encUserName := fmt.Sprint(result["EncUserName"])
				userCookieExpDays := fmt.Sprint(result["UserCookieExpDays"])

				msg := ""
				if !isSuccess {
					msg = `Swal.fire("` + alertTitle + `", "` + alertMsg + `", "` + alertType + `");` // error
				} else {
					msg = `Swal.fire("` + alertTitle + `", "` + alertMsg + `", "` + alertType + `");` // success
				}
				redirectTO := ""
				if len(strings.TrimSpace(redirectURL)) > 0 {
					redirectTO = `var expDaysInt = parseInt(` + userCookieExpDays + `, 10);
					Cookies.set("yabi", "` + encUserName + `", { expires: expDaysInt, path: '' });
					window.location.replace("` + redirectURL + `");`
				}

				return APIResponse(isSuccess, msg, redirectTO)
				break
			case <-timeout:

			}
		}
	}()

	return nil
}

func register(this js.Value, args []js.Value) interface{} {
	jsDoc := js.Global().Get("document")
	if !jsDoc.Truthy() {
		return js.Global().Call("eval", `Swal.fire("Oops!, Error", "Unable to get document objects", "error");`)
	}
	username := jsDoc.Call("getElementById", "username")
	if !username.Truthy() {
		return js.Global().Call("eval", `Swal.fire("Username is Required!", "Unable to get username object", "error");`)
	}
	email := jsDoc.Call("getElementById", "email")
	if !email.Truthy() {
		return js.Global().Call("eval", `Swal.fire("Email is Required!", "Unable to get email object", "error");`)
	}
	password := jsDoc.Call("getElementById", "password")
	if !password.Truthy() {
		return js.Global().Call("eval", `Swal.fire("Password is Required!", "Unable to get password object", "error");`)
	}
	confirmPassword := jsDoc.Call("getElementById", "confirmPassword")
	if !confirmPassword.Truthy() {
		return js.Global().Call("eval", `Swal.fire("Confirm Password is Required!", "Unable to get confirm password object", "error");`)
	}
	chkTOS := jsDoc.Call("getElementById", "chkTOS")
	if !chkTOS.Truthy() {
		return js.Global().Call("eval", `Swal.fire("Oops!, Error", "Unable to get terms of service object", "error");`)
	}

	// Get the CSRF Token value from the client side
	csrfToken := jsDoc.Call("getElementById", "csrfToken")
	if !csrfToken.Truthy() {
		return js.Global().Call("eval", `Swal.fire("Oops!", "Unable to get the CSRF token", "error");`)
	}
	var pCSRFToken string = csrfToken.Get("value").String()

	var pUserName string = username.Get("value").String()
	var pEmail string = email.Get("value").String()
	var pPassword string = password.Get("value").String()
	var pConfirmPassword string = confirmPassword.Get("value").String()
	var pTOS bool = chkTOS.Get("checked").Bool()

	// Compose the JSON post payload to teh API endpoint.
	payLoad := map[string]interface{}{
		"username":        pUserName,
		"email":           pEmail,
		"password":        pPassword,
		"confirmPassword": pConfirmPassword,
		"tos":             fmt.Sprint(pTOS),
		"isActive":        "false",
	}

	bytesRepresentation, err := json.Marshal(payLoad)
	if err != nil {
		return js.Global().Call("eval", `Swal.fire("Oops! Error", "Something went wrong with your user's registration", "error");`)
	}

	// HTTP new request
	siteHost := serverIP + "/api/v1/user/register"
	client := &http.Client{}

	req, err := http.NewRequest("POST", fmt.Sprintf(siteHost), bytes.NewBuffer(bytesRepresentation))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-TOKEN", pCSRFToken)
	if err != nil {
		return js.Global().Call("eval", `Swal.fire("Oops! Error", "Something went wrong with your connection, please try again", "error");`)
	}

	var isSuccess bool = false

	// Get the response from the http.NewRequest post method from a channel
	c1 := make(chan map[string]interface{}, 1)
	var result2 map[string]interface{}
	go func() {
		resp, _ := client.Do(req)
		defer resp.Body.Close()
		json.NewDecoder(resp.Body).Decode(&result2)
		c1 <- result2 // send the response data to our channel name 'c'
	}()

	var result map[string]interface{}
	go func() interface{} {
		for {
			timeout := make(chan bool, 1)
			go func() {
				time.Sleep(time.Second * 1)
				timeout <- true
			}()

			select {
			case result = <-c1:
				i := result["IsSuccess"] // You must return with JSON format value of either 'true' or 'false' only.
				mStatus := fmt.Sprint(i)
				isSuccess, _ = strconv.ParseBool(mStatus)
				alertTitle := fmt.Sprint(result["AlertTitle"])
				alertMsg := fmt.Sprint(result["AlertMsg"])
				alertType := fmt.Sprint(result["AlertType"])
				redirectURL := fmt.Sprint(result["RedirectURL"])

				msg := ""
				if !isSuccess {
					msg = `Swal.fire("` + alertTitle + `", "` + alertMsg + `", "` + alertType + `");` // error
				} else {
					msg = `Swal.fire("` + alertTitle + `", "` + alertMsg + `", "` + alertType + `");` // success
				}
				redirectTO := ""
				if len(strings.TrimSpace(redirectURL)) > 0 {
					redirectTO = `window.location.replace("` + redirectURL + `");`
				}

				return APIResponse(isSuccess, msg, redirectTO)
				break
			case <-timeout:

			}
		}
	}()
	return nil
}

func passwordReset(this js.Value, args []js.Value) interface{} {
	jsDoc := js.Global().Get("document")
	if !jsDoc.Truthy() {
		return js.Global().Call("eval", `Swal.fire("Oops!, Error", "Unable to get document objects", "error");`)
	}
	email := jsDoc.Call("getElementById", "email")
	if !email.Truthy() {
		return js.Global().Call("eval", `Swal.fire("Email is Required!", "Unable to get email object", "error");`)
	}

	// Get the CSRF Token value from the client side
	csrfToken := jsDoc.Call("getElementById", "csrfToken")
	if !csrfToken.Truthy() {
		return js.Global().Call("eval", `Swal.fire("Oops!", "Unable to get the CSRF token", "error");`)
	}
	var pCSRFToken string = csrfToken.Get("value").String()

	var pEmail string = email.Get("value").String()

	// Compose the JSON post payload to teh API endpoint.
	payLoad := map[string]interface{}{
		"email": pEmail,
	}

	bytesRepresentation, err := json.Marshal(payLoad)
	if err != nil {
		return js.Global().Call("eval", `Swal.fire("Oops! Error", "Something went wrong during password reset process", "error");`)
	}

	// HTTP new request
	siteHost := serverIP + "/api/v1/user/password_reset"
	client := &http.Client{}

	req, err := http.NewRequest("POST", fmt.Sprintf(siteHost), bytes.NewBuffer(bytesRepresentation))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-TOKEN", pCSRFToken)
	if err != nil {
		return js.Global().Call("eval", `Swal.fire("Oops! Error", "Something went wrong with your connection, please try again", "error");`)
	}

	var isSuccess bool = false

	// Get the response from the http.NewRequest post method from a channel
	c1 := make(chan map[string]interface{}, 1)
	var result2 map[string]interface{}
	go func() {
		resp, _ := client.Do(req)
		defer resp.Body.Close()
		json.NewDecoder(resp.Body).Decode(&result2)
		c1 <- result2 // send the response data to our channel name 'c'
	}()

	var result map[string]interface{}
	go func() interface{} {
		for {
			timeout := make(chan bool, 1)
			go func() {
				time.Sleep(time.Second * 1)
				timeout <- true
			}()

			select {
			case result = <-c1:
				i := result["IsSuccess"] // You must return with JSON format value of either 'true' or 'false' only.
				mStatus := fmt.Sprint(i)
				isSuccess, _ = strconv.ParseBool(mStatus)
				alertTitle := fmt.Sprint(result["AlertTitle"])
				alertMsg := fmt.Sprint(result["AlertMsg"])
				alertType := fmt.Sprint(result["AlertType"])
				redirectURL := fmt.Sprint(result["RedirectURL"])

				msg := ""
				if !isSuccess {
					msg = `Swal.fire("` + alertTitle + `", "` + alertMsg + `", "` + alertType + `");` // error
				} else {
					msg = `Swal.fire("` + alertTitle + `", "` + alertMsg + `", "` + alertType + `");` // success
				}
				redirectTO := ""
				if len(strings.TrimSpace(redirectURL)) > 0 {
					redirectTO = `window.location.replace("` + redirectURL + `");`
				}

				return APIResponse(isSuccess, msg, redirectTO)
				break
			case <-timeout:

			}
		}
	}()
	return nil
}

// APIResponse is a generic swal response back to the JS client side
func APIResponse(isSuccess bool, msg, redirectURL string) interface{} {
	if !isSuccess {
		if len(strings.TrimSpace(redirectURL)) == 0 {
			return js.Global().Call("eval", msg)
		}
	}
	return js.Global().Call("eval", redirectURL) // redirect to email activation sent page
}

func exposeGoFuncJS() {
	// Exposing the following Go Functions to Javascript client side
	js.Global().Set("login", js.FuncOf(login))
	js.Global().Set("register", js.FuncOf(register))
	js.Global().Set("passwordReset", js.FuncOf(passwordReset))
}

func main() {
	fmt.Println("Welcome to Maharlikans WASM tutorials")

	// This will be overwritten when the isProdServerMode = true
	if isProdServerMode {
		serverIP = "https://your_production_server_id:portnumber"
	}

	c := make(chan bool, 1)

	// Start exposing this following Go functions to JS
	exposeGoFuncJS()
	<-c
}
