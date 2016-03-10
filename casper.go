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

// Casper error variables.
var (
	casperParseError      = Error{Err: "casper: CasperParseError"}
	casperHTTPError       = Error{Err: "casper: CasperHTTPError"}
	casperSignatureError  = Error{Err: "casper: CasperSignatureError"}
	casperAuthError       = Error{Err: "casper: casperAuthError"}
	casperDeprecatedError = Error{Err: "casper: casperDeprecatedError"}
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

// casperlogin logs into Casper and returns a SnapchatRequestLoginModel.
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
