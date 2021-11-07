package main

import (
	"context"
	"log"
	"net/http"

	"git.sr.ht/~lofi/lib"
)

var (
	canSet = map[string]bool{
		"id":   false,
		"jake": false,
		"john": false,
	}
)

func main() {
	l, err := lib.NewLinker(domain, redisDomain)
	if err != nil {
		panic(err)
	}
	defer l.Kill()

	l.AddRoute("/", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Add("Strict-Transport-Security", "max-age=63072000")
		w.Write(rawArt)
	})

	l.AddRoute("/set", func(w http.ResponseWriter, req *http.Request) {
		for key, value := range lib.ParseURI(req.URL) {
			if can(key) {
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

func can(key string) bool {
	if val, exists := canSet[key]; exists {
		return val
	}
	return true
}
