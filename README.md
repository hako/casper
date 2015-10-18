# casper
[![Build Status](https://img.shields.io/travis/hako/casper/master.svg?style=flat-square)](https://travis-ci.org/hako/casper)
 [![License](http://img.shields.io/badge/license-MIT-blue.svg?style=flat-square)](https://travis-ci.org/hako/casper)
[![GoDoc](https://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)](https://godoc.org/github.com/hako/casper)


casper is a small golang library for interacting with the Casper API.

# Installation
`go get github.com/hako/casper`

## Usage

You would need to register an account on the [Casper API portal](https://clients.casper.io/register.php) in order to use this library. Register an account and comeback to the README.

_I won't disappear in 10 seconds :P_

Once you've registered an account and installed the library, to get started simply create a `Casper{}` struct and enter the following:

+ `APIKey` - your Casper API key.

+ `APISecret` - your Casper API secret.

+ `Username` - your Snapchat username.

+ `Password` - your Snapchat password.

`Debug` is optional and is set to false by default.

## Example

```go
package main

import (
	"fmt"
	"strconv"
	"time"
	
	"github.com/hako/casper"
)

func main() {
	casperClient := &casper.Casper{
		APIKey:    "yourapikey",
		APISecret: "yourapisecret",
		Username:  "yoursnapchatusername",
		Password:  "yoursnapchatpassword",
		Debug:     false,
	}
	timestamp := strconv.Itoa(int(time.Now().UnixNano() / 1000000))
	
	// Call any *casper.Casper methods
	attestation, err := casperClient.GetAttestation(casperClient.Username, casperClient.Password, timestamp)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Attestation: " + attestation)
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
