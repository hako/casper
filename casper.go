// Package casper provides methods for interacting with the Casper API.
package casper

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
)

// Casper constants.
const (
	CasperBaseURL   = "https://casper-api.herokuapp.com"
	SnapchatBaseURL = "https://app.snapchat.com"
)

// Casper common variables and error variables
var (
	casperParseError      = Error{Err: "casper: CasperParseError"}
	casperHTTPError       = Error{Err: "casper: CasperHTTPError"}
	casperSignatureError  = Error{Err: "casper: CasperSignatureError"}
	casperAuthError       = Error{Err: "casper: CasperAuthError"}
	casperDeprecatedError = Error{Err: "casper: CasperDeprecatedError"}

	captchaID = ""
	status    = 0
)

// Casper holds credentials to be used when connecting to the Casper API.
type Casper struct {
	APIKey      string
	APISecret   string
	Username    string
	Password    string
	AuthToken   string
	Debug       bool
	ProxyURL    *url.URL
	ProjectName string
}

// Snapchat holds the credentials needed to pass on data to Snapchat's Servers.
type Snapchat struct {
	CasperClient *Casper
}

// Captcha holds data about a Snapchat captcha archive.
type Captcha struct {
	ID   string
	Data []byte
}

// Options holds data about parameters of a Casper request.
type Options struct {
	Headers map[string]string
	Params  map[string]string
}

// Client is an interface for methods that wish to communicate with Casper and Snapchat.
type Client interface {
	performRequest(method string, endpoint string, params map[string]string, headers map[string]string) ([]byte, error)
}

// Error handles errors returned by casper methods.
type Error struct {
	Err    string
	Reason error
}

// Error is a function which CasperError satisfies.
// It returns a properly formatted error message when an error occurs.
func (e Error) Error() string {
	return fmt.Sprintf("%s\nReason: %s", e.Err, e.Reason.Error())
}

// performRequest is a template that creates HTTP requests with proxy and debug support.
func (s *Snapchat) performRequest(method string, endpoint string, params map[string]string, headers map[string]string) ([]byte, error) {
	var tr *http.Transport
	var snapchatForm url.Values
	var req *http.Request

	if s.CasperClient.Debug == true {
		fmt.Printf(method+"\t%s\n", SnapchatBaseURL+endpoint)
	}

	tr = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
	}

	if s.CasperClient.ProxyURL != nil {
		tr.Proxy = http.ProxyURL(s.CasperClient.ProxyURL)
		tr.TLSClientConfig.InsecureSkipVerify = true
	}

	client := &http.Client{Transport: tr}

	if params != nil {
		snapchatForm = url.Values{}
		for k, v := range params {
			snapchatForm.Add(k, v)
		}
	}

	if s.CasperClient.Debug == true {
		fmt.Printf("%s\n", snapchatForm)
	}

	if method == "GET" {
		req, _ = http.NewRequest(method, SnapchatBaseURL+endpoint, nil)
	} else {
		req, _ = http.NewRequest(method, SnapchatBaseURL+endpoint, strings.NewReader(snapchatForm.Encode()))
	}

	if headers != nil {
		for k, v := range headers {
			req.Header.Set(k, v)
		}
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if endpoint == "/bq/get_captcha" {
		captchaID = res.Header["Content-Disposition"][0][20:]
	}

	if endpoint == "/ph/logout" || endpoint == "/loq/send" {
		status = res.StatusCode
	}

	parsedData, err := parseBody(res)
	if err != nil {
		return nil, err
	}

	if s.CasperClient.Debug == true {
		fmt.Println(string(parsedData))
	}

	return parsedData, nil
}

