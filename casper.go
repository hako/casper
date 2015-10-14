// Package casper provides methods for interacting with the Casper API.
package casper

import (
	"compress/gzip"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

// Casper constants.
const (
	SnapchatVersion = "9.16.2.0"
	ApplicationID   = "com.snapchat.android"

	CasperSignRequestURL             = "https://api.casper.io/snapchat/clientauth/signrequest"
	CasperAttestationCreateBinaryURL = "https://api.casper.io/snapchat/attestation/create"
	CasperAttestationAttestBinaryURL = "https://api.casper.io/snapchat/attestation/attest"

	GoogleSafteyNetURL    = "https://www.googleapis.com/androidantiabuse/v1/x/create?alt=PROTO&key=AIzaSyBofcZsgLSS7BOnBjZPEkk4rYwzOIz-lTI"
	AttestationCheckerURL = "https://www.googleapis.com/androidcheck/v1/attestations/attest?alt=JSON&key=AIzaSyDqVnJBjE5ymo--oBJt3On7HQx9xNm1RHA"
)

// Casper error variables.
var (
	casperParseError = Error{Err: "casper: CasperParseError"}
	casperHTTPError  = Error{Err: "casper: CasperHTTPError"}
)

// Casper holds credentials to be used when connecting to the Casper API.
type Casper struct {
	APIKey    string
	APISecret string
	Username  string
	Password  string
	Debug     bool
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

// GenerateRequestSignature creates a Casper API request signature.
func (c *Casper) GenerateRequestSignature(params url.Values, endpoint, signature string) string {
	var requestString string
	var k, v []string

	if endpoint == CasperAttestationAttestBinaryURL {
		k = []string{"nonce", "protobuf", "snapchat_version"}
		v = []string{params.Get("nonce"), params.Get("protobuf"), params.Get("snapchat_version")}
	} else {
		k = []string{"password", "snapchat_version", "timestamp", "username"}
		v = []string{params.Get("password"), params.Get("snapchat_version"), params.Get("timestamp"), params.Get("username")}
	}

	for i := 0; i < len(k); i++ {
		requestString += k[i] + v[i]
	}

	byteString := []byte(requestString)
	mac := hmac.New(sha256.New, []byte(signature))
	mac.Write(byteString)
	return "v1:" + hex.EncodeToString(mac.Sum(nil))
}

// GetAttestation fetches a valid Google attestation using the Casper API.
func (c *Casper) GetAttestation(username, password, timestamp string) (string, error) {
	var tr *http.Transport

	tr = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	if c.Debug == true {
		proxyURL, err := url.Parse("http://192.168.2.1:8889")
		if err != nil {
			casperParseError.Reason = err
			return "", casperParseError
		}
		tr.Proxy = http.ProxyURL(proxyURL)
	}

	// 1 - Fetch the device binary.
	client := &http.Client{Transport: tr}

	clientAuthForm := url.Values{}
	clientAuthForm.Add("username", username)
	clientAuthForm.Add("password", password)
	clientAuthForm.Add("timestamp", timestamp)
	clientAuthForm.Add("snapchat_version", SnapchatVersion)

	casperSignature := c.GenerateRequestSignature(clientAuthForm, CasperAttestationCreateBinaryURL, c.APISecret)

	req, err := http.NewRequest("GET", CasperAttestationCreateBinaryURL, nil)
	req.Header.Set("User-Agent", "CasperGoAPIClient/1.1")
	req.Header.Set("X-Casper-API-Key", c.APIKey)
	req.Header.Set("X-Casper-Signature", casperSignature)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Expect", "100-continue")

	res, err := client.Do(req)
	if err != nil {
		casperHTTPError.Reason = err
		return "", casperHTTPError
	} else if res.StatusCode != 200 {
		casperHTTPError.Reason = errors.New("Request returned non 200 code. (" + res.Status + ")")
		return "", casperHTTPError
	}

	parsed, err := ioutil.ReadAll(res.Body)
	if err != nil {
		casperParseError.Reason = err
		return "", casperParseError
	}

	var binaryData map[string]interface{}
	json.Unmarshal(parsed, &binaryData)
	b64binary := binaryData["binary"].(string)

	protobuf, err := base64.StdEncoding.DecodeString(b64binary)
	if err != nil {
		casperParseError.Reason = err
		return "", casperParseError
	}

	// 2 - Send decoded binary as protobuf to Google for validation.
	createBinaryReq, err := http.NewRequest("POST", GoogleSafteyNetURL, strings.NewReader(string(protobuf)))
	createBinaryReq.Header.Set("Accept-Encoding", "gzip")
	createBinaryReq.Header.Set("User-Agent", "DroidGuard/7329000 (A116 _Quad KOT49H); gzip")
	createBinaryReq.Header.Set("Content-Type", "application/x-protobuf")

	createBinaryRes, err := client.Do(createBinaryReq)

	if err != nil {
		casperHTTPError.Reason = err
		return "", casperHTTPError
	} else if createBinaryRes.StatusCode != 200 {
		casperHTTPError.Reason = errors.New("Request returned non 200 code. (" + createBinaryRes.Status + ")")
		return "", casperHTTPError
	}

	createBinaryGzipRes, err := gzip.NewReader(createBinaryRes.Body)
	if err != nil {
		casperParseError.Reason = err
		return "", casperParseError
	}

	var parsedData map[string]interface{}
	protobufData, err := ioutil.ReadAll(createBinaryGzipRes)
	if err != nil {
		casperParseError.Reason = err
		return "", casperParseError
	}

	json.Unmarshal(protobufData, &parsedData)

	// 3 - Send snapchat version, nonce and protobuf data to Casper API in exchange for an attestation request.
	b64protobuf := base64.StdEncoding.EncodeToString(protobufData)
	hash := sha256.New()
	io.WriteString(hash, username+"|"+password+"|"+timestamp+"|"+"/loq/login")
	nonce := base64.StdEncoding.EncodeToString(hash.Sum(nil))

	attestForm := url.Values{}
	attestForm.Add("nonce", nonce)
	attestForm.Add("protobuf", b64protobuf)
	attestForm.Add("snapchat_version", SnapchatVersion)

	casperSignature = c.GenerateRequestSignature(attestForm, CasperAttestationAttestBinaryURL, c.APISecret)
	attestReq, err := http.NewRequest("POST", CasperAttestationAttestBinaryURL, strings.NewReader(attestForm.Encode()))

	attestReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	attestReq.Header.Set("Accept", "*/*")
	attestReq.Header.Set("Expect", "100-continue")
	attestReq.Header.Set("X-Casper-API-Key", c.APIKey)
	attestReq.Header.Set("X-Casper-Signature", casperSignature)
	attestReq.Header.Set("Accept-Encoding", "gzip;q=0,deflate,sdch")
	attestReq.Header.Set("User-Agent", "CasperGoAPIClient/1.1")

	attestRes, err := client.Do(attestReq)
	if err != nil {
		casperHTTPError.Reason = err
		return "", casperHTTPError
	} else if attestRes.StatusCode != 200 {
		casperHTTPError.Reason = errors.New("Request returned non 200 code. (" + attestRes.Status + ")")
		return "", casperHTTPError
	}

	var attestData map[string]interface{}
	attestation, err := ioutil.ReadAll(attestRes.Body)
	json.Unmarshal(attestation, &attestData)

	// 4 - Get the binary value in the response map and send it to Google as protobuf in exchange for a signed attestation.
	_, attestExists := attestData["binary"].(string)
	if attestExists != true {
		casperParseError.Reason = errors.New("Key 'binary' does not exist.")
		return "", casperParseError
	}
	attestDecodedBody, _ := base64.StdEncoding.DecodeString(attestData["binary"].(string))

	attestCheckReq, err := http.NewRequest("POST", AttestationCheckerURL, strings.NewReader(string(attestDecodedBody)))
	attestCheckReq.Header.Set("User-Agent", "SafetyNet/7899000 (WIKO JZO54K); gzip")
	attestCheckReq.Header.Set("Content-Type", "application/x-protobuf")
	attestCheckReq.Header.Set("Content-Length", string(len(attestDecodedBody)))
	attestCheckReq.Header.Set("Connection", "Keep-Alive")
	attestCheckReq.Header.Set("Accept-Encoding", "gzip")

	attestCheckRes, err := client.Do(attestCheckReq)
	if err != nil {
		casperHTTPError.Reason = err
		return "", casperHTTPError
	} else if attestCheckRes.StatusCode != 200 {
		casperHTTPError.Reason = errors.New("Request returned non 200 code. (" + attestCheckRes.Status + ")")
		return "", casperHTTPError
	}

	attestGzipRes, err := gzip.NewReader(attestCheckRes.Body)
	attestDecompressedRes, err := ioutil.ReadAll(attestGzipRes)
	if err != nil {
		casperParseError.Reason = err
		return "", casperParseError
	}

	var attestSignedData map[string]interface{}
	json.Unmarshal(attestDecompressedRes, &attestSignedData)
	_, attestSigExists := attestSignedData["signedAttestation"].(string)
	if attestSigExists != true {
		casperParseError.Reason = errors.New("Key 'signedAttestation' does not exist.")
		return "", casperParseError
	}
	singedAttestation := attestSignedData["signedAttestation"].(string)
	return singedAttestation, nil
}

// GetClientAuthToken fetches a generated client auth token using the Casper API.
func (c *Casper) GetClientAuthToken(username, password, timestamp string) (string, error) {
	var tr *http.Transport

	tr = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	if c.Debug == true {
		proxyURL, err := url.Parse("http://192.168.2.1:8889")
		if err != nil {
			casperParseError.Reason = err
			return "", casperParseError
		}
		tr.Proxy = http.ProxyURL(proxyURL)
	}

	client := &http.Client{Transport: tr}
	clientAuthForm := url.Values{}
	clientAuthForm.Add("username", username)
	clientAuthForm.Add("password", password)
	clientAuthForm.Add("timestamp", timestamp)
	clientAuthForm.Add("snapchat_version", SnapchatVersion)

	casperSignature := c.GenerateRequestSignature(clientAuthForm, CasperSignRequestURL, c.APISecret)
	req, err := http.NewRequest("POST", CasperSignRequestURL, strings.NewReader(string(clientAuthForm.Encode())))
	req.Header.Set("User-Agent", "CasperGoAPIClient/1.1")
	req.Header.Set("X-Casper-API-Key", c.APIKey)
	req.Header.Set("X-Casper-Signature", casperSignature)
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		casperHTTPError.Reason = err
		return "", casperHTTPError
	} else if resp.StatusCode != 200 {
		casperHTTPError.Reason = errors.New("Request returned non 200 code. (" + resp.Status + ")")
		return "", casperHTTPError
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		casperParseError.Reason = err
		return "", casperParseError
	}

	var data map[string]interface{}
	json.Unmarshal(body, &data)
	_, signatureExists := data["signature"].(string)
	if signatureExists != true {
		casperParseError.Reason = errors.New("Key 'signature' does not exist.")
		return "", casperParseError
	}
	signature := data["signature"].(string)
	return signature, nil
}
