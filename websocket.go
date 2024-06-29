package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"

	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"

	"github.com/Yonle/bostr2/relayHandler"
)

func Accept_Websocket(w http.ResponseWriter, r *http.Request, ip string, ua string) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
		CompressionMode:    websocket.CompressionContextTakeover,
	})

	if err != nil {
		return
	}

	defer conn.CloseNow()

	log.Printf("%s connected (%s)", ip, ua)

	ctx, cancel := context.WithCancel(context.Background())

	defer cancel()

	log.Println("let's cook")
	var relaySession = relayHandler.NewSession(ctx)
	log.Println("it cooked.")

	var once sync.Once

	var s = Session{
		ClientIP:      ip,
		ClientREQ:     make(relayHandler.MessageChan),
		ClientCLOSE:   make(relayHandler.MessageChan),
		ClientEVENT:   make(relayHandler.MessageChan),
		clientMessage: make(relayHandler.MessageChan),

		UpMessage: make(relayHandler.MessageChan),

		events:        make(SessionEvents),
		pendingEOSE:   make(SessionEOSEs),
		subscriptions: make(SessionSubs),

		destroyed: make(chan struct{}),

		relay: relaySession,
		ctx:   ctx,
	}

	go func() {
		for msg := range s.UpMessage {
			if err := wsjson.Write(ctx, conn, msg); err != nil {
				break
			}
		}
	}()

	defer log.Printf("%s disconnect (%s)", ip, ua)

listener:
	for {
		var json relayHandler.MessageData
		if err := wsjson.Read(ctx, conn, &json); err != nil {
			break
		}

		if len(json) < 1 {
			s.UpMessage <- &relayHandler.MessageData{"NOTICE", "error: does not looks like there's something in your message."}
			continue listener
		}

		cmd, ok := json[0].(string)
		if !ok {
			s.UpMessage <- &relayHandler.MessageData{"NOTICE", "error: please check your command."}
			continue listener
		}

		switch cmd {
		case "REQ":
			if len(json) < 3 {
				s.UpMessage <- &relayHandler.MessageData{"NOTICE", "error: invalid REQ"}
				continue listener
			}

			once.Do(s.Start)
			//s.ClientREQ <- &json
		case "CLOSE":
			if len(json) < 2 {
				s.UpMessage <- &relayHandler.MessageData{"NOTICE", "error: invalid CLOSE"}
				continue listener
			}

			s.ClientCLOSE <- &json
		case "EVENT":
			if len(json) < 2 {
				s.UpMessage <- &relayHandler.MessageData{"NOTICE", "error: invalid EVENT"}
				continue listener
			}

			once.Do(s.Start)
			s.ClientEVENT <- &json
		default:
			s.UpMessage <- &relayHandler.MessageData{"NOTICE", fmt.Sprintf("error: unknown command %s", cmd)}
		}
	}
}
