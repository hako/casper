package casper

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

type TestCasperCredentials struct {
	TestAPIKey    string
	TestAPISecret string
}

var (
	// Default test API Keys.
	testCasperKeys = TestCasperCredentials{
		TestAPIKey:    "test_api_key",
		TestAPISecret: "test_api_secret",
	}
	testServer      *httptest.Server
	testHTTPhandler http.Handler
)

// Test SignToken.
func TestSignToken(t *testing.T) {
	var testCasperClient = &Casper{
		APIKey:    testCasperKeys.TestAPIKey,
		APISecret: testCasperKeys.TestAPISecret,
	}

	testData := map[string]string{
		"username": "test_api_user",
		"password": "test_api_password",
	}

	var paramTests = []struct {
		testAPIKey  string
		iat         string
		expectation string
	}{
		{"f7b57dffb27e76a928feb948cdb52bf8", "1457484764", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpYXQiOiIxNDU3NDg0NzY0IiwicGFzc3dvcmQiOiJ0ZXN0X2FwaV9wYXNzd29yZCIsInVzZXJuYW1lIjoidGVzdF9hcGlfdXNlciJ9.CNVOGMoGtq7KnTPV88VMjVFAanC9lPBs-YoeFyzb01g"},
		{"6fb6ebd3c6546a21065a41c3bfebc9e9", "1457484825", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpYXQiOiIxNDU3NDg0ODI1IiwicGFzc3dvcmQiOiJ0ZXN0X2FwaV9wYXNzd29yZCIsInVzZXJuYW1lIjoidGVzdF9hcGlfdXNlciJ9.yev0onEnp_dQKBRFnSk7QrLQ0PeKIwBiUftzcqkFtUc"},
		{"a26b3ed36113f1d827fe779722f35fc1", "1457484900", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpYXQiOiIxNDU3NDg0OTAwIiwicGFzc3dvcmQiOiJ0ZXN0X2FwaV9wYXNzd29yZCIsInVzZXJuYW1lIjoidGVzdF9hcGlfdXNlciJ9.PtSGE5mg-ibX-pTvc9bQIB47aBguUJiZ9UC5eaI1Rwc"},
		{"eeabdfd15b6153deb579b3cd658756fc", "1457484956", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpYXQiOiIxNDU3NDg0OTU2IiwicGFzc3dvcmQiOiJ0ZXN0X2FwaV9wYXNzd29yZCIsInVzZXJuYW1lIjoidGVzdF9hcGlfdXNlciJ9.krjC_S1RikmcCV-mktUANGZvD3NmBC4Tv49_2RlQl2M"},
		{"b582bf010cfd8a89b908dbcbb4cb5165", "1457485003", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpYXQiOiIxNDU3NDg1MDAzIiwicGFzc3dvcmQiOiJ0ZXN0X2FwaV9wYXNzd29yZCIsInVzZXJuYW1lIjoidGVzdF9hcGlfdXNlciJ9.i2gt0NR_MjpcjkKskZpqiTCWLkeNRH91zpDea8zVH9c"},
		{"6b9e30736f2c741ee02c431dc1cf68b0", "1457485086", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpYXQiOiIxNDU3NDg1MDg2IiwicGFzc3dvcmQiOiJ0ZXN0X2FwaV9wYXNzd29yZCIsInVzZXJuYW1lIjoidGVzdF9hcGlfdXNlciJ9.dV6BErNhBH_kJAySGgZg4uLrnotYLloOlk6yVetkYYw"},
		{"1631c6811b59db6d0da74ec31178a8f0", "1457485364", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpYXQiOiIxNDU3NDg1MzY0IiwicGFzc3dvcmQiOiJ0ZXN0X2FwaV9wYXNzd29yZCIsInVzZXJuYW1lIjoidGVzdF9hcGlfdXNlciJ9.A41eyWlze16x_Ol8Ndod4o9aEGkVhLdR-z5GnGcc5XQ"},
	}

	for _, test := range paramTests {
		testCasperClient.APISecret = test.testAPIKey
		testData["iat"] = test.iat // "iat" is highjacked for testing purposes only, never do this outside this environment.
		result, err := testCasperClient.signToken(testData)
		if err != nil {
			t.Errorf("signToken(%q) failed test. \n\n\rWant: \n\r\"%s\" \n\rGot: \n\r\"%s\" \n\n", testData, "<nil>", err)
		}
		if result != test.expectation {
			t.Errorf("signToken(%q) failed test. \n\n\rWant: \n\r\"%s\" \n\rGot: \n\r\"%s\" \n\n", testData, test.expectation, result)
		}
	}
}

// Test ParseBody.
func TestParseBody(t *testing.T) {
	formBody := url.Values{}

	testHTTPhandler = http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(400)
		rw.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(rw, `{"code": 400,"message": "JWT Exception: Signature verification failed"}`)
	})

	testServer = httptest.NewServer(testHTTPhandler)
	defer testServer.Close()

	var paramTests = []struct {
		jwt      string
		expected string
	}{
		{"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpYXQiOiIxNDU3NDg0NzY0IiwicGFzc3dvcmQiOiJ0ZXN0X2FwaV9wYXNzd29yZCIsInVzZXJuYW1lIjoidGVzdF9hcGlfdXNlciJ9.CNVOGMoGtq7KnTPV88VMjVFAanC9lPBs-YoeFyzb01g", `{"code": 400,"message": "JWT Exception: Signature verification failed"}`},
	}

	for _, test := range paramTests {
		formBody.Add("jwt", test.jwt)
		res, err := http.Post(testServer.URL, "application/x-www-form-urlencoded", strings.NewReader(formBody.Encode()))
		if err != nil {
			t.Errorf("parseBody for http.NewRequest(\"POST\", %s, nil) failed test. Reason: %q", testServer.URL, err)
		}
		bytes, parserr := parseBody(res)
		if parserr != nil {
			t.Errorf("parseBody(%q) failed test. \n\n\rWant: \n\r\"%s\" \n\rGot: \n\r\"%s\" \n\n", test.jwt, "<nil>", err)
		}
		parsedBody := string(bytes)
		if parsedBody != test.expected {
			t.Errorf("parseBody(%q) failed test. %s != %s", "*net/http.Response", parsedBody, test.expected)
		}
	}
}

// Test Proxy.
func TestProxy(t *testing.T) {
	var testCasperClient = &Casper{
		APIKey:    testCasperKeys.TestAPIKey,
		APISecret: testCasperKeys.TestAPISecret,
	}

	var paramTests = []struct {
		param string
	}{
		{"http://192.168.2.3"},
		{"http://192.168.2.3:8080"},
		{"https://192.168.2.3"},
		{"http://192.168.2.3:8080"},
	}

	for _, test := range paramTests {
		err := testCasperClient.Proxy(test.param)
		if err != nil {
			t.Errorf("Proxy(%q) failed test. \n\n\rWant: \n\r\"%s\" \n\rGot: \n\r\"%s\" \n\n", test.param, "<nil>", err)
		}
	}
}

// Test InvalidProxy.
func TestInvalidProxy(t *testing.T) {
	var testCasperClient = &Casper{
		APIKey:    testCasperKeys.TestAPIKey,
		APISecret: testCasperKeys.TestAPISecret,
	}

	var paramTests = []struct {
		param string
	}{
		{"192.168.2.3"},
		{"192.168.2.3:8080"},
		{"192.168.2::8080"},
		{"192.168:8080"},
		{"http://192.168.0.%31/"},
		{"http://192.168.0.%31:8080/"},
	}

	for _, test := range paramTests {
		err := testCasperClient.Proxy(test.param)
		if err == nil {
			t.Errorf("Proxy(%q) failed test. \n\n\rWant: \n\r\"%s\" \n\rGot: \n\r\"%s\" \n\n", test.param, "<nil>", err)
		}
	}
}
