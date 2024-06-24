package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"
)

var config Config
var Config_Filename string

func ShowInfo(w http.ResponseWriter, r *http.Request) {
	str := "СУКА БЛЯТЬ - bostr следующего поколения\n\n"

	for _, r := range config.Relays {
		str += fmt.Sprintf("- %s\n", r)
	}

	str += fmt.Sprintf("\nПодключитесь к wss://%s или ws://%s (если не используете TLS)\n", r.Host, r.Host)

	str += "\nPowered by blyat - https://github.com/Yonle/blyat"

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
	flag.StringVar(&Config_Filename, "configfile", "config.yaml", "Путь к файлу конфигурации YAML")
	flag.Parse()

	fmt.Println("СУКА БЛЯТЬ - bostr следующего поколения")
	log.Printf("Чтение %s в текущем каталоге....\n", Config_Filename)

	ReadConfig(Config_Filename, &config)

	http.HandleFunc("/", handleRequest)

	log.Printf("Прослушивание на %s....\n", config.Listen)

	if err := http.ListenAndServe(config.Listen, nil); err != nil {
		panic(err)
	}
}
