package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
)

var config Config
var Config_Filename string
var FaviconBytes []byte

func ShowInfo(w http.ResponseWriter, r *http.Request) {
	str := "bostr2 - bostr next generation\n\n"

	for _, r := range config.Relays {
		str += fmt.Sprintf("- %s\n", r)
	}

	str += fmt.Sprintf("\nConnect to wss://%s or ws://%s (if not using TLS)\n", r.Host, r.Host)

	str += "\nPowered by bostr2 - https://github.com/Yonle/bostr2"

	fmt.Fprint(w, str)
}

func ShowNIP11(w http.ResponseWriter) {
	d, err := json.Marshal(config.NIP_11)

	var header = w.Header()
	header.Set("Content-Type", "application/json")
	header.Set("Access-Control-Allow-Origin", "*")

	if err != nil {
		fmt.Fprintf(w, "{}")
		return
	}

	w.Write(d)
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	xff := r.Header.Get("X-Forwarded-For")
	ua := r.Header.Get("User-Agent")
	ip := strings.Split(xff, ",")[0]

	if len(ip) < 1 {
		ip, _, _ = net.SplitHostPort(r.RemoteAddr)
	}

	if r.Method == http.MethodGet && r.Header.Get("Upgrade") == "websocket" {
		if pass := ConnPerIPRateLimit_Pass(ip, w); !pass {
			return
		}

		Accept_Websocket(w, r, ip, ua)
	} else {
		log.Println(ip, r.Method, r.URL, ua)

		// HUH??
		if len(r.Header.Get("Upgrade")) > 0 {
			http.Error(w, "Invalid Upgrade Header", 400)
			return
		}

		accept := r.Header.Get("Accept")
		if strings.Contains(accept, "application/nostr+json") {
			ShowNIP11(w)
		} else {
			ShowInfo(w, r)
		}
	}
}

func handleFavicon(w http.ResponseWriter, r *http.Request) {
	xff := r.Header.Get("X-Forwarded-For")
	ua := r.Header.Get("User-Agent")
	ip := strings.Split(xff, ",")[0]

	log.Println(ip, r.Method, r.URL, ua)

	var header = w.Header()
	header.Set("Content-Type", "image/vnd.microsoft.icon")
	header.Set("Access-Control-Allow-Origin", "*")

	w.Write(FaviconBytes)
}

func LoadFavicon() {
	if len(config.Favicon) < 1 {
		return
	}

	bytes, err := os.ReadFile(config.Favicon)

	if err != nil {
		log.Printf("Failed to load favicon (%s) into memory: %v", config.Favicon, err)
		return
	}

	FaviconBytes = bytes
	log.Printf("Loaded favicon (%s) into memory", config.Favicon)
}

func LoadConfig() {
	log.Printf("Reading config file %s....\n", Config_Filename)

	ReadConfig(Config_Filename, &config)
}

func Serve() {
	http.HandleFunc("/favicon.ico", handleFavicon)
	http.HandleFunc("/", handleRequest)

	log.Printf("Listening on %s....\n", config.Listen)

	if err := http.ListenAndServe(config.Listen, nil); err != nil {
		panic(err)
	}
}
