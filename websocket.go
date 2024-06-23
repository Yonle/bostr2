package main

import (
	"github.com/gorilla/websocket"
	"net/http"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(_ *http.Request) bool { return true },
}

func Accept_Websocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		return
	}

	var sess = Session{
		Owner:           conn,
		Sub_IDs:         make(SessionSubIDs),
		Event_IDs:       make(SessionEventIDs),
		PendingEOSE:     make(SessionPendingEOSE),
		Relays:          make(SessionRelays),
		UpstreamMessage: make(SessionUpstreamMessage),
		Done:            make(SessionDoneChannel),
	}

	go func() {
		for {
			select {
			case msg := <-sess.UpstreamMessage:
				if err := conn.WriteMessage(websocket.TextMessage, *msg); err != nil {
					return
				}
			case <-sess.Done:
				return
			}
		}
	}()

	defer conn.Close()

	for {
		var json []interface{}
		if err := conn.ReadJSON(&json); err != nil {
			sess.Destroy()
			return
		}

		switch json[0].(string) {
		case "REQ":
			sess.REQ(&json)
		case "CLOSE":
			sess.CLOSE(&json, true)
		case "EVENT":
			sess.EVENT(&json)
		}

	}
}
