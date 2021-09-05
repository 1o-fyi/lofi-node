package main

import (
	_ "embed"
	"strings"

	"git.sr.ht/~lofi/lib"
)

// All of the configuration files are embedded into the binary
// this means rebuilding to change the configuration; pros and cons without a doubt.
// If you know a better way to do this please reach out.

//go:embed cfg/domain.cfg
var domain string

//go:embed cfg/name.cfg
var name string

//go:embed cfg/apath.cfg
var apath string

//go:embed cfg/mode.cfg
var mode string

//go:embed fs/art/home.b64
var rawArt []byte

func init() {
	rawArt = <-lib.DecodeBase64(rawArt)
	apath = trimDecode(apath)
	name = trimDecode(name)
	domain = trimDecode(domain)
	mode = strings.TrimSpace(mode)
}

func trimDecode(s string) string {
	return string(<-lib.DecodeHex([]byte(strings.TrimSpace(s))))
}