// Login performs a login request to Snapchat and returns an Updates model.
func (c *Casper) Login(username string, password string) (Updates, error) {
	model, err := c.login(username, password)
	if err != nil {
		return Updates{}, err
	}
	headers := map[string]string{
		"Accept":                       model.Headers.Accept,
		"Accept-Language":              model.Headers.AcceptLanguage,
		"Accept-Locale":                model.Headers.AcceptLocale,
		"User-Agent":                   model.Headers.UserAgent,
		"X-Snapchat-Client-Auth-Token": model.Headers.XSnapchatClientAuthToken,
		"X-Snapchat-Client-Token":      model.Headers.XSnapchatClientToken,
		"X-Snapchat-UUID":              model.Headers.XSnapchatUUID,
	}
	params := map[string]string{
		"confirm_reactivation": model.Params.ConfirmReactivation,
		"from_deeplink":        model.Params.FromDeeplink,
		"height":               model.Params.Height,
		"nt":                   model.Params.Nt,
		"password":             password,
		"pre_auth_token":       model.Params.PreAuthToken,
		"remember_device":      model.Params.RememberDevice,
		"req_token":            model.Params.ReqToken,
		"screen_height_in":     model.Params.ScreenHeightIn,
		"screen_height_px":     model.Params.ScreenHeightPx,
		"screen_width_in":      model.Params.ScreenWidthIn,
		"screen_width_px":      model.Params.ScreenWidthPx,
		"timestamp":            strconv.FormatInt(model.Params.Timestamp, 10),
		"user_ad_id":           model.Params.UserAdID,
		"username":             username,
		"width":                model.Params.Width,
	}
	s := Snapchat{
		CasperClient: c,
	}
	data, err := s.performRequest("POST", "/loq/login", params, headers)
	if err != nil {
		return Updates{}, err
	}
	// Save only once the user has logged in.
	if c.Username == "" || c.Password == "" {
		c.Username = username
		c.Password = password
	}
	var scdata Updates
	json.Unmarshal(data, &scdata)

	// Save auth token.
	c.AuthToken = scdata.UpdatesResponse.AuthToken
	return scdata, nil
}

// Updates fetches updates from Snapchat and returns an Updates model.
func (c *Casper) Updates() (Updates, error) {
	err := c.checkToken()
	if err != nil {
		return Updates{}, err
	}
	jwtform := map[string]string{
		"username":   c.Username,
		"auth_token": c.AuthToken,
		"endpoint":   "/loq/all_updates",
	}
	token, err := c.signToken(jwtform)
	if err != nil {
		return Updates{}, err
	}
	data, err := c.endpointAuth(token)
	if err != nil {
		return Updates{}, err
	}
	updateEndpoint := data.Endpoints[0]   // update endpoint data
	endpoint := updateEndpoint.Endpoint   // /loq/updates
	headers := c.setSnapchatHeaders(data) // headers
	params := map[string]string{
		"username":  updateEndpoint.Params.Username,
		"req_token": updateEndpoint.Params.ReqToken,
		"timestamp": strconv.FormatInt(updateEndpoint.Params.Timestamp, 10),
	}
	s := Snapchat{
		CasperClient: c,
	}
	scdata, err := s.performRequest("POST", endpoint, params, headers)
	if err != nil {
		return Updates{}, err
	}
	var updateData Updates
	json.Unmarshal(scdata, &updateData)
	return updateData, err
}

// Register registers a account from Snapchat and returns an Register model.
func (c *Casper) Register(username, password, email, birthday string) (Register, error) {
	model, err := c.login(username, password)
	if err != nil {
		return Register{}, err
	}
	headers := map[string]string{
		"Accept":          model.Headers.Accept,
		"Accept-Language": model.Headers.AcceptLanguage,
		"Accept-Locale":   model.Headers.AcceptLocale,
		"User-Agent":      "Snapchat/9.0.0.30 (iPhone5,1; iOS 8.4; gzip)",
	}
	params := map[string]string{
		"email":     email,
		"password":  password,
		"req_token": model.Params.ReqToken,
		"timestamp": strconv.FormatInt(model.Params.Timestamp, 10),
		"username":  username,
		"birthday":  birthday,
	}
	s := Snapchat{
		CasperClient: c,
	}
	data, err := s.performRequest("POST", "/loq/register", params, headers)
	if err != nil {
		return Register{}, err
	}
	var registerData Register
	json.Unmarshal(data, &registerData)

	// Save auth token.
	c.AuthToken = registerData.AuthToken
	c.Password = password
	c.Username = username
	return registerData, nil
}

// GetCaptcha fetches a captcha puzzle from snapchat.
func (c *Casper) GetCaptcha() (Captcha, error) {
	err := c.checkToken()
	if err != nil {
		return Captcha{}, err
	}

	jwtform := map[string]string{
		"username":   c.Username,
		"auth_token": c.AuthToken,
		"endpoint":   "/bq/get_captcha",
	}

	token, err := c.signToken(jwtform)
	if err != nil {
		return Captcha{}, err
	}
	data, err := c.endpointAuth(token)
	if err != nil {
		return Captcha{}, err
	}

	captchaEndpoint := data.Endpoints[0]  // update endpoint data
	endpoint := captchaEndpoint.Endpoint  // /loq/get_captcha
	headers := c.setSnapchatHeaders(data) // headers
	params := map[string]string{
		"username":  captchaEndpoint.Params.Username,
		"req_token": captchaEndpoint.Params.ReqToken,
		"timestamp": strconv.FormatInt(captchaEndpoint.Params.Timestamp, 10),
	}

	s := Snapchat{
		CasperClient: c,
	}

	scdata, err := s.performRequest("POST", endpoint, params, headers)
	if err != nil {
		return Captcha{}, err
	}

	captcha := Captcha{
		ID:   captchaID,
		Data: scdata,
	}

	return captcha, nil
}

