# casper
[![Build Status](https://img.shields.io/travis/hako/casper/master.svg?style=flat-square)](https://travis-ci.org/hako/casper)
 [![License](http://img.shields.io/badge/license-MIT-blue.svg?style=flat-square)](https://travis-ci.org/hako/casper)
[![GoDoc](https://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)](https://godoc.org/github.com/hako/casper)
![CasperStatus](https://www.mgp25.com/cstatus/status.svg)

casper is a small golang library for interacting with the Casper API and the Snapchat API.

# Installation
`go get github.com/hako/casper`

## Usage

You would need to register an account on the [Casper API portal](http://developers.casper.io/register.php) in order to use this library. Register an account and comeback to the README.

_Don't worry, I won't disappear in 10 seconds :P_

Once you've registered an account and installed the library, to get started simply create a `Casper{}` struct and enter the following:

+ `APIKey` - your Casper API key.

+ `APISecret` - your Casper API secret.

+ `Username` - your Snapchat username.

+ `Password` - your Snapchat password.

`Debug` is optional and is set to `false` by default.

`ProjectName` is optional and is empty by default.

`AuthToken` is optional but is required for accessing authenticated endpoints.

## Example

```go
package main

import (
	"github.com/hako/casper"
	"fmt"
)

func main() {	
	casperClient := &casper.Casper{
        APIKey:    "yourapikey",
        APISecret: "yourapisecret",
	}
	data, err := casperClient.Login("yoursnapchatusername", "yoursnapchatpassword")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(data) // JSON
}
```

Or if you already have an auth token...

```go
package main

import (
	"github.com/hako/casper"
	"fmt"
)

func main() {	
	casperClient := &casper.Casper{
        APIKey:    "yourapikey",
        APISecret: "yourapisecret",
        Username:  "yoursnapchatusername",
        Password:  "yoursnapchatpassword",
        AuthToken: "yoursnapchatauthtoken",
	}
	data, err := casperClient.Updates()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(data) // JSON
}
```

See the [godoc](https://godoc.org/github.com/hako/casper) for more functions for interacting with the API.
## Todo
- [ ] More tests.
- [ ] Code cleanup.

## Security

This library requires you to have a Snapchat account.

Placing API keys or any hardcoded sensitive information in Git is not recommended or advised!

By using this library you also agree to the [Casper Terms of Use](http://clients.casper.io/terms.php).

## Kudos
+ [liamcottle](http://github.com/liamcottle) - For providing the Casper API service.
+ [Snapchat](http://snapchat.com) - For just being Snapchat.

## Author
Wesley Hill - ([@hako]("github.com/hako")/[@hakobyte]("twitter.com/hakobyte"))

## License
MIT

## Legal
Before using this library, take a look at the [Casper Terms of Use](http://clients.casper.io/terms.php)

Use at your own risk.
