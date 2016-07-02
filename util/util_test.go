package util

import (
	"net/url"
	"testing"
)

// Test SortURLMap.
func TestSortURLMap(t *testing.T) {
	testFormAttestation := url.Values{}
	testFormAttestation.Add("username", "test_username")
	testFormAttestation.Add("password", "attesttesttesttesttestpassword")
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