// SolveCaptcha solves the captcha puzzle from snapchat.
func (c *Casper) SolveCaptcha(captchaID, solution string) (string, error) {
	err := c.checkToken()
	if err != nil {
		return "", err
	}
	jwtform := map[string]string{
		"username":   c.Username,
		"auth_token": c.AuthToken,
		"endpoint":   "/bq/solve_captcha",
	}
	token, err := c.signToken(jwtform)
	if err != nil {
		return "", err
	}
	data, err := c.endpointAuth(token)
	if err != nil {
		return "", err
	}
	solution = strings.Replace(solution, "\n", "", -1) // Get rid of those pesky newlines.
	solvecaptchaEndpoint := data.Endpoints[0]          // update endpoint data
	endpoint := solvecaptchaEndpoint.Endpoint          // /loq/solve_captcha
	headers := c.setSnapchatHeaders(data)              // headers
	params := map[string]string{
		"captcha_solution": solution,
		"captcha_id":       captchaID,
		"username":         solvecaptchaEndpoint.Params.Username,
		"req_token":        solvecaptchaEndpoint.Params.ReqToken,
		"timestamp":        strconv.FormatInt(solvecaptchaEndpoint.Params.Timestamp, 10),
	}
	s := Snapchat{
		CasperClient: c,
	}
	scdata, err := s.performRequest("POST", endpoint, params, headers)
	if err != nil {
		return "", err
	}
	return string(scdata), err
}

// VerifyPhoneNumber sends a phone number to Snapchat for verification.
func (c *Casper) VerifyPhoneNumber(phoneNumber, countryCode string) ([]byte, error) {
	err := c.checkToken()
	if err != nil {
		return nil, err
	}
	jwtform := map[string]string{
		"username":   c.Username,
		"auth_token": c.AuthToken,
		"endpoint":   "/bq/phone_verify",
	}
	token, err := c.signToken(jwtform)
	if err != nil {
		return nil, err
	}
	data, err := c.endpointAuth(token)
	if err != nil {
		return nil, err
	}
	phoneVerifyEndpoint := data.Endpoints[0] // update endpoint data
	endpoint := phoneVerifyEndpoint.Endpoint // /loq/phone_verify
	headers := c.setSnapchatHeaders(data)    // headers
	params := map[string]string{
		"phoneNumber":      phoneNumber,
		"action":           "updatePhoneNumber",
		"skipConfirmation": "true",
		"countryCode":      countryCode,
		"username":         phoneVerifyEndpoint.Params.Username,
		"req_token":        phoneVerifyEndpoint.Params.ReqToken,
		"timestamp":        strconv.FormatInt(phoneVerifyEndpoint.Params.Timestamp, 10),
	}
	s := Snapchat{
		CasperClient: c,
	}
	scdata, err := s.performRequest("POST", endpoint, params, headers)
	if err != nil {
		return nil, err
	}
	return scdata, err
}

// SendSMSCode sends an SMS code to Snapchat.
func (c *Casper) SendSMSCode(code string) ([]byte, error) {
	err := c.checkToken()
	if err != nil {
		return nil, err
	}
	jwtform := map[string]string{
		"username":   c.Username,
		"auth_token": c.AuthToken,
		"endpoint":   "/bq/phone_verify",
	}
	token, err := c.signToken(jwtform)
	if err != nil {
		return nil, err
	}
	data, err := c.endpointAuth(token)
	if err != nil {
		return nil, err
	}
	code = strings.Replace(code, "\n", "", -1) // Get rid of those pesky newlines.
	phoneVerifyEndpoint := data.Endpoints[0]   // update endpoint data
	endpoint := phoneVerifyEndpoint.Endpoint   // /loq/phone_verify
	headers := c.setSnapchatHeaders(data)      // headers
	params := map[string]string{
		"action":    "verifyPhoneNumber",
		"code":      code,
		"type":      "DEFAULT_TYPE",
		"username":  phoneVerifyEndpoint.Params.Username,
		"req_token": phoneVerifyEndpoint.Params.ReqToken,
		"timestamp": strconv.FormatInt(phoneVerifyEndpoint.Params.Timestamp, 10),
	}
	s := Snapchat{
		CasperClient: c,
	}
	scdata, err := s.performRequest("POST", endpoint, params, headers)
	if err != nil {
		return nil, err
	}
	return scdata, err
}

