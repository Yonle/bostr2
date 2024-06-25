package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(_ *http.Request) bool { return true },
}

func Accept_Websocket(w http.ResponseWriter, r *http.Request, ip string, ua string) {
	conn, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		return
	}

	log.Printf("%s connected (%s)", ip, ua)

	var once sync.Once

	var s = Session{
		ClientIP:      ip,
		ClientREQ:     make(MessageChan),
		ClientCLOSE:   make(MessageChan),
		ClientEVENT:   make(MessageChan),
		clientMessage: make(MessageChan),

		upEVENT:   make(MessageChan),
		upEOSE:    make(MessageChan),
		upAdd:     make(WebSocketChan),
		upDel:     make(WebSocketChan),
		UpMessage: make(MessageChan),

		events:        make(SessionEvents),
		pendingEOSE:   make(SessionEOSEs),
		subscriptions: make(SessionSubs),
		relays:        make(SessionRelays),

		destroy:   make(chan struct{}),
		destroyed: make(chan struct{}),
	}

	go func() {
		for msg := range s.UpMessage {
			if err := conn.WriteJSON(msg); err != nil {
				break
			}
		}
	}()

	defer close(s.destroy)
	defer log.Printf("%s disconnect (%s)", ip, ua)

listener:
	for {
		var json []Message
		if err := conn.ReadJSON(&json); err != nil {
			break
		}

		if len(json) < 1 {
			s.UpMessage <- &[]Message{"NOTICE", "error: does not looks like there's something in your message."}
			continue listener
		}

		cmd, ok := json[0].(string)
		if !ok {
			s.UpMessage <- &[]Message{"NOTICE", "error: please check your command."}
			continue listener
		}

		switch cmd {
		case "REQ":
			if len(json) < 3 {
				s.UpMessage <- &[]Message{"NOTICE", "error: invalid REQ"}
				continue listener
			}

			once.Do(s.Start)
			s.ClientREQ <- &json
		case "CLOSE":
			if len(json) < 2 {
				s.UpMessage <- &[]Message{"NOTICE", "error: invalid CLOSE"}
				continue listener
			}

			s.ClientCLOSE <- &json
		case "EVENT":
			if len(json) < 2 {
				s.UpMessage <- &[]Message{"NOTICE", "error: invalid EVENT"}
				continue listener
			}

			once.Do(s.Start)
			s.ClientEVENT <- &json
		default:
			s.UpMessage <- &[]Message{"NOTICE", fmt.Sprintf("error: unknown command %s", cmd)}
		}
	}
}
