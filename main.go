package main

import (
  "fmt"
	"log"
  "flag"
  "strings"
	"net/http"
  "encoding/json"
)

var config Config
var Config_Filename string

func ShowInfo(w http.ResponseWriter, r *http.Request) {
  str := "СУКА БЛЯТЬ - bostr следующего поколения\n\n"

  for _, r := range config.Relays {
    str += fmt.Sprintf("- %s\n", r)
  }

  str += fmt.Sprintf("\nПодключитесь к ws://%s\n", r.Host)

  str += "\nPowered by blyat - https://github.com/Yonle/blyat"

  fmt.Fprintf(w, str)
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

  w.Write(d);
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet && r.Header.Get("Upgrade") == "websocket" {
		Accept_Websocket(w, r)
  } else {
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

  fmt.Println("СУКА БЛЯТЬ - bostr следующего поколения\n");
  log.Printf("Чтение %s в текущем каталоге....\n", Config_Filename)

  ReadConfig(Config_Filename, &config)

	go http.HandleFunc("/", handleRequest)

  log.Printf("Прослушивание на %s....\n", config.Listen)

	if err := http.ListenAndServe(config.Listen, nil); err != nil {
   	panic(err)
  }
}