// IPRouting gets IP Routing URLs.
func (c *Casper) IPRouting() ([]byte, error) {
	err := c.checkToken()
	if err != nil {
		return nil, err
	}
	jwtform := map[string]string{
		"username":   c.Username,
		"auth_token": c.AuthToken,
		"endpoint":   "/bq/ip_routing",
	}
	token, err := c.signToken(jwtform)
	if err != nil {
		return nil, err
	}
	data, err := c.endpointAuth(token)
	if err != nil {
		return nil, err
	}
	IProutingEndpoint := data.Endpoints[0] // update endpoint data
	endpoint := IProutingEndpoint.Endpoint // /bq/ip_routing
	headers := c.setSnapchatHeaders(data)  // headers
	params := map[string]string{
		"userId":             c.Username,
		"currentUrlEntities": "",
		"username":           IProutingEndpoint.Params.Username,
		"req_token":          IProutingEndpoint.Params.ReqToken,
		"timestamp":          strconv.FormatInt(IProutingEndpoint.Params.Timestamp, 10),
	}
	s := Snapchat{
		CasperClient: c,
	}
	scdata, err := s.performRequest("POST", endpoint, params, headers)
	if err != nil {
		return nil, err
	}
	return scdata, err
}

// SuggestedFriends fetches all the Snapchat suggested friends.
func (c *Casper) SuggestedFriends() ([]byte, error) {
	err := c.checkToken()
	if err != nil {
		return nil, err
	}
	jwtform := map[string]string{
		"username":   c.Username,
		"auth_token": c.AuthToken,
		"endpoint":   "/bq/suggest_friend",
	}
	token, err := c.signToken(jwtform)
	if err != nil {
		return nil, err
	}
	data, err := c.endpointAuth(token)
	if err != nil {
		return nil, err
	}
	suggestedFriendsEndpoint := data.Endpoints[0] // update endpoint data
	endpoint := suggestedFriendsEndpoint.Endpoint // /bq/suggest_friend
	headers := c.setSnapchatHeaders(data)         // headers
	params := map[string]string{
		"action":    "list",
		"username":  suggestedFriendsEndpoint.Params.Username,
		"req_token": suggestedFriendsEndpoint.Params.ReqToken,
		"timestamp": strconv.FormatInt(suggestedFriendsEndpoint.Params.Timestamp, 10),
	}
	s := Snapchat{
		CasperClient: c,
	}
	scdata, err := s.performRequest("POST", endpoint, params, headers)
	if err != nil {
		return nil, err
	}
	return scdata, err
}

// LoadLensSchedule fetches the lens schedule for the authenticated account.
func (c *Casper) LoadLensSchedule() ([]byte, error) {
	err := c.checkToken()
	if err != nil {
		return nil, err
	}
	jwtform := map[string]string{
		"username":   c.Username,
		"auth_token": c.AuthToken,
		"endpoint":   "/lens/load_schedule",
	}
	token, err := c.signToken(jwtform)
	if err != nil {
		return nil, err
	}
	data, err := c.endpointAuth(token)
	if err != nil {
		return nil, err
	}
	lensScheduleEndpoint := data.Endpoints[0] // update endpoint data
	endpoint := lensScheduleEndpoint.Endpoint // /lens/load_schedule
	headers := c.setSnapchatHeaders(data)     // headers
	params := map[string]string{
		"username":  lensScheduleEndpoint.Params.Username,
		"req_token": lensScheduleEndpoint.Params.ReqToken,
		"timestamp": strconv.FormatInt(lensScheduleEndpoint.Params.Timestamp, 10),
	}
	s := Snapchat{
		CasperClient: c,
	}
	scdata, err := s.performRequest("POST", endpoint, params, headers)
	if err != nil {
		return nil, err
	}
	return scdata, err
}

// DiscoverChannels fetches Snapchat discover channels.
func (c *Casper) DiscoverChannels() ([]byte, error) {
	var endpoint = "/discover/channel_list?region=US&country=USA&version=1&language=en"
	s := Snapchat{
		CasperClient: c,
	}
	headers := map[string]string{
		"Accept-Language": "en",
		"Accept-Locale":   "en_US",
		"User-Agent":      "Snapchat/9.26.0.1 (iPhone6,1; iOS 9.0; gzip)",
	}
	scdata, err := s.performRequest("GET", endpoint, nil, headers)
	if err != nil {
		return nil, err
	}
	return scdata, err
}

