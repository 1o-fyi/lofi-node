package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"filippo.io/age"
	"git.sr.ht/~lofi/lib"
	bn256 "github.com/ethereum/go-ethereum/crypto/bn256/cloudflare"
	"github.com/keep-network/keep-core/pkg/bls"
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
	log.Printf("registry size: %d", len(registry)/3)

	l.AddRoute("/", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Add("Strict-Transport-Security", "max-age=7200")
		w.Write(rawArt)
	})

	l.AddRoute("/ip", func(rw http.ResponseWriter, r *http.Request) {
		rw.Write([]byte(r.RemoteAddr))
		rw.WriteHeader(200)
	})

	l.AddRoute("/body", func(rw http.ResponseWriter, r *http.Request) {
		io.Copy(rw, r.Body)
		rw.WriteHeader(200)
	})

	l.AddRoute("/header", func(rw http.ResponseWriter, r *http.Request) {
		for k, v := range r.Header {
			strings.Join([]string{k, strings.Join(v, ", ")}, ":")
		}
		rw.WriteHeader(200)
	})

	l.AddRoute("/ua", func(rw http.ResponseWriter, r *http.Request) {
		rw.Write([]byte(r.UserAgent()))
		rw.WriteHeader(200)
	})

	l.AddRoute("/cs", func(rw http.ResponseWriter, r *http.Request) {
		rw.Write([]byte(fmt.Sprintf("%d", r.TLS.CipherSuite)))
		rw.WriteHeader(200)
	})

	l.AddRoute("/peers", func(rw http.ResponseWriter, r *http.Request) {
		for _, cert := range r.TLS.PeerCertificates {
			rw.Write(cert.Raw)
		}
		rw.WriteHeader(200)
	})

	// TLSUnique contains the "tls-unique" channel binding value (see RFC 5929, Section 3). This value will be nil for TLS 1.3 connections and for all resumed connections.
	// There are conditions in which this value might not be unique to a connection. See the Security Considerations sections of RFC 5705 and RFC 7627, and https://mitls.org/pages/attacks/3SHAKE#channelbindings.
	l.AddRoute("/tu", func(rw http.ResponseWriter, r *http.Request) {
		if r.TLS.TLSUnique == nil {
			rw.Write([]byte("nil"))
		} else {
			rw.Write(r.TLS.TLSUnique)
		}
		rw.WriteHeader(200)
	})

	l.AddRoute("/proto", func(rw http.ResponseWriter, r *http.Request) {
		rw.Write([]byte(r.TLS.NegotiatedProtocol))
		rw.WriteHeader(200)
	})

	l.AddRoute("/mutual", func(rw http.ResponseWriter, r *http.Request) {
		rw.Write([]byte(fmt.Sprintf("%v", r.TLS.NegotiatedProtocolIsMutual)))
		rw.WriteHeader(200)
	})

	l.AddRoute("/chains", func(rw http.ResponseWriter, r *http.Request) {
		for _id, chain := range r.TLS.VerifiedChains {
			rw.Write([]byte(fmt.Sprintf("CHAIN: %d", _id)))
			for _, item := range chain {
				rw.Write(append(item.Raw, []byte("\n")...))
			}
		}
		rw.WriteHeader(200)
	})

	l.AddRoute("/raw", func(rw http.ResponseWriter, r *http.Request) {
		raw := fmt.Sprintf("%+v", r)
		p, err := json.Marshal(raw)
		if err != nil {
			rw.Write([]byte(raw))
		} else {
			rw.Write(p)
		}
		rw.WriteHeader(200)
	})

	// no requests to /set will give a response to the client.
	// this means failure and success should be indistinguishable from each other.
	// This is in an effort to prevent information leakage via timing attacks
	// or sidechannel analysis.
	l.AddRoute("/set", func(w http.ResponseWriter, req *http.Request) {

		uriMap := lib.ParseURI(req.URL)
		// plaintext username (ie; "john")
		senderUsername, uExists := uriMap["user"]
		// message signature
		senderSignatur, sExists := uriMap["sign"]
		// key (eg, message id) from the sender
		senderMid, midExists := uriMap["mid"]
		// senders msg content
		senderMsg, msgExists := uriMap["msg"]

		sout(senderUsername, senderSignatur, senderMid, senderMsg)

		// if the requests did not contain any of the following
		// uri parameters then just return early and give no response
		if lib.Any(!uExists, !sExists, !midExists, !msgExists) {
			return
		}

		// if we can't unmarshal the signature, return early (no response).
		g1, err := unmarshalG1([]byte(senderSignatur))
		if err != nil {
			return
		}

		// if we can't find the username in the register, return early (no response).
		agePb, err := l.RC.Get(context.TODO(), senderUsername).Result()
		if err != nil {
			return
		}

		// if we can't find the G2 point in the register, return early (ie; no response).
		g2Str, err := l.RC.Get(context.TODO(), agePb).Result()
		if err != nil {
			return
		}

		// if we can't unmarshal the G2 public key, return early (no respone).
		g2, err := unmarshalG2([]byte(g2Str))
		if err != nil {
			return
		}

		// if invalid signature, return early (no response).
		if !bls.Verify(g2, <-lib.DecodeHex([]byte(senderMsg)), g1) {
			return
		}

		// success! update redis with the message
		_, err = l.RC.Set(context.TODO(), senderMid, senderMsg, 0).Result()
		if err != nil {
			return
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
		_, err = unmarshalG2(g2Raw)
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

func unmarshalG2(g2Raw []byte) (*bn256.G2, error) {
	var err error
	g2D := lib.Sb(g2Raw).T(lib.DecodeHex).Bytes()
	g2 := &bn256.G2{}
	_, err = g2.Unmarshal(g2D)
	if err != nil {
		return nil, errors.New("malformed G2 point from registry, failed to parse")
	}
	return g2, nil
}

func unmarshalG1(g1Raw []byte) (*bn256.G1, error) {
	var err error
	g1D := lib.Sb(g1Raw).T(lib.DecodeHex).Bytes()
	g1 := &bn256.G1{}
	_, err = g1.Unmarshal(g1D)
	if err != nil {
		return nil, errors.New("malformed G2 point from registry, failed to parse")
	}
	return g1, nil
}

func out(b ...[]byte) {
	os.Stdout.Write([]byte("\n"))
	for _, r := range b {
		os.Stdout.Write(r)
		os.Stdout.Write([]byte("\n"))
	}
}

func sout(b ...string) {
	for _, r := range b {
		out([]byte(r))
	}
}
