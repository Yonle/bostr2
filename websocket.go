package main

import (
	"context"
	"log"
	"net/http"

	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

func Accept_Websocket(w http.ResponseWriter, r *http.Request, ip string, ua string) {
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
		CompressionMode:    websocket.CompressionContextTakeover,
	})

	if err != nil {
		return
	}

	defer c.CloseNow()

	ctx, cancel := context.WithCancel(context.Background())

	defer cancel()

	log.Printf("%s связанный (%s)", ip, ua)

	var s = &Session{
		ClientIP:    ip,
		ClientREQ:   make(MessageChan),
		ClientCLOSE: make(MessageChan),
		ClientEVENT: make(MessageChan),
		UpMessage:     make(MessageChan),
	}

	go func() {
		for msg := range s.UpMessage {
			if err := wsjson.Write(ctx, c, msg); err != nil {
				break
			}
		}
	}()

	defer s.Destroy()
	defer log.Printf("%s отключен (%s)", ip, ua)
	defer c.Close(websocket.StatusUnsupportedData, "Данные не в формате JSON")

	for {
		var json []Message
		if err := wsjson.Read(ctx, c, &json); err != nil {
			break
		}

		switch json[0].(string) {
		case "REQ":
			if len(json) < 3 {
				s.UpMessage <- &[]Message{"NOTICE", "error: invalid request"}
			}

			s.ClientREQ <- &json
		case "CLOSE":
			s.ClientCLOSE <- &json
		case "EVENT":
			s.ClientEVENT <- &json
		}
	}
}