// RegisterUsername registers a username from Snapchat and returns an Updates model.
func (c *Casper) RegisterUsername(username string, email string) (Updates, error) {
	err := c.checkToken()
	if err != nil {
		return Updates{}, err
	}
	jwtform := map[string]string{
		"username":   c.Username,
		"auth_token": c.AuthToken,
		"endpoint":   "/loq/register_username",
	}
	token, err := c.signToken(jwtform)
	if err != nil {
		return Updates{}, err
	}
	endpointData, err := c.endpointAuth(token)
	if err != nil {
		return Updates{}, err
	}
	registerUsernameEndpoint := endpointData.Endpoints[0] // register_username endpoint data
	endpoint := registerUsernameEndpoint.Endpoint         // /loq/register_username
	headers := c.setSnapchatHeaders(endpointData)         // headers
	params := map[string]string{
		"req_token":         registerUsernameEndpoint.Params.ReqToken,
		"timestamp":         strconv.FormatInt(registerUsernameEndpoint.Params.Timestamp, 10),
		"username":          email,
		"selected_username": username,
	}
	s := Snapchat{
		CasperClient: c,
	}
	data, err := s.performRequest("POST", endpoint, params, headers)
	if err != nil {
		return Updates{}, err
	}
	var registerUsernameData Updates
	json.Unmarshal(data, &registerUsernameData)
	return registerUsernameData, nil
}

// DownloadSnapTag fetches the authenticated users Snaptag.
func (c *Casper) DownloadSnapTag(id, format string) ([]byte, error) {
	err := c.checkToken()
	if err != nil {
		return nil, err
	}
	jwtform := map[string]string{
		"username":   c.Username,
		"auth_token": c.AuthToken,
		"endpoint":   "/bq/snaptag_download",
	}
	token, err := c.signToken(jwtform)
	if err != nil {
		return nil, err
	}
	data, err := c.endpointAuth(token)
	if err != nil {
		return nil, err
	}
	downloadSnaptagEndpoint := data.Endpoints[0] // update endpoint data
	endpoint := downloadSnaptagEndpoint.Endpoint // /bq/snaptag_download
	headers := c.setSnapchatHeaders(data)        // headers
	params := map[string]string{
		"type":      format,
		"user_id":   id,
		"username":  downloadSnaptagEndpoint.Params.Username,
		"req_token": downloadSnaptagEndpoint.Params.ReqToken,
		"timestamp": strconv.FormatInt(downloadSnaptagEndpoint.Params.Timestamp, 10),
	}
	s := Snapchat{
		CasperClient: c,
	}
	scdata, err := s.performRequest("POST", endpoint, params, headers)
	if err != nil {
		return nil, err
	}
	return scdata, err
}

// Upload sends media to Snapchat.
// TODO: Implement multipart requests instead of returning Options.
func (c *Casper) Upload() (Options, error) {
	err := c.checkToken()
	if err != nil {
		return Options{}, err
	}
	jwtform := map[string]string{
		"username":   c.Username,
		"auth_token": c.AuthToken,
		"endpoint":   "/ph/upload",
	}
	token, err := c.signToken(jwtform)
	if err != nil {
		return Options{}, err
	}
	data, err := c.endpointAuth(token)
	if err != nil {
		return Options{}, err
	}
	uploadEndpoint := data.Endpoints[0]   // upload endpoint data
	headers := c.setSnapchatHeaders(data) // headers
	params := map[string]string{
		"username":  uploadEndpoint.Params.Username,
		"req_token": uploadEndpoint.Params.ReqToken,
		"timestamp": strconv.FormatInt(uploadEndpoint.Params.Timestamp, 10),
	}
	options := Options{
		headers,
		params,
	}
	return options, err
}

// Send sends media to other Snapchat users.
func (c *Casper) Send(mediaID string, recipients []string, time int) ([]byte, error) {
	err := c.checkToken()
	if err != nil {
		return nil, err
	}
	rp, rperr := json.Marshal(recipients)
	if rperr != nil {
		return nil, rperr
	}
	timeString := strconv.Itoa(int(time))
	jwtform := map[string]string{
		"username":   c.Username,
		"auth_token": c.AuthToken,
		"endpoint":   "/loq/send",
	}
	token, err := c.signToken(jwtform)
	if err != nil {
		return nil, err
	}
	data, err := c.endpointAuth(token)
	if err != nil {
		return nil, err
	}
	sendEndpoint := data.Endpoints[0]     // update endpoint data
	endpoint := sendEndpoint.Endpoint     // /loq/send
	headers := c.setSnapchatHeaders(data) // headers
	params := map[string]string{
		"username":            sendEndpoint.Params.Username,
		"req_token":           sendEndpoint.Params.ReqToken,
		"timestamp":           strconv.FormatInt(sendEndpoint.Params.Timestamp, 10),
		"media_id":            mediaID,
		"recipients":          string(rp),
		"reply":               "0",
		"time":                timeString,
		"country_code":        "US",
		"camera_front_facing": "0",
		"zipped":              "0",
	}
	s := Snapchat{
		CasperClient: c,
	}
	scdata, err := s.performRequest("POST", endpoint, params, headers)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, errors.New("snapchat: Something went wrong")
	}
	return scdata, err
}

