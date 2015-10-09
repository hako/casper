package casper

import (
	"net/url"
	"testing"
)

type TestCasperCredentials struct {
	TestAPIKey    string
	TestAPISecret string
}

// Setup test API Keys. Replace with your own to carry out tests.
var (
	testCasperKeys = TestCasperCredentials{
		TestAPIKey:    "test_api_key",
		TestAPISecret: "test_api_secret",
	}
)

// Test Request signature.
func TestGenerateRequestSignature(t *testing.T) {

	var testCasperClient = &Casper{
		APIKey:         testCasperKeys.TestAPIKey,
		APISecret:      testCasperKeys.TestAPISecret,
		Username:       "test_username",
		GoogleMail:     "testgmail@gmail.com",
		GooglePassword: "attesttesttesttesttestpassword",
	}

	testForm := url.Values{}
	testForm.Add("username", testCasperClient.Username)
	testForm.Add("password", testCasperClient.GooglePassword)
	testForm.Add("timestamp", "0123456789")

	var paramTests = []struct {
		params      url.Values
		endpoint    string
		signature   string
		expectation string
	}{
		{testForm, CasperAttestationAttestBinaryURL, testCasperClient.APISecret, "v1:285f5177da4984ef582b1fa665071f83c0400e94019b9be4b3aacaf0037c74ba"},
	}

	for _, test := range paramTests {
		result := testCasperClient.GenerateRequestSignature(test.params, test.endpoint, test.signature)
		if result != test.expectation {
			t.Errorf("GenerateRequestSignature(%q, %q, %q) failed test. \n\n\rWant: \n\r\"%s\" \n\rGot: \n\r\"%s\" \n\n", test.params, test.endpoint, test.signature, test.expectation, result)
		}
	}
}
