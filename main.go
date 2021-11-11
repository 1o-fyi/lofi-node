package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"log"
	"net/http"

	"filippo.io/age"
	"git.sr.ht/~lofi/lib"
	bn256 "github.com/ethereum/go-ethereum/crypto/bn256/cloudflare"
)

func main() {

	l, err := lib.NewLinker(domain, redisDomain)
	if err != nil {
		panic(err)
	}
	defer l.Kill()

	registry, err := updateRegistry(l)
	if err != nil {
		panic(err)
	}
	log.Printf("registry size: %d", len(registry))

	l.AddRoute("/", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Add("Strict-Transport-Security", "max-age=63072000")
		w.Write(rawArt)
	})

	l.AddRoute("/set", func(w http.ResponseWriter, req *http.Request) {
		for key, value := range lib.ParseURI(req.URL) {
			if _, ok := registry[key]; can(key) && !ok {
				response, _ := l.RC.Set(context.TODO(), key, value, 0).Result()
				w.Write([]byte(response))
			}
		}
	})

	l.AddRoute("/get", func(w http.ResponseWriter, req *http.Request) {
		for key := range lib.ParseURI(req.URL) {
			response, _ := l.RC.Get(context.TODO(), key).Result()
			w.Write([]byte(response))
		}
	})

	log.Fatal(l.Serve(lib.FmtLetsEncrypt(domain)))

}

// parses the registry of users from github repo
// https://raw.githubusercontent.com/1o-fyi/register/main/REGISTER
//
// this is a simple security measure to prevent unknown users from interacting.
func updateRegistry(l *lib.Linker) (map[string]string, error) {

	// query the registry url & wrap the respone with a scanner
	resp, err := http.Get(registryUrl)
	if err != nil {
		return nil, err
	}
	sc := bufio.NewScanner(resp.Body)

	// this doesn't need to exist, was mainly for testing purposes.
	// going to keep it for now but can likely be removed & nothing should
	// really depend on using this.
	registry := map[string]string{}

	// the default scanner in go will give us a newline
	// every time Scan() is called.
	for sc.Scan() {

		// raw line
		line := sc.Bytes()

		// ignore empty lines && commented lines
		if len(line) <= 1 || line[0] == '#' {
			continue
		}

		// split on domain seperator (::)
		splitLine := bytes.Split(line, []byte("::"))

		// ensure that the result is exactly three distinct parts
		// anything else is considered a malformed line
		if len(splitLine) != 3 {
			log.Println("malfored line did not split into three equal parts", splitLine)
			continue
		}

		// the three pieces of a registry line
		user, pbRaw, g2Raw := splitLine[0], splitLine[1], splitLine[2]

		// ensure that we can parse the age public key
		pbReader := bytes.NewReader(pbRaw)
		_, err := age.ParseRecipients(pbReader)
		if err != nil {
			return nil, errors.New("malformed public age key from registry, failed to parse")
		}

		// ensure that we can parse the G2 point public key
		g2D := lib.Sb(g2Raw).T(lib.DecodeHex).Bytes()
		g2 := &bn256.G2{}
		_, err = g2.Unmarshal(g2D)
		if err != nil {
			return nil, errors.New("malformed G2 point from registry, failed to parse")
		}

		// update the redis cache, we store username -> public key -> pairing curve public key
		// as a linked list that wraps around from head to tail.
		// so given any one of the following: [username, public key, G2 public key]
		// you can query to get the others.
		if _, err := l.RC.Set(context.TODO(), string(user), string(pbRaw), 0).Result(); err != nil {
			return nil, errors.New("failed to add user -> pb mapping from registry")
		}
		if _, err := l.RC.Set(context.TODO(), string(pbRaw), string(g2Raw), 0).Result(); err != nil {
			return nil, errors.New("failed to add pb -> G2 mapping from registry")
		}
		if _, err := l.RC.Set(context.TODO(), string(g2Raw), string(user), 0).Result(); err != nil {
			return nil, errors.New("failed to add G2 -> user mapping from registry")
		}

		// update this debug registry map
		registry[string(user)] = string(pbRaw)
		registry[string(pbRaw)] = string(g2Raw)
		registry[string(g2Raw)] = string(user)
	}

	return registry, nil
}