// RetrySend retries to resend media to Snapchat users.
// TODO: Implement multipart requests instead of returning Options.
func (c *Casper) RetrySend() (Options, error) {
	err := c.checkToken()
	if err != nil {
		return Options{}, err
	}
	jwtform := map[string]string{
		"username":   c.Username,
		"auth_token": c.AuthToken,
		"endpoint":   "/loq/retry",
	}
	token, err := c.signToken(jwtform)
	if err != nil {
		return Options{}, err
	}
	data, err := c.endpointAuth(token)
	if err != nil {
		return Options{}, err
	}
	uploadEndpoint := data.Endpoints[0]   // upload endpoint data
	headers := c.setSnapchatHeaders(data) // headers
	params := map[string]string{
		"username":  uploadEndpoint.Params.Username,
		"req_token": uploadEndpoint.Params.ReqToken,
		"timestamp": strconv.FormatInt(uploadEndpoint.Params.Timestamp, 10),
	}
	options := Options{
		headers,
		params,
	}
	return options, err
}

// Stories fetches the current users Snapchat stories.
// Useful if you only want the Snapchat stories. [Not working as of now]
func (c *Casper) Stories() (Stories, error) {
	err := c.checkToken()
	if err != nil {
		return Stories{}, err
	}
	jwtform := map[string]string{
		"username":   c.Username,
		"auth_token": c.AuthToken,
		"endpoint":   "/bq/stories",
	}
	token, err := c.signToken(jwtform)
	if err != nil {
		return Stories{}, err
	}
	data, err := c.endpointAuth(token)
	if err != nil {
		return Stories{}, err
	}
	storiesEndpoint := data.Endpoints[1]  // upload endpoint data
	endpoint := storiesEndpoint.Endpoint  // /bq/stories
	headers := c.setSnapchatHeaders(data) // headers
	params := map[string]string{
		"username":  storiesEndpoint.Params.Username,
		"req_token": storiesEndpoint.Params.ReqToken,
		"timestamp": strconv.FormatInt(storiesEndpoint.Params.Timestamp, 10),
	}
	s := Snapchat{
		CasperClient: c,
	}
	scdata, err := s.performRequest("POST", endpoint, params, headers)
	if err != nil {
		return Stories{}, err
	}
	var storiesData Stories
	json.Unmarshal(scdata, &storiesData)
	return storiesData, err
}

// RetryPostStory retries to resend media to Snapchat users.
// TODO: Implement multipart requests instead of returning Options.
func (c *Casper) RetryPostStory() (Options, error) {
	err := c.checkToken()
	if err != nil {
		return Options{}, err
	}
	jwtform := map[string]string{
		"username":   c.Username,
		"auth_token": c.AuthToken,
		"endpoint":   "/loq/retry_post_story",
	}
	token, err := c.signToken(jwtform)
	if err != nil {
		return Options{}, err
	}
	data, err := c.endpointAuth(token)
	if err != nil {
		return Options{}, err
	}
	uploadEndpoint := data.Endpoints[0]   // upload endpoint data
	headers := c.setSnapchatHeaders(data) // headers
	params := map[string]string{
		"username":  uploadEndpoint.Params.Username,
		"req_token": uploadEndpoint.Params.ReqToken,
		"timestamp": strconv.FormatInt(uploadEndpoint.Params.Timestamp, 10),
	}
	options := Options{
		headers,
		params,
	}
	return options, err
}

// Logout logs the current use out of Snapchat.
func (c *Casper) Logout() (bool, error) {
	err := c.checkToken()
	if err != nil {
		return false, err
	}
	jwtform := map[string]string{
		"username":   c.Username,
		"auth_token": c.AuthToken,
		"endpoint":   "/ph/logout",
	}
	token, err := c.signToken(jwtform)
	if err != nil {
		return false, err
	}
	endpointData, err := c.endpointAuth(token)
	if err != nil {
		return false, err
	}
	logoutEndpoint := endpointData.Endpoints[0]   // logout endpoint data
	endpoint := logoutEndpoint.Endpoint           // /loq/logout
	headers := c.setSnapchatHeaders(endpointData) // update endpoint data
	params := map[string]string{
		"req_token": logoutEndpoint.Params.ReqToken,
		"timestamp": strconv.FormatInt(logoutEndpoint.Params.Timestamp, 10),
		"username":  c.Username,
	}
	s := Snapchat{
		CasperClient: c,
	}
	_, err = s.performRequest("POST", endpoint, params, headers)
	if err != nil {
		return false, err
	}
	if status != 200 {
		return false, errors.New("snapchat: Something went wrong")
	}
	return true, nil
}

