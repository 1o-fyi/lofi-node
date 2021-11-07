package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"log"
	"net/http"

	"git.sr.ht/~lofi/lib"
)

var (
	canSet = map[string]bool{
		"id": false,
	}
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
	resp, err := http.Get(registryUrl)
	if err != nil {
		return nil, err
	}
	sc := bufio.NewScanner(resp.Body)
	registry := map[string]string{}
	for sc.Scan() {
		line := sc.Bytes()
		// ignore commented lines && empty lines
		if len(line) <= 1 || line[0] == '#' {
			continue
		}
		// split on domain seperator (::)
		splitLine := bytes.Split(line, []byte("::"))
		if len(splitLine) != 2 {
			continue
		}
		// update map
		user, pb := splitLine[0], splitLine[1]
		if _, err := l.RC.Set(context.TODO(), string(user), string(pb), 0).Result(); err != nil {
			return nil, errors.New("failed to add user from registry")
		}
		registry[string(user)] = string(pb)
		canSet[string(user)] = false
	}
	return registry, nil
}

func can(key string) bool {
	if val, exists := canSet[key]; exists {
		return val
	}
	return true
}
