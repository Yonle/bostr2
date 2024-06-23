package main

import (
	"context"
	"net/http"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

func Accept_Websocket(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})

	if err != nil {
		return
	}

	defer c.CloseNow()

	ctx, cancel := context.WithCancel(context.Background())

	defer cancel()

	var sess = Session{
		Sub_IDs:     make(SessionSubIDs),
		Event_IDs:   make(SessionEventIDs),
		PendingEOSE: make(SessionPendingEOSE),
		Relays:      make(SessionRelays),
		CancelZone:  make(SessionRelayCancelContext),

		UpstreamMessage: make(SessionUpstreamMessage),
		Done:            make(SessionDoneChannel),
	}

	go func() {
	listener:
		for {
			select {
			case msg := <-sess.UpstreamMessage:
				if err := wsjson.Write(ctx, c, *msg); err != nil {
					break listener
				}
			case <-sess.Done:
				break listener
			}
		}
	}()

	defer sess.Destroy()
	defer c.Close(websocket.StatusUnsupportedData, "Данные не в формате JSON")

	for {
		var json []interface{}
		if err := wsjson.Read(ctx, c, &json); err != nil {
			break
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