// Proxy sets given string addr, as a proxy addr. Primarily for debugging purposes.
func (c *Casper) Proxy(addr string) error {
	proxyURL, err := url.Parse(addr)
	if err != nil {
		casperParseError.Reason = err
		return casperParseError
	}
	if proxyURL.Scheme == "" {
		return errors.New("invalid proxy url")
	}
	c.ProxyURL = proxyURL
	return nil
}

// login logs into Casper and returns a SnapchatRequestLoginModel.
func (c *Casper) login(username string, password string) (SnapchatRequestLoginModel, error) {
	jwtform := map[string]string{
		"username": username,
		"password": password,
	}
	token, err := c.signToken(jwtform)
	if err != nil {
		return SnapchatRequestLoginModel{}, err
	}
	data, err := c.performRequest("POST", "/snapchat/ios/login", map[string]string{"jwt": token}, nil)
	if err != nil {
		return SnapchatRequestLoginModel{}, err
	}
	var model SnapchatRequestLoginModel
	json.Unmarshal(data, &model)
	return model, nil
}

// endpointAuth handles requests and responses to mutiple snapchat endpoints.
func (c *Casper) endpointAuth(token string) (SnapchatRequestModel, error) {
	data, err := c.performRequest("POST", "/snapchat/ios/endpointauth", map[string]string{"jwt": token}, nil)
	if err != nil {
		return SnapchatRequestModel{}, err
	}
	var scdata SnapchatRequestModel
	json.Unmarshal(data, &scdata)
	return scdata, nil
}

// signToken produces a JWT token signed with HS256. (HMAC-SHA256)
func (c *Casper) signToken(params map[string]string) (string, error) {
	t := time.Now().Unix()
	token := jwt.New(jwt.SigningMethodHS256)
	token.Claims["iat"] = t
	for k, v := range params {
		token.Claims[k] = v
	}
	jwtString, err := token.SignedString([]byte(c.APISecret))
	if err != nil {
		return "", err
	}
	return jwtString, nil
}

// setSnapchatHeaders converts the SnapchatRequestModel csrm to a map[string][string]
// much more easier to add to request headers.
func (c *Casper) setSnapchatHeaders(csrm SnapchatRequestModel) map[string]string {
	scEndpoint := csrm.Endpoints[0]
	headers := map[string]string{
		"Accept":                       scEndpoint.Headers.Accept,
		"User-Agent":                   scEndpoint.Headers.UserAgent,
		"X-Snapchat-Client-Auth-Token": scEndpoint.Headers.XSnapchatClientAuthToken,
		"X-Snapchat-UUID":              scEndpoint.Headers.XSnapchatUUID,
	}
	return headers
}

// performRequest is a template that creates HTTP requests with proxy and debug support.
func (c *Casper) performRequest(method string, endpoint string, params map[string]string, headers map[string]string) ([]byte, error) {
	var tr *http.Transport
	var casperForm url.Values
	var req *http.Request

	if c.Debug == true {
		fmt.Printf(method+"\t%s\n", CasperBaseURL+endpoint)
	}

	tr = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
	}

	if c.ProxyURL != nil {
		tr.Proxy = http.ProxyURL(c.ProxyURL)
		tr.TLSClientConfig.InsecureSkipVerify = true
	}

	client := &http.Client{Transport: tr}

	if params != nil {
		casperForm = url.Values{}
		for k, v := range params {
			casperForm.Add(k, v)
		}
	}

	if c.Debug == true {
		fmt.Printf("%s\n", casperForm)
	}

	if method == "GET" {
		req, _ = http.NewRequest(method, CasperBaseURL+endpoint, nil)
	} else {
		req, _ = http.NewRequest(method, CasperBaseURL+endpoint, strings.NewReader(casperForm.Encode()))
	}

	if c.ProjectName != "" {
		c.ProjectName = c.ProjectName + " "
	}

	req.Header.Set("User-Agent", "CasperGoAPIClient/"+c.ProjectName+"1.1")
	req.Header.Set("X-Casper-API-Key", c.APIKey)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := client.Do(req)
	if err != nil {
		casperHTTPError.Reason = err
		return nil, casperHTTPError
	}
	defer res.Body.Close()

	parsedData, err := parseBody(res)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		var model APIErrorResponseModel
		json.Unmarshal(parsedData, &model)
		casperHTTPError.Reason = errors.New(model.Message + "  (" + res.Status + ")")
		return nil, casperHTTPError
	}

	if c.Debug == true {
		fmt.Println(string(parsedData))
	}
	return parsedData, nil
}

