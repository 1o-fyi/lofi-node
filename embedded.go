package main

import (
	_ "embed"
	"strings"

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
	// go:embed embedded/cfg/domain.cfg
	domain string

	// redisdomain is the domain where your redis
	// instance can be reached, same rules as above apply.
	//go:embed embedded/cfg/redisDomain.cfg
	redisDomain string

	//go:embed embedded/art/home.b64
	rawArt []byte
)

func init() {
	rawArt = <-lib.DecodeBase64(rawArt)
	redisDomain = trimDecode(redisDomain)
	domain = trimDecode(domain)
}

func trimDecode(s string) string {
	return string(<-lib.DecodeHex([]byte(strings.TrimSpace(s))))
}
