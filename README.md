# casper
casper is a small golang library for interacting with the Casper API.

# Installation
`go get github.com/hako/casper`

## Usage

You would need to signup to [Casper API portal](http://clients.casper.io) in order to use this library. Signup and comeback to the README.

_I won't disappear in 10 seconds :P_

Replace `"yourapikey"` with your Casper API Key.

Replace `"youapisecret"` with your Casper API Secret.

And so on.

```go
package main

import (
	"github.com/hako/casper"
)

func main() {
	casperClient := &casper.Casper{
		ApiKey:          "yourapikey",
		ApiSecret:       "youapisecret",
		Username:        "yoursnapchatusername",
		GoogleMail:      "yourgmailaccount",
		GooglePassword:  "yourgmailaccountpassword",
		Debug:            false,
	}
	// Call any *casper.Casper methods.
}
```
## Todo
- [ ] More tests.
- [ ] Clean up code.

## Security

As you can see from the example above, using this library requires a gmail account since this API connects to Google to request tokens linked to your Google account.

Placing API keys or any hardcoded sensitive information in git is not recommended or advised!

Either use environment variables or use a throwaway gmail account.

By using this library you also agree to the [Casper Terms of Use](http://clients.casper.io/terms.php).

## Kudos
+ [liamcottle](http://github.com/liamcottle) - For providing the Casper API service.
+ [Snapchat](http://snapchat.com) - For just being Snapchat.

## License
MIT

## Author
Wesley Hill - ([@hako]("github.com/hako")/[@hakobyte]("twitter.com/hakobyte"))

## Legal
Before using this library, take a look at the [Casper Terms of Use](http://clients.casper.io/terms.php)

Use at your own risk.