// parseBody is a helper function that parses the *http.Response body res to bytes.
func parseBody(res *http.Response) ([]byte, error) {
	parsedBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		casperParseError.Reason = err
		return nil, casperParseError
	}
	return parsedBody, nil
}

// checkToken checks if a Snapchat authtoken exists.
func (c *Casper) checkToken() error {
	if c.AuthToken == "" || c.Username == "" {
		casperAuthError.Reason = errors.New("auth token or username does not exist")
		return casperAuthError
	}
	return nil
}

// Casper Structs

// SnapchatRequestLoginModel is a struct containing the endpoint headers parameters specifically for login.
type SnapchatRequestLoginModel struct {
	Code    int `json:"code"`
	Headers struct {
		Accept                   string `json:"Accept"`
		AcceptLanguage           string `json:"Accept-Language"`
		AcceptLocale             string `json:"Accept-Locale"`
		UserAgent                string `json:"User-Agent"`
		XSnapchatClientAuthToken string `json:"X-Snapchat-Client-Auth-Token"`
		XSnapchatClientToken     string `json:"X-Snapchat-Client-Token"`
		XSnapchatUUID            string `json:"X-Snapchat-UUID"`
	} `json:"headers"`
	Params struct {
		ConfirmReactivation string `json:"confirm_reactivation"`
		FromDeeplink        string `json:"from_deeplink"`
		Height              string `json:"height"`
		Nt                  string `json:"nt"`
		Password            string `json:"password"`
		PreAuthToken        string `json:"pre_auth_token"`
		RememberDevice      string `json:"remember_device"`
		ReqToken            string `json:"req_token"`
		ScreenHeightIn      string `json:"screen_height_in"`
		ScreenHeightPx      string `json:"screen_height_px"`
		ScreenWidthIn       string `json:"screen_width_in"`
		ScreenWidthPx       string `json:"screen_width_px"`
		Timestamp           int64  `json:"timestamp"`
		UserAdID            string `json:"user_ad_id"`
		Username            string `json:"username"`
		Width               string `json:"width"`
	} `json:"params"`
	Settings struct {
		ForceClearHeaders bool `json:"force_clear_headers"`
		ForceClearParams  bool `json:"force_clear_params"`
	} `json:"settings"`
	URL string `json:"url"`
}

// SnapchatRequestModel is a generic struct containing the endpoint headers and parameters for any Snapchat endpoint.
type SnapchatRequestModel struct {
	Code      int `json:"code"`
	Endpoints []struct {
		CacheMillis int    `json:"cache_millis"`
		Endpoint    string `json:"endpoint"`
		Headers     struct {
			Accept                   string `json:"Accept"`
			UserAgent                string `json:"User-Agent"`
			XSnapchatClientAuthToken string `json:"X-Snapchat-Client-Auth-Token"`
			XSnapchatUUID            string `json:"X-Snapchat-UUID"`
		} `json:"headers"`
		Params struct {
			Username  string `json:"username"`
			ReqToken  string `json:"req_token"`
			Timestamp int64  `json:"timestamp"`
		} `json:"params"`
	} `json:"endpoints"`
	Settings struct {
		ForceExpireCached bool `json:"force_expire_cached"`
	} `json:"settings"`
}

// APIErrorResponseModel is a struct containing just a HTTP status and a message specifying an error occured.
type APIErrorResponseModel struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// GetAttestation fetches a valid Google attestation using the Casper API.
// [DEPRECATED]
func (c *Casper) GetAttestation(username, password, timestamp string) (string, error) {
	casperDeprecatedError.Reason = errors.New("func (*Casper) GetAttestation is deprecated and will not work.\nPlease refrain from using this method")
	return "", casperDeprecatedError
}

// GetClientAuthToken fetches a generated client auth token using the Casper API.
// [DEPRECATED]
func (c *Casper) GetClientAuthToken(username, password, timestamp string) (string, error) {
	casperDeprecatedError.Reason = errors.New("func (*Casper) GetClientAuthToken is deprecated and will not work.\nPlease refrain from using this method")
	return "", casperDeprecatedError
}
