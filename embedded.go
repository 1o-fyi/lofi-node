package main

import (
	_ "embed"
	"os"

	"git.sr.ht/~lofi/lib"
)

// All of the configuration files are embedded into the binary
// this means rebuilding to change the configuration; pros and cons without a doubt.
// If you know a better way to do this please reach out.

var (
	// domain is the public facing url of
	// the host. ( e.g foo.com )
	// No need to prefix with https, however,
	// you can specify the port if you don't want
	// to use the default of 443 ( e.g. foo.com:880 )
	//go:embed embedded/cfg/domain.cfg
	domain string

	// redisdomain is the domain where your redis
	// instance can be reached, same rules as above apply.
	//go:embed embedded/cfg/redisDomain.cfg
	redisDomain string

	// the list of registered users from github
	//go:embed embedded/cfg/register.cfg
	registryUrl string

	// the artwork used on the homepage.
	//go:embed embedded/art/home.b64
	rawArt []byte
)

func init() {
	rawArt = <-lib.DecodeBase64(rawArt)
	redisDomain = string(<-lib.DecodeHex([]byte(redisDomain)))
	domain = string(<-lib.DecodeHex([]byte(domain)))
	registryUrl = string(<-lib.DecodeHex([]byte(registryUrl)))

	os.Stdout.Write([]byte("\n[ lofi-node config ]"))
	os.Stdout.Write([]byte("\ndomain: " + domain))
	os.Stdout.Write([]byte("\nredis: " + redisDomain))
	os.Stdout.Write([]byte("\nregistry: " + registryUrl))
	os.Stdout.Write([]byte("\n"))
}
