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
		APIKey:    testCasperKeys.TestAPIKey,
		APISecret: testCasperKeys.TestAPISecret,
		Username:  "test_username",
		Password:  "attesttesttesttesttestpassword",
	}

	testFormAttestation := url.Values{}
	testFormAttestation.Add("username", testCasperClient.Username)
	testFormAttestation.Add("password", testCasperClient.Password)
	testFormAttestation.Add("snapchat_version", "9.16.2.0")
	testFormAttestation.Add("timestamp", "0123456789")

	testFormClientAuth := url.Values{}
	testFormClientAuth.Add("nonce", "nonceBase64Data")
	testFormClientAuth.Add("protobuf", "protoBase64Data")
	testFormClientAuth.Add("snapchat_version", "9.16.2.0")

	var paramTests = []struct {
		params      url.Values
		signature   string
		expectation string
	}{
		{testFormAttestation, testCasperClient.APISecret, "v1:924f4b500afc0b4eace1a32477256ebcfa0668dd7ecd46632f922f9fd7d406ba"},
		{testFormClientAuth, testCasperClient.APISecret, "v1:25b7aa08787816613c1099aa2323427b142eb95fac9e591a3c4003c4b56b6c81"},
	}

	for _, test := range paramTests {
		result := testCasperClient.GenerateRequestSignature(test.params, test.signature)
		if result != test.expectation {
			t.Errorf("GenerateRequestSignature(%q, %q) failed test. \n\n\rWant: \n\r\"%s\" \n\rGot: \n\r\"%s\" \n\n", test.params, test.signature, test.expectation, result)
		}
	}
}

// Test SortURLMap.
func TestSortURLMap(t *testing.T) {

	var testCasperClient = &Casper{
		APIKey:    testCasperKeys.TestAPIKey,
		APISecret: testCasperKeys.TestAPISecret,
		Username:  "test_username",
		Password:  "attesttesttesttesttestpassword",
	}

	testFormAttestation := url.Values{}
	testFormAttestation.Add("username", testCasperClient.Username)
	testFormAttestation.Add("password", testCasperClient.Password)
	testFormAttestation.Add("snapchat_version", "9.16.2.0")
	testFormAttestation.Add("timestamp", "0123456789")

	testFormClientAuth := url.Values{}
	testFormClientAuth.Add("nonce", "nonceBase64Data")
	testFormClientAuth.Add("protobuf", "protoBase64Data")
	testFormClientAuth.Add("snapchat_version", "9.16.2.0")

	var paramTests = []struct {
		params      url.Values
		expectation string
	}{
		{testFormAttestation, "passwordattesttesttesttesttestpasswordsnapchat_version9.16.2.0timestamp0123456789usernametest_username"},
		{testFormClientAuth, "noncenonceBase64DataprotobufprotoBase64Datasnapchat_version9.16.2.0"},
	}

	for _, test := range paramTests {
		result := sortURLMap(test.params)
		if result != test.expectation {
			t.Errorf("sortURLMap(%q) failed test. \n\n\rWant: \n\r\"%s\" \n\rGot: \n\r\"%s\" \n\n", test.params, test.expectation, result)
		}
	}
}
