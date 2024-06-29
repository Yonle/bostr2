package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"strings"
)

var config Config
var Config_Filename string

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
		ip = r.RemoteAddr
	}

	if r.Method == http.MethodGet && r.Header.Get("Upgrade") == "websocket" {
		Accept_Websocket(w, r, ip, ua)
	} else {
		log.Println(ip, r.Method, r.URL, ua)
		accept := r.Header.Get("Accept")
		if strings.Contains(accept, "application/nostr+json") {
			ShowNIP11(w)
		} else {
			ShowInfo(w, r)
		}
	}
}

func main() {
	flag.StringVar(&Config_Filename, "configfile", "config.yaml", "Path to YAML config file")
	flag.Parse()

	fmt.Println("bostr2 - bostr next generation")
	log.Printf("Reading config file %s....\n", Config_Filename)

	ReadConfig(Config_Filename, &config)

	http.HandleFunc("/", handleRequest)

	log.Printf("Listening on %s....\n", config.Listen)

	if err := http.ListenAndServe(config.Listen, nil); err != nil {
		panic(err)
	}
}
